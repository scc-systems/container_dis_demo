// Package media stores uploaded videos on the local filesystem and exposes
// enough metadata for the admin UI and the /media/ file server.
package media

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrUnsupportedFormat = errors.New("media: unsupported file format")
	ErrNotFound          = errors.New("media: not found")
	ErrInvalidID         = errors.New("media: invalid id")
)

var allowedExts = map[string]struct{}{
	".mp4":  {},
	".webm": {},
	".mov":  {},
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".webp": {},
	".gif":  {},
}

type VideoMeta struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	Ext         string    `json:"ext"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	UploadedAt  time.Time `json:"uploaded_at"`
	URLPath     string    `json:"url_path"`
}

type Store struct {
	root string

	mu    sync.RWMutex
	items map[string]VideoMeta
}

func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("media: mkdir %s: %w", root, err)
	}
	s := &Store{root: root, items: map[string]VideoMeta{}}
	if err := s.loadExisting(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Root() string { return s.root }

func (s *Store) loadExisting() error {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return fmt.Errorf("media: read dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		full := filepath.Join(s.root, e.Name())
		raw, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		var m VideoMeta
		if err := json.Unmarshal(raw, &m); err != nil || m.ID == "" {
			continue
		}
		s.items[m.ID] = m
	}
	return nil
}

// Save persists the video body and its metadata. contentType is stored as-is
// (the browser derives one from the upload). declaredSize is only used for
// metadata; the actual bytes written come from r.
func (s *Store) Save(filename, contentType string, declaredSize int64, r io.Reader) (VideoMeta, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := allowedExts[ext]; !ok {
		return VideoMeta{}, ErrUnsupportedFormat
	}
	id, err := newID()
	if err != nil {
		return VideoMeta{}, fmt.Errorf("media: id: %w", err)
	}

	videoPath := filepath.Join(s.root, id+ext)
	f, err := os.Create(videoPath)
	if err != nil {
		return VideoMeta{}, fmt.Errorf("media: create: %w", err)
	}
	written, err := io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(videoPath)
		return VideoMeta{}, fmt.Errorf("media: write: %w", err)
	}

	size := written
	if declaredSize > 0 && declaredSize < size {
		size = declaredSize
	}
	meta := VideoMeta{
		ID:          id,
		Filename:    filepath.Base(filename),
		Ext:         ext,
		ContentType: contentType,
		SizeBytes:   size,
		UploadedAt:  time.Now().UTC(),
		URLPath:     "/media/" + id + ext,
	}
	if err := s.writeMeta(meta); err != nil {
		_ = os.Remove(videoPath)
		return VideoMeta{}, err
	}
	s.mu.Lock()
	s.items[id] = meta
	s.mu.Unlock()
	return meta, nil
}

func (s *Store) writeMeta(m VideoMeta) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("media: marshal meta: %w", err)
	}
	metaPath := filepath.Join(s.root, m.ID+".json")
	return os.WriteFile(metaPath, raw, 0o644)
}

func (s *Store) List() []VideoMeta {
	s.mu.RLock()
	out := make([]VideoMeta, 0, len(s.items))
	for _, m := range s.items {
		out = append(out, m)
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		return out[i].UploadedAt.After(out[j].UploadedAt)
	})
	return out
}

func (s *Store) Get(id string) (VideoMeta, bool) {
	if !validID(id) {
		return VideoMeta{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.items[id]
	return m, ok
}

func (s *Store) Delete(id string) error {
	if !validID(id) {
		return ErrInvalidID
	}
	s.mu.Lock()
	m, ok := s.items[id]
	if !ok {
		s.mu.Unlock()
		return ErrNotFound
	}
	delete(s.items, id)
	s.mu.Unlock()

	videoErr := os.Remove(filepath.Join(s.root, m.ID+m.Ext))
	metaErr := os.Remove(filepath.Join(s.root, m.ID+".json"))
	if videoErr != nil && !os.IsNotExist(videoErr) {
		return fmt.Errorf("media: remove video: %w", videoErr)
	}
	if metaErr != nil && !os.IsNotExist(metaErr) {
		return fmt.Errorf("media: remove meta: %w", metaErr)
	}
	return nil
}

func newID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// validID enforces hex-only IDs so a client-supplied id can't escape the
// media root or clobber sibling files. Empty and slashy inputs are rejected.
func validID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		hexish := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		if !hexish {
			return false
		}
	}
	return true
}
