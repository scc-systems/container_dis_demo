package admin

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/scc-systems/wallsync/internal/media"
	"github.com/scc-systems/wallsync/internal/scene"
)

func (a *Admin) ServeListImages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	var images []media.VideoMeta
	for _, m := range a.Media.List() {
		switch m.Ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif":
			images = append(images, m)
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"images": images})
}

func (a *Admin) ServeUploadImage(w http.ResponseWriter, r *http.Request) {
	if a.MaxUploadBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, a.MaxUploadBytes)
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "upload too large or malformed: "+err.Error(), http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "missing 'image' form field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	meta, err := a.Media.Save(header.Filename, contentType, header.Size, file)
	if err != nil {
		if errors.Is(err, media.ErrUnsupportedFormat) {
			http.Error(w, "unsupported format (allow .jpg, .png, .webp, .gif)", http.StatusUnsupportedMediaType)
			return
		}
		a.Logger.Error("image save failed", "err", err, "filename", header.Filename)
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}
	a.Logger.Info("image uploaded", "id", meta.ID, "filename", meta.Filename, "size", meta.SizeBytes)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(meta)
}

func (a *Admin) ServeDeleteImage(w http.ResponseWriter, r *http.Request) {
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
		a.Logger.Error("image delete failed", "err", err, "id", id)
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	a.Logger.Info("image deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

type playImageRequest struct {
	ImageID string `json:"image_id"`
	Mode    string `json:"mode"`
}

func (a *Admin) ServePlayImage(w http.ResponseWriter, r *http.Request) {
	var req playImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	meta, ok := a.Media.Get(req.ImageID)
	if !ok {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}
	mode := req.Mode
	if mode != "mirror" {
		mode = "span"
	}
	epoch := a.Director.Switch(scene.SceneImage, map[string]any{
		"url":      a.BasePath + meta.URLPath,
		"mode":     mode,
		"image_id": meta.ID,
		"filename": meta.Filename,
	})
	a.Logger.Info("image play", "id", meta.ID, "mode", mode, "epoch_ms", epoch)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"scene":    scene.SceneImage,
		"epoch_ms": epoch,
		"image":    meta,
		"mode":     mode,
	})
}

func (a *Admin) ServeStopImage(w http.ResponseWriter, r *http.Request) {
	epoch := a.Director.Switch(scene.SceneThreeUnified, nil)
	a.Logger.Info("image stop", "return_scene", scene.SceneThreeUnified, "epoch_ms", epoch)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"scene":    scene.SceneThreeUnified,
		"epoch_ms": epoch,
	})
}
