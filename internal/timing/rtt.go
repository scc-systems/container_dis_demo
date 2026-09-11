package timing

import (
	"crypto/rand"
	"encoding/hex"
	"math"
	"sort"
	"sync"
	"time"
)

type PingBook struct {
	mu          sync.Mutex
	outstanding map[string]time.Time
	ttl         time.Duration
}

func NewPingBook(ttl time.Duration) *PingBook {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &PingBook{
		outstanding: make(map[string]time.Time),
		ttl:         ttl,
	}
}

func (pb *PingBook) Issue() (nonce string, sentAt time.Time) {
	var buf [4]byte
	_, _ = rand.Read(buf[:])
	nonce = hex.EncodeToString(buf[:])
	sentAt = time.Now()
	pb.mu.Lock()
	pb.outstanding[nonce] = sentAt
	pb.gcLocked(sentAt)
	pb.mu.Unlock()
	return
}

func (pb *PingBook) Resolve(nonce string, clientTsMs int64) (rtt time.Duration, offsetMs float64, ok bool) {
	pb.mu.Lock()
	sentAt, present := pb.outstanding[nonce]
	if present {
		delete(pb.outstanding, nonce)
	}
	pb.mu.Unlock()
	if !present {
		return 0, 0, false
	}
	now := time.Now()
	rtt = now.Sub(sentAt)
	halfRtt := float64(rtt) / 2 / float64(time.Millisecond)
	serverMidMs := float64(sentAt.UnixMilli()) + halfRtt
	offsetMs = float64(clientTsMs) - serverMidMs
	return rtt, offsetMs, true
}

func (pb *PingBook) gcLocked(now time.Time) {
	deadline := now.Add(-pb.ttl)
	for k, v := range pb.outstanding {
		if v.Before(deadline) {
			delete(pb.outstanding, k)
		}
	}
}

type RollingStats struct {
	mu      sync.Mutex
	buf     []float64
	capacity int
	i       int
	filled  bool
}

func NewRollingStats(capacity int) *RollingStats {
	if capacity <= 0 {
		capacity = 30
	}
	return &RollingStats{
		buf:      make([]float64, capacity),
		capacity: capacity,
	}
}

func (r *RollingStats) Push(v float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf[r.i] = v
	r.i = (r.i + 1) % r.capacity
	if r.i == 0 {
		r.filled = true
	}
}

func (r *RollingStats) snapshot() []float64 {
	n := r.capacity
	if !r.filled {
		n = r.i
	}
	if n == 0 {
		return nil
	}
	s := make([]float64, n)
	copy(s, r.buf[:n])
	sort.Float64s(s)
	return s
}

func (r *RollingStats) Median() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.snapshot()
	if len(s) == 0 {
		return 0
	}
	return s[len(s)/2]
}

func (r *RollingStats) P95() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.snapshot()
	if len(s) == 0 {
		return 0
	}
	idx := int(math.Ceil(0.95*float64(len(s)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s) {
		idx = len(s) - 1
	}
	return s[idx]
}

func (r *RollingStats) Latest() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.filled && r.i == 0 {
		return 0
	}
	prev := r.i - 1
	if prev < 0 {
		prev = r.capacity - 1
	}
	return r.buf[prev]
}
