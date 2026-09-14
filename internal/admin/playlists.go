package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/scc-systems/wallsync/internal/playlist"
	"github.com/scc-systems/wallsync/internal/scene"
)

type playlistRequest struct {
	Name  string          `json:"name"`
	Loop  bool            `json:"loop"`
	Items []playlist.Item `json:"items"`
}

func (a *Admin) ServeListPlaylists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	items := a.Playlists.List()
	if items == nil {
		items = []playlist.Playlist{}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"playlists": items})
}

func (a *Admin) ServeCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	var req playlistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := a.validateItems(req.Items); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	p, err := a.Playlists.Create(req.Name, req.Loop, req.Items)
	if err != nil {
		if errors.Is(err, playlist.ErrEmptyName) {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		if errors.Is(err, playlist.ErrInvalidItem) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		a.Logger.Error("playlist create failed", "err", err)
		http.Error(w, "create failed", http.StatusInternalServerError)
		return
	}
	a.Logger.Info("playlist created", "id", p.ID, "name", p.Name, "items", len(p.Items))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(p)
}

func (a *Admin) ServeUpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req playlistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := a.validateItems(req.Items); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	p, err := a.Playlists.Update(id, req.Name, req.Loop, req.Items)
	switch {
	case errors.Is(err, playlist.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
		return
	case errors.Is(err, playlist.ErrInvalidID):
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	case errors.Is(err, playlist.ErrEmptyName):
		http.Error(w, "name required", http.StatusBadRequest)
		return
	case errors.Is(err, playlist.ErrInvalidItem):
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	case err != nil:
		a.Logger.Error("playlist update failed", "err", err, "id", id)
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	a.Logger.Info("playlist updated", "id", id, "name", p.Name, "items", len(p.Items))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(p)
}

func (a *Admin) ServeDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.Playlists.Delete(id)
	switch {
	case errors.Is(err, playlist.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
		return
	case errors.Is(err, playlist.ErrInvalidID):
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	case err != nil:
		a.Logger.Error("playlist delete failed", "err", err, "id", id)
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	a.Logger.Info("playlist deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

// validateItems checks each queue/playlist item beyond structural validation:
// referenced media must exist; scene ids must be valid; stretched/span mode
// is disallowed at the API level to match the UI's greyed-out button.
func (a *Admin) validateItems(items []playlist.Item) error {
	for i, it := range items {
		if err := playlist.ValidateItem(it); err != nil {
			return numberedErr(i, err)
		}
		switch it.Kind {
		case playlist.KindScene:
			if !scene.Valid(it.Scene) {
				return numberedErr(i, errors.New("unknown scene"))
			}
		case playlist.KindVideo, playlist.KindImage:
			if _, ok := a.Media.Get(it.RefID); !ok {
				return numberedErr(i, errors.New("media not found"))
			}
			if it.Mode == "stretch" || it.Mode == "span" {
				return numberedErr(i, errors.New("stretched/span mode disabled"))
			}
		}
	}
	return nil
}

func numberedErr(i int, err error) error {
	return fmt.Errorf("item %d: %w", i, err)
}
