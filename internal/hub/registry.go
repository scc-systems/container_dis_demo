package hub

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/scc-systems/wallsync/internal/timing"
)

type NodeState struct {
	ID           int
	Conn         *websocket.Conn
	Connected    bool
	ConnectedAt  time.Time
	LastPongAt   time.Time
	RTTStats     *timing.RollingStats
	OffsetStats  *timing.RollingStats
	SendCh       chan []byte
	Drops        atomic.Uint64
	pings        *timing.PingBook
	ctx          context.Context
	cancel       context.CancelFunc
}

type Registry struct {
	mu    sync.RWMutex
	nodes []*NodeState
}

func NewRegistry(nodeCount int) *Registry {
	return &Registry{
		nodes: make([]*NodeState, nodeCount),
	}
}

func (r *Registry) Get(id int) *NodeState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if id < 0 || id >= len(r.nodes) {
		return nil
	}
	return r.nodes[id]
}

func (r *Registry) Snapshot() []*NodeState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*NodeState, len(r.nodes))
	copy(out, r.nodes)
	return out
}

// Install atomically replaces any existing NodeState for this ID.
// If a previous conn exists and is Connected, it is cancelled and closed with WS 4000.
func (r *Registry) Install(ns *NodeState) {
	r.mu.Lock()
	prev := r.nodes[ns.ID]
	r.nodes[ns.ID] = ns
	r.mu.Unlock()
	if prev != nil && prev.Connected {
		prev.cancel()
		_ = prev.Conn.Close(websocket.StatusCode(4000), "superseded")
	}
}

// TryInstall installs ns only if the slot is empty or disconnected.
// Returns false (without installing) if a live connection already holds this node ID.
func (r *Registry) TryInstall(ns *NodeState) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	prev := r.nodes[ns.ID]
	if prev != nil && prev.Connected {
		return false
	}
	r.nodes[ns.ID] = ns
	return true
}

// Remove clears a NodeState entry ONLY if it still matches self (avoids clobbering a newer connection).
func (r *Registry) Remove(self *NodeState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if self.ID < 0 || self.ID >= len(r.nodes) {
		return
	}
	if r.nodes[self.ID] == self {
		self.Connected = false
	}
}
