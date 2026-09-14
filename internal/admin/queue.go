package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/scc-systems/wallsync/internal/playback"
	"github.com/scc-systems/wallsync/internal/playlist"
)

func (a *Admin) ServeQueueSnapshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	snap := a.Playback.Snapshot()
	_ = json.NewEncoder(w).Encode(snap)
}

type queueLoadRequest struct {
	PlaylistID string `json:"playlist_id"`
}

func (a *Admin) ServeQueueLoad(w http.ResponseWriter, r *http.Request) {
	var req queueLoadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	p, ok := a.Playlists.Get(req.PlaylistID)
	if !ok {
		http.Error(w, "playlist not found", http.StatusNotFound)
		return
	}
	a.Playback.Load(p.Items, p.Loop)
	a.Logger.Info("queue loaded", "playlist_id", p.ID, "name", p.Name, "items", len(p.Items))
	writeQueueSnapshot(w, a.Playback)
}

type queueItemRequest struct {
	Item     playlist.Item `json:"item"`
	Position *int          `json:"position,omitempty"`
}

func (a *Admin) ServeQueueAddItem(w http.ResponseWriter, r *http.Request) {
	var req queueItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := a.validateItems([]playlist.Item{req.Item}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var err error
	if req.Position != nil {
		err = a.Playback.Insert(req.Item, *req.Position)
	} else {
		err = a.Playback.Append(req.Item)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.Logger.Info("queue item added", "kind", req.Item.Kind, "ref", req.Item.RefID, "scene", req.Item.Scene)
	writeQueueSnapshot(w, a.Playback)
}

func (a *Admin) ServeQueueRemoveItem(w http.ResponseWriter, r *http.Request) {
	idx, ok := indexParam(r.PathValue("index"))
	if !ok {
		http.Error(w, "invalid index", http.StatusBadRequest)
		return
	}
	err := a.Playback.Remove(idx)
	switch {
	case errors.Is(err, playback.ErrOutOfRange):
		http.Error(w, "out of range", http.StatusBadRequest)
		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeQueueSnapshot(w, a.Playback)
}

type queueReorderRequest struct {
	From int `json:"from"`
	To   int `json:"to"`
}

func (a *Admin) ServeQueueReorder(w http.ResponseWriter, r *http.Request) {
	var req queueReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := a.Playback.Reorder(req.From, req.To); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeQueueSnapshot(w, a.Playback)
}

func (a *Admin) ServeQueuePlay(w http.ResponseWriter, r *http.Request) {
	if err := a.Playback.Play(); err != nil {
		if errors.Is(err, playback.ErrEmptyQueue) {
			http.Error(w, "queue empty", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.Logger.Info("queue play")
	writeQueueSnapshot(w, a.Playback)
}

func (a *Admin) ServeQueuePause(w http.ResponseWriter, r *http.Request) {
	a.Playback.Pause()
	a.Logger.Info("queue pause")
	writeQueueSnapshot(w, a.Playback)
}

func (a *Admin) ServeQueueSkip(w http.ResponseWriter, r *http.Request) {
	if err := a.Playback.Skip(); err != nil {
		if errors.Is(err, playback.ErrEmptyQueue) {
			http.Error(w, "queue empty", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.Logger.Info("queue skip")
	writeQueueSnapshot(w, a.Playback)
}

func (a *Admin) ServeQueueStop(w http.ResponseWriter, r *http.Request) {
	a.Playback.Stop()
	a.Logger.Info("queue stop")
	writeQueueSnapshot(w, a.Playback)
}

type queueSaveAsRequest struct {
	Name string `json:"name"`
	Loop bool   `json:"loop"`
}

func (a *Admin) ServeQueueSaveAs(w http.ResponseWriter, r *http.Request) {
	var req queueSaveAsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	snap := a.Playback.Snapshot()
	items := make([]playlist.Item, 0, len(snap.Items))
	for _, it := range snap.Items {
		items = append(items, playlist.Item{
			Kind:      playlist.ItemKind(it.Kind),
			RefID:     it.RefID,
			Scene:     it.Scene,
			Mode:      it.Mode,
			DurationS: it.DurationS,
		})
	}
	p, err := a.Playlists.Create(req.Name, req.Loop, items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.Logger.Info("queue saved as playlist", "id", p.ID, "name", p.Name)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(p)
}

func writeQueueSnapshot(w http.ResponseWriter, p *playback.Player) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(p.Snapshot())
}

func indexParam(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}
