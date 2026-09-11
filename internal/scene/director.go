package scene

import (
	"log/slog"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scc-systems/wallsync/internal/config"
	"github.com/scc-systems/wallsync/internal/hub"
	"github.com/scc-systems/wallsync/internal/protocol"
	"github.com/scc-systems/wallsync/internal/timing"
)

type Snapshot struct {
	Scene   string
	EpochMs int64
	Params  map[string]any
	Seed    uint32
}

type Director struct {
	mu           sync.RWMutex
	current      string
	epochMs      int64
	params       map[string]any
	seed         uint32
	autoRotating atomic.Bool
	cfg          config.Config
	hub          *hub.Hub
	logger       *slog.Logger
}

func NewDirector(cfg config.Config, h *hub.Hub, logger *slog.Logger) *Director {
	if logger == nil {
		logger = slog.Default()
	}
	initial := cfg.InitialScene
	if !Valid(initial) {
		initial = SceneThreeUnified
	}
	return &Director{
		current: initial,
		epochMs: time.Now().UnixMilli() + cfg.SceneSwitchLead.Milliseconds(),
		params:  map[string]any{},
		seed:    rand.Uint32(),
		cfg:     cfg,
		hub:     h,
		logger:  logger,
	}
}

func (d *Director) Snapshot() Snapshot {
	d.mu.RLock()
	defer d.mu.RUnlock()
	p := make(map[string]any, len(d.params))
	for k, v := range d.params {
		p[k] = v
	}
	return Snapshot{Scene: d.current, EpochMs: d.epochMs, Params: p, Seed: d.seed}
}

// Welcome implements hub.WelcomeProvider.
func (d *Director) Welcome(nodeID int, nowMs int64) protocol.WelcomePayload {
	snap := d.Snapshot()
	return protocol.WelcomePayload{
		NodeID:        nodeID,
		AssignedTile:  protocol.AssignedTile{Col: nodeID, Rows: []int{0, 1, 2}},
		CurrentScene:  snap.Scene,
		SceneEpochMs:  snap.EpochMs,
		ServerTimeMs:  nowMs,
		TickHz:        d.cfg.TickHz,
		WallWidth:     d.cfg.WallWidth,
		WallHeight:    d.cfg.WallHeight,
		PerNodeWidth:  d.cfg.PerNodeWidth,
		PerNodeHeight: d.cfg.PerNodeHeight,
		NodeCount:     d.cfg.NodeCount,
		Seed:          snap.Seed,
	}
}

// Switch transitions to `next` with a future epoch = now + SceneSwitchLead.
// Broadcasts a scene_change message. Returns the new epoch ms.
func (d *Director) Switch(next string, params map[string]any) int64 {
	if !Valid(next) {
		return d.Snapshot().EpochMs
	}
	newEpoch := time.Now().UnixMilli() + d.cfg.SceneSwitchLead.Milliseconds()
	newSeed := rand.Uint32()
	d.mu.Lock()
	d.current = next
	d.epochMs = newEpoch
	if params != nil {
		d.params = params
	} else {
		d.params = map[string]any{}
	}
	d.seed = newSeed
	d.mu.Unlock()

	d.logger.Info("scene switch", "scene", next, "epoch_ms", newEpoch, "seed", newSeed)

	broadcastParams := map[string]any{"seed": newSeed}
	for k, v := range params {
		if k == "seed" {
			continue
		}
		broadcastParams[k] = v
	}
	raw, err := protocol.Marshal(protocol.TypeSceneChange, d.hub.NextSeq(), timing.EpochNowMs(), protocol.SceneChangePayload{
		Scene:   next,
		EpochMs: newEpoch,
		Params:  broadcastParams,
	})
	if err == nil {
		d.hub.Broadcast(raw)
	}
	return newEpoch
}

func (d *Director) SwitchByIndex(i int) int64 {
	i = ((i % len(All)) + len(All)) % len(All)
	return d.Switch(All[i], nil)
}

// Next advances to the next scene in rotation order.
func (d *Director) Next() int64 {
	cur := d.Snapshot().Scene
	idx := 0
	for i, s := range All {
		if s == cur {
			idx = (i + 1) % len(All)
			break
		}
	}
	return d.Switch(All[idx], nil)
}

func (d *Director) AutoRotating() bool { return d.autoRotating.Load() }

func (d *Director) ToggleAutoRotate() bool {
	for {
		old := d.autoRotating.Load()
		if d.autoRotating.CompareAndSwap(old, !old) {
			return !old
		}
	}
}
