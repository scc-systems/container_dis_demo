package playlist

import (
	"errors"
	"os"
	"testing"
)

func mustStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func TestCreateAndListRoundTrip(t *testing.T) {
	s := mustStore(t)
	items := []Item{
		{Kind: KindScene, Scene: "warp-jump", DurationS: 20},
		{Kind: KindVideo, RefID: "aaaa1111bbbb2222", Mode: "mirror", DurationS: 30},
	}
	p, err := s.Create("morning-loop", true, items)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID == "" || p.Name != "morning-loop" || !p.Loop || len(p.Items) != 2 {
		t.Fatalf("unexpected playlist: %+v", p)
	}
	list := s.List()
	if len(list) != 1 || list[0].ID != p.ID {
		t.Fatalf("List = %+v, want single with id %s", list, p.ID)
	}
}

func TestCreateRejectsInvalidItems(t *testing.T) {
	s := mustStore(t)
	cases := []struct {
		name  string
		items []Item
	}{
		{"missing scene", []Item{{Kind: KindScene, DurationS: 5}}},
		{"missing ref for video", []Item{{Kind: KindVideo, Mode: "mirror", DurationS: 5}}},
		{"missing ref for image", []Item{{Kind: KindImage, Mode: "mirror", DurationS: 5}}},
		{"zero duration", []Item{{Kind: KindScene, Scene: "warp-jump", DurationS: 0}}},
		{"unknown kind", []Item{{Kind: "song", DurationS: 5}}},
	}
	for _, tc := range cases {
		if _, err := s.Create("x", false, tc.items); !errors.Is(err, ErrInvalidItem) {
			t.Errorf("%s: expected ErrInvalidItem, got %v", tc.name, err)
		}
	}
}

func TestCreateRejectsEmptyName(t *testing.T) {
	s := mustStore(t)
	_, err := s.Create("", false, []Item{{Kind: KindScene, Scene: "warp-jump", DurationS: 5}})
	if !errors.Is(err, ErrEmptyName) {
		t.Fatalf("expected ErrEmptyName, got %v", err)
	}
}

func TestUpdateReplacesFields(t *testing.T) {
	s := mustStore(t)
	p, err := s.Create("orig", true, []Item{{Kind: KindScene, Scene: "warp-jump", DurationS: 5}})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := s.Update(p.ID, "renamed", false, []Item{
		{Kind: KindScene, Scene: "solar-panorama", DurationS: 45},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "renamed" || updated.Loop || len(updated.Items) != 1 || updated.Items[0].Scene != "solar-panorama" {
		t.Fatalf("Update did not replace fields: %+v", updated)
	}
	if !updated.UpdatedAt.After(p.UpdatedAt) {
		t.Fatalf("UpdatedAt did not advance: before=%v after=%v", p.UpdatedAt, updated.UpdatedAt)
	}
}

func TestUpdateNotFound(t *testing.T) {
	s := mustStore(t)
	_, err := s.Update("deadbeef", "x", false, []Item{{Kind: KindScene, Scene: "warp-jump", DurationS: 1}})
	if !errors.Is(err, ErrInvalidID) && !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrInvalidID/ErrNotFound, got %v", err)
	}
}

func TestDeleteRemovesFromDisk(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Create("x", true, []Item{{Kind: KindScene, Scene: "warp-jump", DurationS: 5}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := len(s.List()); got != 0 {
		t.Fatalf("List still has %d entries", got)
	}
	// Reopen — the deletion must have been flushed to disk.
	s2, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(s2.List()); got != 0 {
		t.Fatalf("after reopen still has %d entries", got)
	}
}

func TestReopenRestoresPlaylists(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := s.Create("a", true, []Item{{Kind: KindScene, Scene: "warp-jump", DurationS: 5}})
	b, _ := s.Create("b", false, []Item{{Kind: KindScene, Scene: "solar-panorama", DurationS: 30}})

	s2, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	list := s2.List()
	if len(list) != 2 {
		t.Fatalf("len(list)=%d, want 2", len(list))
	}
	ids := map[string]bool{list[0].ID: true, list[1].ID: true}
	if !ids[a.ID] || !ids[b.ID] {
		t.Fatalf("missing ids: got %+v want a=%s b=%s", ids, a.ID, b.ID)
	}
}

func TestFlushIsAtomic(t *testing.T) {
	// Sanity: after Create, no .tmp file lingers next to playlists.json.
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("x", true, []Item{{Kind: KindScene, Scene: "warp-jump", DurationS: 5}}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() == "playlists.json.tmp" {
			t.Fatalf("leftover .tmp file after Create")
		}
	}
}

func TestDeleteRejectsBadID(t *testing.T) {
	s := mustStore(t)
	for _, id := range []string{"../etc/passwd", "foo/bar", "..", "with space", ""} {
		if err := s.Delete(id); !errors.Is(err, ErrInvalidID) {
			t.Errorf("Delete(%q): expected ErrInvalidID, got %v", id, err)
		}
	}
}
