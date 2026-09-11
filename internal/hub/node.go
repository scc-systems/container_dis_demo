package hub

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/eleanor/wallsync/internal/protocol"
	"github.com/eleanor/wallsync/internal/timing"
)

// WelcomeProvider gives the current welcome payload for a newly-connected node.
// Implemented by the scene.Director.
type WelcomeProvider interface {
	Welcome(nodeID int, nowMs int64) protocol.WelcomePayload
}

type ConnHandler struct {
	Hub          *Hub
	Welcomes     WelcomeProvider
	PingInterval time.Duration
	Logger       *slog.Logger
	NodeCount    int
}

func (h *ConnHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	nodeIDStr := r.URL.Query().Get("node")
	nodeID, err := strconv.Atoi(nodeIDStr)
	if err != nil || nodeID < 0 || nodeID >= h.NodeCount {
		http.Error(w, "invalid node id", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // LAN demo; no origin gating
	})
	if err != nil {
		h.Logger.Warn("ws accept failed", "err", err)
		return
	}
	conn.SetReadLimit(64 * 1024)

	ctx := r.Context()
	h.handleConn(ctx, conn, nodeID)
}

func (h *ConnHandler) handleConn(parentCtx context.Context, conn *websocket.Conn, expectedNode int) {
	// Read hello with a bounded deadline.
	helloCtx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
	_, data, err := conn.Read(helloCtx)
	cancel()
	if err != nil {
		_ = conn.Close(websocket.StatusPolicyViolation, "hello timeout")
		return
	}
	env, err := protocol.Unmarshal(data)
	if err != nil || env.Type != protocol.TypeHello {
		_ = conn.Close(websocket.StatusUnsupportedData, "expected hello")
		return
	}
	var hello protocol.HelloPayload
	if err := protocol.UnmarshalPayload(env, &hello); err != nil {
		_ = conn.Close(websocket.StatusUnsupportedData, "bad hello payload")
		return
	}
	if hello.NodeID != expectedNode {
		_ = conn.Close(websocket.StatusPolicyViolation, "node id mismatch")
		return
	}

	ns := newNodeState(parentCtx, expectedNode, conn, h.Hub.sendBuffer)
	if !h.Hub.Registry.TryInstall(ns) {
		h.Logger.Warn("node in use — rejecting duplicate connection", "node_id", expectedNode)
		ns.cancel()
		_ = conn.Close(websocket.StatusCode(4001), "node in use")
		return
	}
	h.Logger.Info("node connected", "node_id", expectedNode, "screen_w", hello.Screen.W, "screen_h", hello.Screen.H)

	// Send welcome.
	welcome := h.Welcomes.Welcome(expectedNode, timing.EpochNowMs())
	welcome.NodeID = expectedNode
	raw, err := protocol.Marshal(protocol.TypeWelcome, h.Hub.NextSeq(), timing.EpochNowMs(), welcome)
	if err == nil {
		writeCtx, wcancel := context.WithTimeout(ns.ctx, 2*time.Second)
		_ = conn.Write(writeCtx, websocket.MessageText, raw)
		wcancel()
	}

	defer func() {
		h.Hub.Registry.Remove(ns)
		ns.cancel()
		_ = conn.Close(websocket.StatusNormalClosure, "goodbye")
		h.Logger.Info("node disconnected", "node_id", expectedNode)
	}()

	// Start write pump, ping loop, then run read pump inline.
	go h.writePump(ns)
	go h.pingLoop(ns)
	h.readPump(ns)
}

func (h *ConnHandler) writePump(ns *NodeState) {
	for {
		select {
		case <-ns.ctx.Done():
			return
		case msg, ok := <-ns.SendCh:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(ns.ctx, 2*time.Second)
			err := ns.Conn.Write(writeCtx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				ns.cancel()
				return
			}
		}
	}
}

func (h *ConnHandler) pingLoop(ns *NodeState) {
	if h.PingInterval <= 0 {
		return
	}
	t := time.NewTicker(h.PingInterval)
	defer t.Stop()
	// Watchdog: if no pong for 3× interval, force close.
	watchdog := 3 * h.PingInterval
	for {
		select {
		case <-ns.ctx.Done():
			return
		case <-t.C:
			if time.Since(ns.LastPongAt) > watchdog {
				h.Logger.Warn("pong watchdog fired", "node_id", ns.ID)
				ns.cancel()
				_ = ns.Conn.Close(websocket.StatusPolicyViolation, "pong timeout")
				return
			}
			nonce, _ := ns.pings.Issue()
			raw, err := protocol.Marshal(protocol.TypePing, h.Hub.NextSeq(), timing.EpochNowMs(), protocol.PingPayload{Nonce: nonce})
			if err != nil {
				continue
			}
			select {
			case ns.SendCh <- raw:
			default:
				ns.Drops.Add(1)
			}
		}
	}
}

func (h *ConnHandler) readPump(ns *NodeState) {
	for {
		_, data, err := ns.Conn.Read(ns.ctx)
		if err != nil {
			var closeErr websocket.CloseError
			if errors.As(err, &closeErr) {
				h.Logger.Info("node closed", "node_id", ns.ID, "code", closeErr.Code)
			}
			return
		}
		env, err := protocol.Unmarshal(data)
		if err != nil {
			continue
		}
		switch env.Type {
		case protocol.TypePong:
			var p protocol.PongPayload
			if err := protocol.UnmarshalPayload(env, &p); err != nil {
				continue
			}
			rtt, offset, ok := ns.pings.Resolve(p.Nonce, p.ClientTsMs)
			if ok {
				ns.RTTStats.Push(float64(rtt) / float64(time.Millisecond))
				ns.OffsetStats.Push(offset)
				ns.LastPongAt = time.Now()
				if raw, err := protocol.Marshal(protocol.TypeNodeStats, 0, timing.EpochNowMs(), protocol.NodeStatsPayload{
					RTTMedianMs: ns.RTTStats.Median(),
					RTTP95Ms:    ns.RTTStats.P95(),
					OffsetMs:    ns.OffsetStats.Median(),
					Drops:       ns.Drops.Load(),
				}); err == nil {
					select {
					case ns.SendCh <- raw:
					default:
					}
				}
			}
		default:
			// Ignore unknown client-originated messages for now.
			_ = json.RawMessage{}
		}
	}
}
