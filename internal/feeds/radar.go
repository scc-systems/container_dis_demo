package feeds

import (
	"fmt"
	"math"

	"github.com/eleanor/wallsync/internal/protocol"
)

// RadarAngle returns the sweep angle in degrees for t_ms.
// One full sweep per 4 seconds.
func RadarAngle(tMs int64) float32 {
	t := float64(tMs) / 1000.0
	return float32(math.Mod(t*90.0, 360.0)) // 90 deg/s
}

// Blips returns a deterministic list of ~5 blips for t_ms.
func Blips(tMs int64) []protocol.Blip {
	t := float64(tMs) / 1000.0
	n := 5
	out := make([]protocol.Blip, n)
	for i := 0; i < n; i++ {
		fi := float64(i)
		r := 0.35 + 0.5*math.Abs(math.Sin(0.13*t+fi*0.7))
		th := math.Mod(fi*1.256+t*0.4*(1+0.1*fi), 2*math.Pi)
		out[i] = protocol.Blip{R: float32(r), Theta: float32(th), ID: fmt.Sprintf("B%d", i)}
	}
	return out
}
