package feeds

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/scc-systems/wallsync/internal/hub"
	"github.com/scc-systems/wallsync/internal/protocol"
	"github.com/scc-systems/wallsync/internal/scene"
	"github.com/scc-systems/wallsync/internal/timing"
)

type Ticker struct {
	Hub      *hub.Hub
	Director *scene.Director
	Hz       int
	Logger   *slog.Logger
	FFTBins  int
}

func (t *Ticker) Run(ctx context.Context) {
	hz := t.Hz
	if hz <= 0 {
		hz = 30
	}
	period := time.Second / time.Duration(hz)
	tk := time.NewTicker(period)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tk.C:
			t.tick(now)
		}
	}
}

func (t *Ticker) tick(now time.Time) {
	snap := t.Director.Snapshot()
	nowMs := now.UnixMilli()
	if nowMs < snap.EpochMs {
		// Silent pre-epoch window: clients honour epoch_ms from scene_change.
		return
	}
	tMs := nowMs - snap.EpochMs

	payload := protocol.TickPayload{
		Scene: snap.Scene,
		TMs:   tMs,
	}
	switch snap.Scene {
	case scene.SceneThreeUnified:
		payload.Camera = &protocol.CameraTick{
			OrbitDeg: float32(math.Mod(float64(tMs)/1000.0*20.0, 360.0)),
		}
	case scene.SceneSolarPanorama:
		// Fully client-deterministic from t + seed; nothing to broadcast.
		payload.Seed = snap.Seed
	case scene.SceneWarpJump:
		// Fully client-deterministic from t + seed; nothing to broadcast.
		payload.Seed = snap.Seed
	}

	raw, err := protocol.Marshal(protocol.TypeTick, t.Hub.NextSeq(), timing.EpochNowMs(), payload)
	if err != nil {
		return
	}
	t.Hub.Broadcast(raw)
}
