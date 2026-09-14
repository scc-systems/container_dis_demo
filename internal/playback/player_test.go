package playback

import (
	"bytes"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/scc-systems/wallsync/internal/media"
	"github.com/scc-systems/wallsync/internal/playlist"
	"github.com/scc-systems/wallsync/internal/scene"
)

// fakeDirector records every Switch call so tests can assert playback order.
type fakeDirector struct {
	mu     sync.Mutex
	switch_ []switchCall
}

type switchCall struct {
	scene  string
	params map[string]any
}

func (f *fakeDirector) Switch(next string, params map[string]any) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.switch_ = append(f.switch_, switchCall{scene: next, params: params})
	return 0
}

func (f *fakeDirector) calls() []switchCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]switchCall, len(f.switch_))
	copy(out, f.switch_)
	return out
}

func newPlayer(t *testing.T) (*Player, *fakeDirector) {
	t.Helper()
	fd := &fakeDirector{}
	p := New(fd, nil, "", nil)
	return p, fd
}

func TestPlayAdvancesInOrder(t *testing.T) {
	p, fd := newPlayer(t)
	items := []playlist.Item{
		{Kind: playlist.KindScene, Scene: scene.SceneThreeUnified, DurationS: 1},
		{Kind: playlist.KindScene, Scene: scene.SceneSolarPanorama, DurationS: 1},
		{Kind: playlist.KindScene, Scene: scene.SceneWarpJump, DurationS: 1},
	}
	p.Load(items, false)
	if err := p.Play(); err != nil {
		t.Fatalf("Play: %v", err)
	}
	// After ~2.5s all three items plus the idle-on-end switch should have fired.
	time.Sleep(3200 * time.Millisecond)
	calls := fd.calls()
	want := []string{scene.SceneThreeUnified, scene.SceneSolarPanorama, scene.SceneWarpJump, scene.SceneIdle}
	if len(calls) != len(want) {
		t.Fatalf("got %d switches, want %d (%+v)", len(calls), len(want), calls)
	}
	for i, w := range want {
		if calls[i].scene != w {
			t.Fatalf("switch %d = %s, want %s", i, calls[i].scene, w)
		}
	}
}

func TestLoopWrapsAtEnd(t *testing.T) {
	p, fd := newPlayer(t)
	items := []playlist.Item{
		{Kind: playlist.KindScene, Scene: scene.SceneThreeUnified, DurationS: 1},
		{Kind: playlist.KindScene, Scene: scene.SceneWarpJump, DurationS: 1},
	}
	p.Load(items, true)
	if err := p.Play(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2500 * time.Millisecond)
	p.Pause()
	calls := fd.calls()
	if len(calls) < 3 {
		t.Fatalf("expected at least 3 switches (wrapped), got %d: %+v", len(calls), calls)
	}
	if calls[0].scene != scene.SceneThreeUnified || calls[1].scene != scene.SceneWarpJump {
		t.Fatalf("first two = %s, %s; want three-unified then warp-jump", calls[0].scene, calls[1].scene)
	}
	if calls[2].scene != scene.SceneThreeUnified {
		t.Fatalf("expected wrap-around to three-unified at index 2, got %s", calls[2].scene)
	}
}

func TestSkipAdvancesImmediately(t *testing.T) {
	p, fd := newPlayer(t)
	items := []playlist.Item{
		{Kind: playlist.KindScene, Scene: scene.SceneThreeUnified, DurationS: 60},
		{Kind: playlist.KindScene, Scene: scene.SceneWarpJump, DurationS: 60},
	}
	p.Load(items, false)
	if err := p.Play(); err != nil {
		t.Fatal(err)
	}
	// Immediately skip — should switch to warp-jump without waiting 60s.
	time.Sleep(50 * time.Millisecond)
	if err := p.Skip(); err != nil {
		t.Fatalf("Skip: %v", err)
	}
	calls := fd.calls()
	if len(calls) < 2 {
		t.Fatalf("expected at least 2 switches after Skip, got %d", len(calls))
	}
	if calls[1].scene != scene.SceneWarpJump {
		t.Fatalf("after Skip expected warp-jump, got %s", calls[1].scene)
	}
}

func TestStopReturnsToIdle(t *testing.T) {
	p, fd := newPlayer(t)
	p.Load([]playlist.Item{{Kind: playlist.KindScene, Scene: scene.SceneThreeUnified, DurationS: 60}}, false)
	if err := p.Play(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	p.Stop()
	calls := fd.calls()
	if len(calls) < 2 {
		t.Fatalf("expected >=2 switches, got %d", len(calls))
	}
	if calls[len(calls)-1].scene != scene.SceneIdle {
		t.Fatalf("Stop should end with idle, got %s", calls[len(calls)-1].scene)
	}
}

func TestPlayEmptyQueueErrors(t *testing.T) {
	p, _ := newPlayer(t)
	if err := p.Play(); !errors.Is(err, ErrEmptyQueue) {
		t.Fatalf("expected ErrEmptyQueue, got %v", err)
	}
}

func TestReorderKeepsCurrent(t *testing.T) {
	p, _ := newPlayer(t)
	items := []playlist.Item{
		{Kind: playlist.KindScene, Scene: scene.SceneThreeUnified, DurationS: 60},
		{Kind: playlist.KindScene, Scene: scene.SceneSolarPanorama, DurationS: 60},
		{Kind: playlist.KindScene, Scene: scene.SceneWarpJump, DurationS: 60},
	}
	p.Load(items, false)
	// index 0 (three-unified) is current. Move index 2 (warp-jump) to position 0.
	if err := p.Reorder(2, 0); err != nil {
		t.Fatalf("Reorder: %v", err)
	}
	snap := p.Snapshot()
	if snap.Items[0].Scene != scene.SceneWarpJump {
		t.Fatalf("after reorder items[0] = %s, want warp-jump", snap.Items[0].Scene)
	}
	// The originally-current three-unified item should now be at index 1.
	if snap.Index != 1 {
		t.Fatalf("after reorder index = %d, want 1", snap.Index)
	}
	if snap.Items[snap.Index].Scene != scene.SceneThreeUnified {
		t.Fatalf("current item drifted: %s", snap.Items[snap.Index].Scene)
	}
}

func TestInsertShiftsIndex(t *testing.T) {
	p, _ := newPlayer(t)
	items := []playlist.Item{
		{Kind: playlist.KindScene, Scene: scene.SceneThreeUnified, DurationS: 60},
		{Kind: playlist.KindScene, Scene: scene.SceneWarpJump, DurationS: 60},
	}
	p.Load(items, false)
	// After Load, index is 0. Insert before it — index should shift to 1.
	if err := p.Insert(playlist.Item{Kind: playlist.KindScene, Scene: scene.SceneSolarPanorama, DurationS: 60}, 0); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	snap := p.Snapshot()
	if snap.Index != 1 {
		t.Fatalf("after Insert at 0, index = %d, want 1", snap.Index)
	}
	if snap.Items[snap.Index].Scene != scene.SceneThreeUnified {
		t.Fatalf("current drifted: %s", snap.Items[snap.Index].Scene)
	}
}

func TestVideoUsesKnownNaturalDuration(t *testing.T) {
	ms, err := media.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := ms.Save("clip.mp4", "video/mp4", 4, bytes.NewReader([]byte("data")))
	if err != nil {
		t.Fatal(err)
	}
	if err := ms.SetDuration(m.ID, 800); err != nil { // 0.8s natural duration
		t.Fatal(err)
	}
	fd := &fakeDirector{}
	p := New(fd, ms, "", nil)
	p.Load([]playlist.Item{
		{Kind: playlist.KindVideo, RefID: m.ID, Mode: "mirror", DurationS: 0},
		{Kind: playlist.KindScene, Scene: scene.SceneWarpJump, DurationS: 60},
	}, false)
	if err := p.Play(); err != nil {
		t.Fatal(err)
	}
	// The video should advance to warp-jump after ~0.8s. Wait 1.5s to be safe.
	time.Sleep(1500 * time.Millisecond)
	p.Pause()
	calls := fd.calls()
	if len(calls) < 2 {
		t.Fatalf("expected >=2 switches after natural end, got %d: %+v", len(calls), calls)
	}
	if calls[1].scene != scene.SceneWarpJump {
		t.Fatalf("expected warp-jump after video's natural end, got %s", calls[1].scene)
	}
}

func TestReportVideoDurationRearms(t *testing.T) {
	ms, err := media.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := ms.Save("clip.mp4", "video/mp4", 4, bytes.NewReader([]byte("data")))
	if err != nil {
		t.Fatal(err)
	}
	// No SetDuration up front — Player will use the fallback timer.
	fd := &fakeDirector{}
	p := New(fd, ms, "", nil)
	p.Load([]playlist.Item{
		{Kind: playlist.KindVideo, RefID: m.ID, Mode: "mirror", DurationS: 0},
		{Kind: playlist.KindScene, Scene: scene.SceneWarpJump, DurationS: 60},
	}, false)
	if err := p.Play(); err != nil {
		t.Fatal(err)
	}
	// Simulate a client reporting 600ms as the natural duration.
	time.Sleep(50 * time.Millisecond)
	p.ReportVideoDuration(m.ID, 600)
	time.Sleep(900 * time.Millisecond)
	p.Pause()
	calls := fd.calls()
	if len(calls) < 2 {
		t.Fatalf("expected >=2 switches after re-arm, got %d: %+v", len(calls), calls)
	}
	if calls[1].scene != scene.SceneWarpJump {
		t.Fatalf("expected warp-jump after re-armed duration, got %s", calls[1].scene)
	}
}

func TestReportVideoDurationIgnoredForWrongVideo(t *testing.T) {
	ms, err := media.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := ms.Save("clip.mp4", "video/mp4", 4, bytes.NewReader([]byte("data")))
	if err != nil {
		t.Fatal(err)
	}
	fd := &fakeDirector{}
	p := New(fd, ms, "", nil)
	p.Load([]playlist.Item{
		{Kind: playlist.KindVideo, RefID: m.ID, Mode: "mirror", DurationS: 0},
	}, false)
	if err := p.Play(); err != nil {
		t.Fatal(err)
	}
	// Wrong id — should not affect the fallback timer.
	p.ReportVideoDuration("other-video-id", 100)
	time.Sleep(300 * time.Millisecond)
	p.Pause()
	// Only the initial video switch should have happened; no advance yet
	// because fallback is 10 minutes.
	calls := fd.calls()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 switch (initial video), got %d: %+v", len(calls), calls)
	}
}

func TestPauseStopsTimer(t *testing.T) {
	p, fd := newPlayer(t)
	p.Load([]playlist.Item{
		{Kind: playlist.KindScene, Scene: scene.SceneThreeUnified, DurationS: 1},
		{Kind: playlist.KindScene, Scene: scene.SceneWarpJump, DurationS: 1},
	}, false)
	if err := p.Play(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	p.Pause()
	before := len(fd.calls())
	// Wait past the 1s duration — no advance should happen.
	time.Sleep(1500 * time.Millisecond)
	after := len(fd.calls())
	if after != before {
		t.Fatalf("Pause did not stop timer: before=%d after=%d calls=%+v", before, after, fd.calls())
	}
}
