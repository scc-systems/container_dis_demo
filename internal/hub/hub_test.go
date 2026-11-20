package hub

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/scc-systems/wallsync/internal/protocol"
	"github.com/scc-systems/wallsync/internal/timing"
)

const testNodes = 3

type fakeWelcomes struct{}

func (fakeWelcomes) Welcome(nodeID int, nowMs int64) protocol.WelcomePayload {
	return protocol.WelcomePayload{CurrentScene: "test-scene", ServerTimeMs: nowMs, TickHz: 30}
}

// newTestServer starts a ConnHandler behind an httptest server. Client
// messages other than pong are sent on the returned channel.
func newTestServer(t *testing.T, ping time.Duration) (*Hub, *httptest.Server, <-chan protocol.Envelope) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(logger, testNodes)
	msgs := make(chan protocol.Envelope, 16)
	srv := httptest.NewServer(&ConnHandler{
		Hub:          h,
		Welcomes:     fakeWelcomes{},
		PingInterval: ping,
		Logger:       logger,
		NodeCount:    testNodes,
		OnClientMessage: func(nodeID int, env protocol.Envelope) {
			msgs <- env
		},
	})
	t.Cleanup(srv.Close)
	return h, srv, msgs
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func dial(t *testing.T, ctx context.Context, srv *httptest.Server, node int) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/?node=" + strconv.Itoa(node)
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial node %d: %v", node, err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })
	return conn
}

func send(t *testing.T, ctx context.Context, conn *websocket.Conn, msgType string, payload any) {
	t.Helper()
	raw, err := protocol.Marshal(msgType, 0, 0, payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", msgType, err)
	}
	if err := conn.Write(ctx, websocket.MessageText, raw); err != nil {
		t.Fatalf("write %s: %v", msgType, err)
	}
}

func read(t *testing.T, ctx context.Context, conn *websocket.Conn) protocol.Envelope {
	t.Helper()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	env, err := protocol.Unmarshal(data)
	if err != nil {
		t.Fatalf("unmarshal %q: %v", data, err)
	}
	return env
}

func sendHello(t *testing.T, ctx context.Context, conn *websocket.Conn, node int) {
	t.Helper()
	send(t, ctx, conn, protocol.TypeHello, protocol.HelloPayload{
		NodeID:     node,
		ClientTsMs: timing.EpochNowMs(),
		Screen:     protocol.Screen{W: 5760, H: 3240},
	})
}

// connect dials a node and completes the hello/welcome handshake.
func connect(t *testing.T, ctx context.Context, srv *httptest.Server, node int) *websocket.Conn {
	t.Helper()
	conn := dial(t, ctx, srv, node)
	sendHello(t, ctx, conn, node)
	env := read(t, ctx, conn)
	if env.Type != protocol.TypeWelcome {
		t.Fatalf("first message: got %q want %q", env.Type, protocol.TypeWelcome)
	}
	var w protocol.WelcomePayload
	if err := protocol.UnmarshalPayload(env, &w); err != nil {
		t.Fatalf("welcome payload: %v", err)
	}
	if w.NodeID != node || w.CurrentScene != "test-scene" || w.TickHz != 30 {
		t.Fatalf("welcome payload: %+v", w)
	}
	return conn
}

// wantClose reads until the server closes the connection and checks the code.
func wantClose(t *testing.T, ctx context.Context, conn *websocket.Conn, want websocket.StatusCode) {
	t.Helper()
	for {
		_, _, err := conn.Read(ctx)
		if err == nil {
			continue
		}
		if got := websocket.CloseStatus(err); got != want {
			t.Fatalf("close status: got %v want %v (err: %v)", got, want, err)
		}
		return
	}
}

func TestRejectsInvalidNodeID(t *testing.T) {
	_, srv, _ := newTestServer(t, 0)
	for _, q := range []string{"", "?node=x", "?node=-1", "?node=" + strconv.Itoa(testNodes)} {
		resp, err := http.Get(srv.URL + "/" + q)
		if err != nil {
			t.Fatalf("GET %q: %v", q, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("GET %q: status %d, want 400", q, resp.StatusCode)
		}
	}
}

func TestHandshakeAndBroadcast(t *testing.T) {
	h, srv, _ := newTestServer(t, 0)
	ctx := testCtx(t)
	a := connect(t, ctx, srv, 0)
	b := connect(t, ctx, srv, 2)

	msg, _ := protocol.Marshal(protocol.TypeTick, h.NextSeq(), 0, map[string]int{"n": 1})
	h.Broadcast(msg)
	for i, conn := range []*websocket.Conn{a, b} {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("client %d read: %v", i, err)
		}
		if string(data) != string(msg) {
			t.Fatalf("client %d: got %s want %s", i, data, msg)
		}
	}

	direct, _ := protocol.Marshal(protocol.TypeSceneChange, h.NextSeq(), 0, map[string]string{"scene": "x"})
	if !h.SendTo(2, direct) {
		t.Fatal("SendTo(2) returned false for a connected node")
	}
	if env := read(t, ctx, b); env.Type != protocol.TypeSceneChange {
		t.Fatalf("SendTo: got %q want %q", env.Type, protocol.TypeSceneChange)
	}
	if h.SendTo(1, direct) {
		t.Fatal("SendTo(1) returned true for a node that never connected")
	}
}

func TestBadHello(t *testing.T) {
	_, srv, _ := newTestServer(t, 0)
	ctx := testCtx(t)

	t.Run("wrong type", func(t *testing.T) {
		conn := dial(t, ctx, srv, 0)
		send(t, ctx, conn, protocol.TypePong, protocol.PongPayload{})
		wantClose(t, ctx, conn, websocket.StatusUnsupportedData)
	})
	t.Run("not json", func(t *testing.T) {
		conn := dial(t, ctx, srv, 0)
		if err := conn.Write(ctx, websocket.MessageText, []byte("hello?")); err != nil {
			t.Fatal(err)
		}
		wantClose(t, ctx, conn, websocket.StatusUnsupportedData)
	})
	t.Run("node id mismatch", func(t *testing.T) {
		conn := dial(t, ctx, srv, 0)
		sendHello(t, ctx, conn, 1)
		wantClose(t, ctx, conn, websocket.StatusPolicyViolation)
	})
}

func TestDuplicateNodeRejected(t *testing.T) {
	_, srv, _ := newTestServer(t, 0)
	ctx := testCtx(t)
	first := connect(t, ctx, srv, 1)

	dup := dial(t, ctx, srv, 1)
	sendHello(t, ctx, dup, 1)
	wantClose(t, ctx, dup, websocket.StatusCode(4001))

	// The first connection is unaffected.
	send(t, ctx, first, protocol.TypeVideoDuration, map[string]float64{"seconds": 1})
}

func TestReconnectAfterClose(t *testing.T) {
	_, srv, _ := newTestServer(t, 0)
	ctx := testCtx(t)
	conn := connect(t, ctx, srv, 0)
	if err := conn.Close(websocket.StatusNormalClosure, "bye"); err != nil {
		t.Fatalf("close: %v", err)
	}

	// The server frees the slot asynchronously; until then it answers 4001.
	for {
		c := dial(t, ctx, srv, 0)
		sendHello(t, ctx, c, 0)
		_, data, err := c.Read(ctx)
		if err == nil {
			env, _ := protocol.Unmarshal(data)
			if env.Type != protocol.TypeWelcome {
				t.Fatalf("reconnect: got %q want welcome", env.Type)
			}
			return
		}
		if websocket.CloseStatus(err) != websocket.StatusCode(4001) {
			t.Fatalf("reconnect: %v", err)
		}
		select {
		case <-ctx.Done():
			t.Fatal("slot never freed after close")
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func TestClientMessageForwarded(t *testing.T) {
	_, srv, msgs := newTestServer(t, 0)
	ctx := testCtx(t)
	conn := connect(t, ctx, srv, 0)
	send(t, ctx, conn, protocol.TypeVideoDuration, map[string]float64{"seconds": 12.5})

	select {
	case env := <-msgs:
		if env.Type != protocol.TypeVideoDuration || !strings.Contains(string(env.Payload), "12.5") {
			t.Fatalf("forwarded message: %+v", env)
		}
	case <-ctx.Done():
		t.Fatal("client message was not forwarded")
	}
}

func TestPingPongProducesNodeStats(t *testing.T) {
	_, srv, _ := newTestServer(t, 50*time.Millisecond)
	ctx := testCtx(t)
	conn := connect(t, ctx, srv, 0)

	env := read(t, ctx, conn)
	if env.Type != protocol.TypePing {
		t.Fatalf("got %q want ping", env.Type)
	}
	var ping protocol.PingPayload
	if err := protocol.UnmarshalPayload(env, &ping); err != nil || ping.Nonce == "" {
		t.Fatalf("ping payload: %+v (%v)", ping, err)
	}
	send(t, ctx, conn, protocol.TypePong, protocol.PongPayload{
		Nonce:          ping.Nonce,
		ServerTsMsEcho: env.ServerTsMs,
		ClientTsMs:     timing.EpochNowMs(),
	})

	for {
		env := read(t, ctx, conn)
		if env.Type == protocol.TypePing {
			continue // the next ping may arrive first
		}
		if env.Type != protocol.TypeNodeStats {
			t.Fatalf("got %q want node_stats", env.Type)
		}
		var stats protocol.NodeStatsPayload
		if err := protocol.UnmarshalPayload(env, &stats); err != nil {
			t.Fatalf("node_stats payload: %v", err)
		}
		if stats.RTTMedianMs < 0 || stats.RTTMedianMs > 1000 {
			t.Fatalf("implausible RTT: %+v", stats)
		}
		return
	}
}

func TestPongWatchdogClosesSilentNode(t *testing.T) {
	_, srv, _ := newTestServer(t, 20*time.Millisecond)
	ctx := testCtx(t)
	conn := connect(t, ctx, srv, 0)
	// Never answer pings: the server should give up after 3 intervals. The
	// watchdog cancels the node context before closing, which drops the TCP
	// connection, so the client may see EOF instead of the close frame.
	for {
		_, _, err := conn.Read(ctx)
		if err == nil {
			continue
		}
		if errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("watchdog never closed the connection")
		}
		if got := websocket.CloseStatus(err); got != -1 && got != websocket.StatusPolicyViolation {
			t.Fatalf("close status: got %v want %v or none (err: %v)", got, websocket.StatusPolicyViolation, err)
		}
		return
	}
}
