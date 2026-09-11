package hub

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/scc-systems/wallsync/internal/timing"
)

type Hub struct {
	Registry   *Registry
	seq        atomic.Uint64
	logger     *slog.Logger
	sendBuffer int
}

func New(logger *slog.Logger, nodeCount int) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		Registry:   NewRegistry(nodeCount),
		logger:     logger,
		sendBuffer: 64,
	}
}

func (h *Hub) NextSeq() uint64 {
	return h.seq.Add(1)
}

// Broadcast enqueues `data` on every connected node's SendCh without blocking.
// A slow node drops the message (increments Drops).
func (h *Hub) Broadcast(data []byte) {
	for _, ns := range h.Registry.Snapshot() {
		if ns == nil || !ns.Connected {
			continue
		}
		select {
		case ns.SendCh <- data:
		default:
			ns.Drops.Add(1)
		}
	}
}

// SendTo attempts to send to a single node.
func (h *Hub) SendTo(id int, data []byte) bool {
	ns := h.Registry.Get(id)
	if ns == nil || !ns.Connected {
		return false
	}
	select {
	case ns.SendCh <- data:
		return true
	default:
		ns.Drops.Add(1)
		return false
	}
}

// newNodeState prepares a NodeState for a fresh connection.
func newNodeState(ctx context.Context, id int, conn *websocket.Conn, sendBuf int) *NodeState {
	nctx, cancel := context.WithCancel(ctx)
	return &NodeState{
		ID:          id,
		Conn:        conn,
		Connected:   true,
		ConnectedAt: time.Now(),
		LastPongAt:  time.Now(),
		RTTStats:    timing.NewRollingStats(30),
		OffsetStats: timing.NewRollingStats(30),
		SendCh:      make(chan []byte, sendBuf),
		pings:       timing.NewPingBook(30 * time.Second),
		ctx:         nctx,
		cancel:      cancel,
	}
}
