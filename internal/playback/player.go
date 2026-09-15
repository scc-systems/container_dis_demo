// Package playback drives the runtime queue: an in-memory ordered list of
// playlist items that advances through the scene director on a per-item timer.
// The queue is transient (not persisted); saved playlists live in
// internal/playlist and are loaded into the queue on operator request.
package playback

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/scc-systems/wallsync/internal/media"
	"github.com/scc-systems/wallsync/internal/playlist"
	"github.com/scc-systems/wallsync/internal/protocol"
	"github.com/scc-systems/wallsync/internal/scene"
)

var (
	ErrEmptyQueue   = errors.New("playback: queue empty")
	ErrOutOfRange   = errors.New("playback: index out of range")
	ErrInvalidItem  = errors.New("playback: invalid item")
	ErrUnknownMedia = errors.New("playback: media not found")
	ErrUnknownScene = errors.New("playback: unknown scene")
)

// Director is the subset of scene.Director we need. Declared as an interface
// so tests can pass a fake without spinning up a hub.
type Director interface {
	Switch(next string, params map[string]any) int64
}

// fallbackVideoDuration is used when a video item has duration_s <= 0 and
// no natural duration has been reported yet. The client's loadedmetadata
// event typically arrives well before this fires; when it does, the timer
// is re-armed with the real duration.
const fallbackVideoDuration = 10 * time.Minute

type Player struct {
	director Director
	mediaSt  *media.Store
	logger   *slog.Logger
	basePath string

	mu      sync.Mutex
	items   []playlist.Item
	index   int
	loop    bool
	playing bool
	// paused is true when we were playing, then Pause() was called. The wall
	// still shows the paused item (video keeps looping via <video loop>, image
	// and scene stay mounted); a subsequent Play() must re-arm the timer WITHOUT
	// a scene switch so the video doesn't re-seek to t=0. Distinct from
	// "!playing && !paused" which means the queue has never started (or was
	// stopped) and Play() should do a full mount.
	paused bool
	timer  *time.Timer
	// awaitingDurationFor is set when the currently-playing item is a video
	// with unknown natural duration and we've armed the fallback timer. Cleared
	// when a duration report arrives (and the timer is re-armed) or on any
	// non-video advance.
	awaitingDurationFor string
}

func New(dir Director, m *media.Store, basePath string, logger *slog.Logger) *Player {
	if logger == nil {
		logger = slog.Default()
	}
	return &Player{
		director: dir,
		mediaSt:  m,
		logger:   logger,
		basePath: basePath,
	}
}

// Load replaces the queue with items and pauses playback at index 0.
func (p *Player) Load(items []playlist.Item, loop bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopTimerLocked()
	p.items = append([]playlist.Item(nil), items...)
	p.index = 0
	p.loop = loop
	p.playing = false
	p.paused = false
}

func (p *Player) Append(item playlist.Item) error {
	if err := playlist.ValidateItem(item); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items = append(p.items, item)
	return nil
}

// Insert places item at position (clamped into range). Adjusts current index
// so playback keeps pointing at the same underlying item.
func (p *Player) Insert(item playlist.Item, position int) error {
	if err := playlist.ValidateItem(item); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if position < 0 {
		position = 0
	}
	if position > len(p.items) {
		position = len(p.items)
	}
	p.items = append(p.items[:position], append([]playlist.Item{item}, p.items[position:]...)...)
	if position <= p.index {
		p.index++
	}
	return nil
}

// Remove drops the item at idx. If it's the current item, playback is stopped
// and the wall goes idle.
func (p *Player) Remove(idx int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if idx < 0 || idx >= len(p.items) {
		return ErrOutOfRange
	}
	current := (idx == p.index) && p.playing
	p.items = append(p.items[:idx], p.items[idx+1:]...)
	switch {
	case len(p.items) == 0:
		p.stopTimerLocked()
		p.playing = false
		p.index = 0
		if current {
			p.director.Switch(scene.SceneIdle, nil)
		}
	case idx < p.index:
		p.index--
	case idx == p.index:
		if p.index >= len(p.items) {
			p.index = 0
		}
		if current {
			p.startCurrentLocked()
		}
	}
	return nil
}

// Reorder moves item at from to position to (clamped). Adjusts current index
// so playback keeps pointing at the same underlying item.
func (p *Player) Reorder(from, to int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if from < 0 || from >= len(p.items) {
		return ErrOutOfRange
	}
	if to < 0 {
		to = 0
	}
	if to >= len(p.items) {
		to = len(p.items) - 1
	}
	if from == to {
		return nil
	}
	current := p.index
	item := p.items[from]
	p.items = append(p.items[:from], p.items[from+1:]...)
	p.items = append(p.items[:to], append([]playlist.Item{item}, p.items[to:]...)...)
	switch {
	case current == from:
		p.index = to
	case from < current && to >= current:
		p.index--
	case from > current && to <= current:
		p.index++
	}
	return nil
}

func (p *Player) Play() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.items) == 0 {
		return ErrEmptyQueue
	}
	if p.playing {
		return nil
	}
	if p.index < 0 || p.index >= len(p.items) {
		p.index = 0
	}
	// Resume path: the paused item is still on the wall, so re-arm the timer
	// without a scene switch. Video keeps looping seamlessly; no re-seek, no
	// fade-to-black flash.
	if p.paused {
		p.paused = false
		p.playing = true
		p.armTimerLocked(p.items[p.index])
		return nil
	}
	return p.startCurrentLocked()
}

func (p *Player) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.playing {
		return
	}
	p.stopTimerLocked()
	p.playing = false
	p.paused = true
}

// Skip advances immediately to the next item.
func (p *Player) Skip() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.items) == 0 {
		return ErrEmptyQueue
	}
	p.stopTimerLocked()
	return p.advanceLocked()
}

// Stop pauses playback, resets index to 0, and drops the wall to idle.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopTimerLocked()
	p.playing = false
	p.paused = false
	p.index = 0
	p.director.Switch(scene.SceneIdle, nil)
}

// Snapshot returns a serialisable view of the queue for admin UI polling.
func (p *Player) Snapshot() protocol.QueueSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	items := make([]protocol.QueueItemView, len(p.items))
	for i, it := range p.items {
		items[i] = protocol.QueueItemView{
			Kind:      string(it.Kind),
			Label:     p.itemLabelLocked(it),
			DurationS: it.DurationS,
			Mode:      it.Mode,
			RefID:     it.RefID,
			Scene:     it.Scene,
		}
	}
	return protocol.QueueSnapshot{
		Playing: p.playing,
		Paused:  p.paused,
		Loop:    p.loop,
		Index:   p.index,
		Items:   items,
	}
}

// SetLoop toggles the loop flag; if turned off mid-run at the last item, the
// next advance will Stop() rather than wrap.
func (p *Player) SetLoop(loop bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.loop = loop
}

// -- internal ---------------------------------------------------------------

func (p *Player) itemLabelLocked(it playlist.Item) string {
	switch it.Kind {
	case playlist.KindScene:
		return it.Scene
	case playlist.KindVideo, playlist.KindImage:
		if p.mediaSt != nil {
			if m, ok := p.mediaSt.Get(it.RefID); ok {
				return m.Filename
			}
		}
		return string(it.Kind) + ":" + it.RefID
	}
	return string(it.Kind)
}

func (p *Player) stopTimerLocked() {
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
}

func (p *Player) startCurrentLocked() error {
	if len(p.items) == 0 {
		p.playing = false
		return ErrEmptyQueue
	}
	if p.index < 0 || p.index >= len(p.items) {
		p.index = 0
	}
	item := p.items[p.index]
	if err := p.switchToLocked(item); err != nil {
		p.playing = false
		return err
	}
	p.playing = true
	p.paused = false
	p.armTimerLocked(item)
	return nil
}

func (p *Player) armTimerLocked(item playlist.Item) {
	p.awaitingDurationFor = ""
	if item.DurationS > 0 {
		p.timer = time.AfterFunc(time.Duration(item.DurationS)*time.Second, p.onTimerFire)
		return
	}
	// duration_s == 0 means "play to natural end" — only meaningful for videos.
	if item.Kind == playlist.KindVideo && p.mediaSt != nil {
		if m, ok := p.mediaSt.Get(item.RefID); ok && m.DurationMs > 0 {
			p.timer = time.AfterFunc(time.Duration(m.DurationMs)*time.Millisecond, p.onTimerFire)
			return
		}
		p.awaitingDurationFor = item.RefID
		p.timer = time.AfterFunc(fallbackVideoDuration, p.onTimerFire)
		return
	}
	// Non-video item with no duration: sensible short default.
	p.timer = time.AfterFunc(10*time.Second, p.onTimerFire)
}

// ReportVideoDuration is called from the WS read pump when a wall reports the
// natural duration of the currently-loading video. If the currently-playing
// queue item is that video and we're on the fallback timer, re-arm precisely.
func (p *Player) ReportVideoDuration(videoID string, ms int64) {
	if ms <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.playing || p.awaitingDurationFor != videoID {
		return
	}
	if p.timer != nil {
		p.timer.Stop()
	}
	p.awaitingDurationFor = ""
	p.timer = time.AfterFunc(time.Duration(ms)*time.Millisecond, p.onTimerFire)
}

func (p *Player) onTimerFire() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.playing {
		return
	}
	if err := p.advanceLocked(); err != nil {
		p.logger.Error("queue advance failed", "err", err)
	}
}

// advanceLocked moves to the next item, wrapping if loop is set. If we fall
// off the end without loop, stops playback and returns to idle.
func (p *Player) advanceLocked() error {
	if len(p.items) == 0 {
		p.playing = false
		return ErrEmptyQueue
	}
	next := p.index + 1
	if next >= len(p.items) {
		if !p.loop {
			p.playing = false
			p.index = 0
			p.director.Switch(scene.SceneIdle, nil)
			return nil
		}
		next = 0
	}
	p.index = next
	return p.startCurrentLocked()
}

func (p *Player) switchToLocked(item playlist.Item) error {
	switch item.Kind {
	case playlist.KindScene:
		if !scene.Valid(item.Scene) {
			return fmt.Errorf("%w: %s", ErrUnknownScene, item.Scene)
		}
		p.director.Switch(item.Scene, nil)
		return nil
	case playlist.KindVideo:
		return p.switchMediaLocked(item, scene.SceneVideo)
	case playlist.KindImage:
		return p.switchMediaLocked(item, scene.SceneImage)
	}
	return fmt.Errorf("%w: kind %s", ErrInvalidItem, item.Kind)
}

func (p *Player) switchMediaLocked(item playlist.Item, sceneID string) error {
	if p.mediaSt == nil {
		return ErrUnknownMedia
	}
	m, ok := p.mediaSt.Get(item.RefID)
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownMedia, item.RefID)
	}
	mode := item.Mode
	if mode == "" {
		mode = "mirror"
	}
	params := map[string]any{
		"url":      p.basePath + m.URLPath,
		"mode":     mode,
		"filename": m.Filename,
	}
	if sceneID == scene.SceneVideo {
		params["video_id"] = m.ID
	} else {
		params["image_id"] = m.ID
	}
	p.director.Switch(sceneID, params)
	return nil
}
