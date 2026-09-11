package scene

import (
	"context"
	"time"
)

// Autorotate runs permanently, advancing the scene each `period` only when
// d.AutoRotating() is true. The operator toggles this via the admin UI.
func Autorotate(ctx context.Context, d *Director, period time.Duration) {
	if period <= 0 {
		period = 20 * time.Second
	}
	t := time.NewTicker(period)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if d.AutoRotating() {
				d.Next()
			}
		}
	}
}
