package media

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func TestSaveThenListRoundTrip(t *testing.T) {
	s := newStore(t)
	body := bytes.NewReader([]byte("fake-mp4-bytes"))
	meta, err := s.Save("clip.mp4", "video/mp4", int64(len("fake-mp4-bytes")), body)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if meta.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if meta.Filename != "clip.mp4" {
		t.Fatalf("filename mismatch: %q", meta.Filename)
	}
	if meta.SizeBytes != int64(len("fake-mp4-bytes")) {
		t.Fatalf("size mismatch: %d", meta.SizeBytes)
	}
	if !strings.HasSuffix(meta.URLPath, ".mp4") {
		t.Fatalf("URLPath missing extension: %q", meta.URLPath)
	}

	list := s.List()
	if len(list) != 1 || list[0].ID != meta.ID {
		t.Fatalf("List returned %+v, want 1 with id %s", list, meta.ID)
	}
}

func TestSaveRejectsBadExtension(t *testing.T) {
	s := newStore(t)
	_, err := s.Save("evil.exe", "application/octet-stream", 3, bytes.NewReader([]byte("bad")))
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestSaveAcceptsMp4WebmMov(t *testing.T) {
	s := newStore(t)
	for _, name := range []string{"a.mp4", "b.MP4", "c.webm", "d.mov", "e.MOV"} {
		if _, err := s.Save(name, "video/mp4", 3, bytes.NewReader([]byte("xxx"))); err != nil {
			t.Fatalf("Save(%q): %v", name, err)
		}
	}
	if got := len(s.List()); got != 5 {
		t.Fatalf("got %d entries, want 5", got)
	}
}

func TestDeleteRemovesFileAndMeta(t *testing.T) {
	s := newStore(t)
	meta, err := s.Save("clip.mp4", "video/mp4", 3, bytes.NewReader([]byte("xxx")))
	if err != nil {
		t.Fatal(err)
	}
	videoPath := filepath.Join(s.Root(), meta.ID+".mp4")
	metaPath := filepath.Join(s.Root(), meta.ID+".json")
	if _, err := os.Stat(videoPath); err != nil {
		t.Fatalf("video file missing after Save: %v", err)
	}
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("meta file missing after Save: %v", err)
	}

	if err := s.Delete(meta.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(videoPath); !os.IsNotExist(err) {
		t.Fatalf("video file still present: %v", err)
	}
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Fatalf("meta file still present: %v", err)
	}
	if len(s.List()) != 0 {
		t.Fatalf("List still shows entries: %+v", s.List())
	}
}

func TestDeleteMissingIsNotFound(t *testing.T) {
	s := newStore(t)
	if err := s.Delete("deadbeef00112233"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteRejectsPathTraversal(t *testing.T) {
	s := newStore(t)
	for _, id := range []string{"../etc/passwd", "foo/bar", "..", "with space", ""} {
		if err := s.Delete(id); !errors.Is(err, ErrInvalidID) {
			t.Errorf("Delete(%q): expected ErrInvalidID, got %v", id, err)
		}
	}
}

func TestNewRestoresExistingEntries(t *testing.T) {
	dir := t.TempDir()
	s1, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	m, err := s1.Save("clip.mp4", "video/mp4", 3, bytes.NewReader([]byte("xxx")))
	if err != nil {
		t.Fatal(err)
	}

	s2, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	list := s2.List()
	if len(list) != 1 || list[0].ID != m.ID || list[0].Filename != "clip.mp4" {
		t.Fatalf("restored list = %+v, want single entry with id %s", list, m.ID)
	}
}

func TestListSortedNewestFirst(t *testing.T) {
	s := newStore(t)
	a, _ := s.Save("a.mp4", "video/mp4", 1, bytes.NewReader([]byte("a")))
	b, _ := s.Save("b.mp4", "video/mp4", 1, bytes.NewReader([]byte("b")))
	c, _ := s.Save("c.mp4", "video/mp4", 1, bytes.NewReader([]byte("c")))
	list := s.List()
	if len(list) != 3 {
		t.Fatalf("len=%d", len(list))
	}
	got := []string{list[0].ID, list[1].ID, list[2].ID}
	want := []string{c.ID, b.ID, a.ID}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}
