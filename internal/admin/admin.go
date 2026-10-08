package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/scc-systems/wallsync/internal/hub"
	"github.com/scc-systems/wallsync/internal/media"
	"github.com/scc-systems/wallsync/internal/playback"
	"github.com/scc-systems/wallsync/internal/playlist"
	"github.com/scc-systems/wallsync/internal/protocol"
	"github.com/scc-systems/wallsync/internal/scene"
	"github.com/scc-systems/wallsync/internal/timing"
)

type Admin struct {
	Hub            *hub.Hub
	Director       *scene.Director
	Logger         *slog.Logger
	Media          *media.Store
	Playlists      *playlist.Store
	Playback       *playback.Player
	MaxUploadBytes int64
	BasePath       string
}

func (a *Admin) StatusPayload() protocol.AdminStatusPayload {
	snap := a.Director.Snapshot()
	now := time.Now()
	nowMs := now.UnixMilli()
	regSnap := a.Hub.Registry.Snapshot()
	nodes := make([]protocol.AdminNodeStatus, 0, len(regSnap))
	for i, ns := range regSnap {
		if ns == nil {
			nodes = append(nodes, protocol.AdminNodeStatus{ID: i, Connected: false})
			continue
		}
		nodes = append(nodes, protocol.AdminNodeStatus{
			ID:          ns.ID,
			Connected:   ns.Connected,
			RTTMedianMs: ns.RTTStats.Median(),
			RTTP95Ms:      ns.RTTStats.P95(),
			OffsetMs:      ns.OffsetStats.Median(),
			LastPongMsAgo: now.Sub(ns.LastPongAt).Milliseconds(),
			Drops:         ns.Drops.Load(),
			ConnectedAtMs: ns.ConnectedAt.UnixMilli(),
		})
	}
	var queue *protocol.QueueSnapshot
	if a.Playback != nil {
		q := a.Playback.Snapshot()
		queue = &q
	}
	return protocol.AdminStatusPayload{
		Scene:      snap.Scene,
		Playing:    playingLabel(snap.Scene, snap.Params),
		EpochMs:    snap.EpochMs,
		NowMs:      nowMs,
		AutoRotate: a.Director.AutoRotating(),
		Nodes:      nodes,
		Queue:      queue,
	}
}

func playingLabel(sceneID string, params map[string]any) string {
	switch sceneID {
	case scene.SceneIdle:
		return "idle"
	case scene.SceneVideo, scene.SceneImage:
		filename, _ := params["filename"].(string)
		mode, _ := params["mode"].(string)
		if filename == "" {
			return sceneID
		}
		if mode == "" {
			return filename
		}
		return filename + " (" + mode + ")"
	default:
		return sceneID
	}
}

func (a *Admin) ServeToggleAutorotate(w http.ResponseWriter, r *http.Request) {
	enabled := a.Director.ToggleAutoRotate()
	a.Logger.Info("autorotate toggled", "enabled", enabled)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"auto_rotate": enabled})
}

func (a *Admin) ServeStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(a.StatusPayload())
}

type sceneChangeRequest struct {
	Scene string `json:"scene"`
}

// ServeRestart re-broadcasts the current scene with its existing params so the
// wall reseeks (for video/image) or reseeds (for procedural scenes). A no-op
// when idle — the operator has nothing to restart.
func (a *Admin) ServeRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	snap := a.Director.Snapshot()
	if snap.Scene == scene.SceneIdle {
		http.Error(w, "nothing playing to restart", http.StatusBadRequest)
		return
	}
	epoch := a.Director.Switch(snap.Scene, snap.Params)
	a.Logger.Info("restart", "scene", snap.Scene, "epoch_ms", epoch)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"scene":    snap.Scene,
		"epoch_ms": epoch,
	})
}

func (a *Admin) ServeSceneSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req sceneChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if !scene.Valid(req.Scene) {
		http.Error(w, "unknown scene", http.StatusBadRequest)
		return
	}
	epoch := a.Director.Switch(req.Scene, nil)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"scene":    req.Scene,
		"epoch_ms": epoch,
	})
}

// ServeExit flushes a success response and then exits the process with code 0,
// relying on Docker's restart policy to bring wallsync back up.
func (a *Admin) ServeExit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	a.Logger.Info("admin requested exit; relying on docker restart policy")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	go func() {
		time.Sleep(200 * time.Millisecond)
		os.Exit(0)
	}()
}

func (a *Admin) ServeAdminWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer conn.Close(websocket.StatusNormalClosure, "bye")

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			payload := a.StatusPayload()
			raw, err := protocol.Marshal(protocol.TypeAdminStatus, 0, timing.EpochNowMs(), payload)
			if err != nil {
				continue
			}
			writeCtx, wcancel := context.WithTimeout(ctx, 2*time.Second)
			err = conn.Write(writeCtx, websocket.MessageText, raw)
			wcancel()
			if err != nil {
				return
			}
		}
	}
}
