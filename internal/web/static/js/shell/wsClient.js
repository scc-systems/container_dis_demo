import * as wallClock from './wallClock.js';

const RECONNECT_MIN = 500;
const RECONNECT_MAX = 8000;

export class WsClient extends EventTarget {
  constructor(url, nodeId) {
    super();
    this.url = url;
    this.nodeId = nodeId;
    this.socket = null;
    this.backoff = RECONNECT_MIN;
    this.dead = false;
  }

  connect() {
    if (this.dead) return;
    const s = new WebSocket(this.url);
    this.socket = s;
    s.addEventListener('open', () => {
      this._sendHello();
      this.dispatchEvent(new CustomEvent('open'));
    });
    s.addEventListener('close', (ev) => {
      if (ev.code === 4001) {
        // Node slot is held by another tab — wait 5s then retry without touching backoff.
        this.dispatchEvent(new CustomEvent('node_taken'));
        if (!this.dead) setTimeout(() => this.connect(), 5000);
        return;
      }
      this.dispatchEvent(new CustomEvent('close'));
      if (!this.dead) {
        setTimeout(() => this.connect(), this.backoff);
        this.backoff = Math.min(RECONNECT_MAX, this.backoff * 2);
      }
    });
    s.addEventListener('error', () => {
      try { s.close(); } catch (_) {}
    });
    s.addEventListener('message', (ev) => this._onMessage(ev.data));
  }

  destroy() {
    this.dead = true;
    if (this.socket) {
      try { this.socket.close(); } catch (_) {}
    }
  }

  _sendHello() {
    this._send({
      type: 'hello',
      payload: {
        node_id: this.nodeId,
        client_ts_ms: Date.now(),
        user_agent: navigator.userAgent,
        screen: { w: window.innerWidth, h: window.innerHeight }
      }
    });
  }

  _send(obj) {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(obj));
    }
  }

  _onMessage(data) {
    let env;
    try { env = JSON.parse(data); } catch (_) { return; }
    const p = env.payload || {};
    switch (env.type) {
      case 'welcome':
        wallClock.setEpoch(p.scene_epoch_ms);
        this.dispatchEvent(new CustomEvent('welcome', { detail: p }));
        break;
      case 'ping':
        this._send({
          type: 'pong',
          payload: {
            nonce: p.nonce,
            server_ts_ms_echo: env.server_ts_ms,
            client_ts_ms: Date.now()
          }
        });
        // Approximate our own local RTT is unknown; the server tracks it.
        break;
      case 'scene_change':
        wallClock.setEpoch(p.epoch_ms);
        this.dispatchEvent(new CustomEvent('scene_change', { detail: p }));
        break;
      case 'tick':
        this.dispatchEvent(new CustomEvent('tick', { detail: p }));
        break;
      case 'node_stats':
        this.dispatchEvent(new CustomEvent('node_stats', { detail: p }));
        break;
    }
  }
}
