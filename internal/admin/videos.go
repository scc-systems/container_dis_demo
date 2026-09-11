package admin

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/eleanor/wallsync/internal/media"
	"github.com/eleanor/wallsync/internal/scene"
)

func (a *Admin) ServeListVideos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	var videos []media.VideoMeta
	for _, m := range a.Media.List() {
		switch m.Ext {
		case ".mp4", ".webm", ".mov":
			videos = append(videos, m)
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"videos": videos})
}

func (a *Admin) ServeUploadVideo(w http.ResponseWriter, r *http.Request) {
	if a.MaxUploadBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, a.MaxUploadBytes)
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "upload too large or malformed: "+err.Error(), http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("video")
	if err != nil {
		http.Error(w, "missing 'video' form field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	meta, err := a.Media.Save(header.Filename, contentType, header.Size, file)
	if err != nil {
		if errors.Is(err, media.ErrUnsupportedFormat) {
			http.Error(w, "unsupported format (allow .mp4, .webm, .mov)", http.StatusUnsupportedMediaType)
			return
		}
		a.Logger.Error("video save failed", "err", err, "filename", header.Filename)
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}
	a.Logger.Info("video uploaded", "id", meta.ID, "filename", meta.Filename, "size", meta.SizeBytes)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(meta)
}

func (a *Admin) ServeDeleteVideo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.Media.Delete(id)
	switch {
	case errors.Is(err, media.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
		return
	case errors.Is(err, media.ErrInvalidID):
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	case err != nil:
		a.Logger.Error("video delete failed", "err", err, "id", id)
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	a.Logger.Info("video deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

type playRequest struct {
	VideoID string `json:"video_id"`
	Mode    string `json:"mode"`
}

func (a *Admin) ServePlayVideo(w http.ResponseWriter, r *http.Request) {
	var req playRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	meta, ok := a.Media.Get(req.VideoID)
	if !ok {
		http.Error(w, "video not found", http.StatusNotFound)
		return
	}
	mode := req.Mode
	if mode != "stretch" {
		mode = "mirror"
	}
	epoch := a.Director.Switch(scene.SceneVideo, map[string]any{
		"url":      meta.URLPath,
		"mode":     mode,
		"video_id": meta.ID,
		"filename": meta.Filename,
	})
	a.Logger.Info("video play", "id", meta.ID, "mode", mode, "epoch_ms", epoch)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"scene":    scene.SceneVideo,
		"epoch_ms": epoch,
		"video":    meta,
		"mode":     mode,
	})
}

func (a *Admin) ServeStopVideo(w http.ResponseWriter, r *http.Request) {
	epoch := a.Director.Switch(scene.SceneThreeUnified, nil)
	a.Logger.Info("video stop", "return_scene", scene.SceneThreeUnified, "epoch_ms", epoch)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"scene":    scene.SceneThreeUnified,
		"epoch_ms": epoch,
	})
}
