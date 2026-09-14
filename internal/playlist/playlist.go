// Package playlist persists named playlists to a single JSON file on disk,
// mirroring the file-backed pattern used by internal/media. A playlist is an
// ordered list of items (scene / video / image) with a per-item duration used
// by the runtime queue player.
package playlist

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var (
	ErrNotFound     = errors.New("playlist: not found")
	ErrInvalidID    = errors.New("playlist: invalid id")
	ErrInvalidItem  = errors.New("playlist: invalid item")
	ErrEmptyName    = errors.New("playlist: name required")
)

type ItemKind string

const (
	KindScene ItemKind = "scene"
	KindVideo ItemKind = "video"
	KindImage ItemKind = "image"
)

// Item is one entry in a playlist. RefID is the media store id for
// video/image items; Scene is the scene id for scene items. DurationS is the
// seconds the queue player will hold this item on screen before advancing.
type Item struct {
	Kind      ItemKind `json:"kind"`
	RefID     string   `json:"ref_id,omitempty"`
	Scene     string   `json:"scene,omitempty"`
	Mode      string   `json:"mode,omitempty"`
	DurationS int      `json:"duration_s"`
}

type Playlist struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Loop      bool      `json:"loop"`
	Items     []Item    `json:"items"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// fileFormat is the on-disk shape of playlists.json.
type fileFormat struct {
	Playlists []Playlist `json:"playlists"`
}

type Store struct {
	path string

	mu    sync.RWMutex
	items map[string]Playlist
}

// New opens (or creates) the playlists file rooted at dir. Missing file is not
// an error — the store starts empty and writes on first Save.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("playlist: mkdir %s: %w", dir, err)
	}
	s := &Store{
		path:  filepath.Join(dir, "playlists.json"),
		items: map[string]Playlist{},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Path() string { return s.path }

func (s *Store) load() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("playlist: read %s: %w", s.path, err)
	}
	var ff fileFormat
	if err := json.Unmarshal(raw, &ff); err != nil {
		return fmt.Errorf("playlist: parse %s: %w", s.path, err)
	}
	for _, p := range ff.Playlists {
		if p.ID == "" {
			continue
		}
		if p.Items == nil {
			p.Items = []Item{}
		}
		s.items[p.ID] = p
	}
	return nil
}

func (s *Store) flush() error {
	all := make([]Playlist, 0, len(s.items))
	for _, p := range s.items {
		all = append(all, p)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.Before(all[j].CreatedAt) })
	raw, err := json.MarshalIndent(fileFormat{Playlists: all}, "", "  ")
	if err != nil {
		return fmt.Errorf("playlist: marshal: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("playlist: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("playlist: rename %s -> %s: %w", tmp, s.path, err)
	}
	return nil
}

// List returns all saved playlists in creation order.
func (s *Store) List() []Playlist {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Playlist, 0, len(s.items))
	for _, p := range s.items {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *Store) Get(id string) (Playlist, bool) {
	if !validID(id) {
		return Playlist{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.items[id]
	return p, ok
}

// Create persists a new playlist. Name must be non-empty; items are validated
// by ValidateItem (structural only — media/scene existence is the caller's
// responsibility since this package has no dependency on media/scene).
func (s *Store) Create(name string, loop bool, items []Item) (Playlist, error) {
	if name == "" {
		return Playlist{}, ErrEmptyName
	}
	for i, it := range items {
		if err := ValidateItem(it); err != nil {
			return Playlist{}, fmt.Errorf("playlist: item %d: %w", i, err)
		}
	}
	id, err := newID()
	if err != nil {
		return Playlist{}, fmt.Errorf("playlist: id: %w", err)
	}
	now := time.Now().UTC()
	p := Playlist{
		ID:        id,
		Name:      name,
		Loop:      loop,
		Items:     cloneItems(items),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Lock()
	s.items[id] = p
	err = s.flush()
	if err != nil {
		delete(s.items, id)
	}
	s.mu.Unlock()
	if err != nil {
		return Playlist{}, err
	}
	return p, nil
}

// Update replaces the mutable fields of an existing playlist.
func (s *Store) Update(id, name string, loop bool, items []Item) (Playlist, error) {
	if !validID(id) {
		return Playlist{}, ErrInvalidID
	}
	if name == "" {
		return Playlist{}, ErrEmptyName
	}
	for i, it := range items {
		if err := ValidateItem(it); err != nil {
			return Playlist{}, fmt.Errorf("playlist: item %d: %w", i, err)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.items[id]
	if !ok {
		return Playlist{}, ErrNotFound
	}
	prev := p
	p.Name = name
	p.Loop = loop
	p.Items = cloneItems(items)
	p.UpdatedAt = time.Now().UTC()
	s.items[id] = p
	if err := s.flush(); err != nil {
		s.items[id] = prev
		return Playlist{}, err
	}
	return p, nil
}

func (s *Store) Delete(id string) error {
	if !validID(id) {
		return ErrInvalidID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	prev, ok := s.items[id]
	if !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	if err := s.flush(); err != nil {
		s.items[id] = prev
		return err
	}
	return nil
}

func ValidateItem(it Item) error {
	switch it.Kind {
	case KindScene:
		if it.Scene == "" {
			return fmt.Errorf("%w: scene items require scene id", ErrInvalidItem)
		}
		if it.DurationS < 1 {
			return fmt.Errorf("%w: scene items require duration_s >= 1", ErrInvalidItem)
		}
	case KindImage:
		if it.RefID == "" {
			return fmt.Errorf("%w: image items require ref_id", ErrInvalidItem)
		}
		if it.DurationS < 1 {
			return fmt.Errorf("%w: image items require duration_s >= 1", ErrInvalidItem)
		}
	case KindVideo:
		if it.RefID == "" {
			return fmt.Errorf("%w: video items require ref_id", ErrInvalidItem)
		}
		if it.DurationS < 0 {
			return fmt.Errorf("%w: video duration_s must be 0 (play to end) or >= 1", ErrInvalidItem)
		}
	default:
		return fmt.Errorf("%w: unknown kind %q", ErrInvalidItem, it.Kind)
	}
	return nil
}

// cloneItems returns a defensive copy that is guaranteed non-nil, so the JSON
// encoding is always a `[]` array rather than `null` for empty playlists.
func cloneItems(items []Item) []Item {
	out := make([]Item, len(items))
	copy(out, items)
	return out
}

func newID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

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
