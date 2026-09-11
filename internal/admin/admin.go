package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/scc-systems/wallsync/internal/hub"
	"github.com/scc-systems/wallsync/internal/media"
	"github.com/scc-systems/wallsync/internal/protocol"
	"github.com/scc-systems/wallsync/internal/scene"
	"github.com/scc-systems/wallsync/internal/timing"
)

type Admin struct {
	Hub            *hub.Hub
	Director       *scene.Director
	Logger         *slog.Logger
	Media          *media.Store
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
	return protocol.AdminStatusPayload{
		Scene:      snap.Scene,
		EpochMs:    snap.EpochMs,
		NowMs:      nowMs,
		AutoRotate: a.Director.AutoRotating(),
		Nodes:      nodes,
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
