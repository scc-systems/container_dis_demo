const PARTICLE_COUNT = 80;
const STATS_DOM_INTERVAL = 1000;

function initParticles(W, H) {
  const out = [];
  for (let i = 0; i < PARTICLE_COUNT; i++) {
    const speed = 0.12 + Math.random() * 0.18;
    const angle = Math.random() * Math.PI * 2;
    out.push({
      x: Math.random() * W,
      y: Math.random() * H,
      vx: Math.cos(angle) * speed,
      vy: Math.sin(angle) * speed,
      r: 0.8 + Math.random() * 1.4,
      a: 0.12 + Math.random() * 0.25,
    });
  }
  return out;
}

export default function IdleSceneFactory() {
  const state = {
    container: null,
    stageEl: null,
    pcanvas: null,
    pctx: null,
    particles: [],
    statsEl: null,
    readyEl: null,
    stats: null,
    lastStatsAt: null,
    lastStatsDom: 0,
    connected: true,
    styleEl: null,
    client: null,
    onOpen: null,
    onClose: null,
    onNodeTaken: null,
  };

  function formatStats() {
    const s = state.stats;
    const ago = state.lastStatsAt ? Math.round(Date.now() - state.lastStatsAt) : '–';
    return [
      `state      ${state.connected ? 'connected' : 'offline'}`,
      `rtt med    ${s ? s.rtt_med_ms.toFixed(1) + ' ms' : '–'}`,
      `rtt p95    ${s ? s.rtt_p95_ms.toFixed(1) + ' ms' : '–'}`,
      `offset     ${s ? s.offset_ms.toFixed(1) + ' ms' : '–'}`,
      `last pong  ${s ? ago + ' ms' : '–'}`,
      `drops      ${s ? s.drops : '–'}`,
    ].join('\n');
  }

  return {
    name: 'idle',

    async init(ctx) {
      state.stageEl = document.getElementById('stage');
      if (state.stageEl) state.stageEl.style.display = 'none';

      // Inject keyframe animation
      const style = document.createElement('style');
      style.textContent = `
        @keyframes idle-pulse {
          0%,100% { color:#0d2a40; text-shadow:none; }
          50% { color:#2a7abf; text-shadow:0 0 30px rgba(42,122,191,0.5); }
        }
        .idle-ready { animation: idle-pulse 3.5s ease-in-out infinite; }
      `;
      document.head.appendChild(style);
      state.styleEl = style;

      const container = document.createElement('div');
      container.style.cssText = 'position:fixed;inset:0;overflow:hidden;background:#000;z-index:1;';
      state.container = container;

      // Particle canvas
      const pcanvas = document.createElement('canvas');
      pcanvas.width = window.innerWidth;
      pcanvas.height = window.innerHeight;
      pcanvas.style.cssText = 'position:absolute;inset:0;width:100%;height:100%;';
      state.pcanvas = pcanvas;
      state.pctx = pcanvas.getContext('2d');
      state.particles = initParticles(pcanvas.width, pcanvas.height);
      container.appendChild(pcanvas);

      // Text overlay
      const overlay = document.createElement('div');
      overlay.style.cssText = 'position:absolute;inset:0;pointer-events:none;font-family:ui-monospace,"SF Mono",Menlo,monospace;';
      container.appendChild(overlay);

      // NODE N
      const nodeLabel = document.createElement('div');
      nodeLabel.textContent = `NODE ${ctx.nodeId}`;
      nodeLabel.style.cssText = [
        'position:absolute', 'top:14%', 'width:100%', 'text-align:center',
        'font-size:clamp(2rem,6vw,4rem)', 'letter-spacing:0.45em',
        'color:#0a2030', 'text-transform:uppercase',
      ].join(';');
      overlay.appendChild(nodeLabel);

      // SYSTEM READY
      const readyLabel = document.createElement('div');
      readyLabel.textContent = 'SYSTEM READY';
      readyLabel.className = 'idle-ready';
      readyLabel.style.cssText = [
        'position:absolute', 'top:50%', 'width:100%', 'text-align:center',
        'transform:translateY(-50%)',
        'font-size:clamp(1rem,2.5vw,1.6rem)', 'letter-spacing:0.35em',
        'text-transform:uppercase',
      ].join(';');
      state.readyEl = readyLabel;
      overlay.appendChild(readyLabel);

      // Stats card
      const statsCard = document.createElement('div');
      statsCard.style.cssText = [
        'position:absolute', 'bottom:16px', 'left:16px',
        'padding:10px 14px',
        'background:rgba(9,13,18,0.75)', 'border:1px solid #1a2535',
        'border-radius:6px', 'font-size:11px', 'line-height:1.8',
        'color:#2a5070', 'white-space:pre',
      ].join(';');
      statsCard.textContent = formatStats();
      state.statsEl = statsCard;
      overlay.appendChild(statsCard);

      document.body.appendChild(container);

      // Track connection state via client events
      state.client = ctx.client;
      state.connected = true;
      state.onOpen = () => { state.connected = true; };
      state.onClose = () => { state.connected = false; };
      state.onNodeTaken = () => { state.connected = false; };
      ctx.client.addEventListener('open',       state.onOpen);
      ctx.client.addEventListener('close',      state.onClose);
      ctx.client.addEventListener('node_taken', state.onNodeTaken);
    },

    tick(_t) {
      const { pctx, pcanvas, particles } = state;
      if (!pctx) return;
      const W = pcanvas.width, H = pcanvas.height;

      pctx.clearRect(0, 0, W, H);
      for (const p of particles) {
        p.x += p.vx;
        p.y += p.vy;
        if (p.x < 0)  { p.vx =  Math.abs(p.vx); p.x = 0; }
        if (p.x > W)  { p.vx = -Math.abs(p.vx); p.x = W; }
        if (p.y < 0)  { p.vy =  Math.abs(p.vy); p.y = 0; }
        if (p.y > H)  { p.vy = -Math.abs(p.vy); p.y = H; }
        pctx.beginPath();
        pctx.arc(p.x, p.y, p.r, 0, Math.PI * 2);
        pctx.fillStyle = `rgba(50,130,200,${p.a})`;
        pctx.fill();
      }

      // Throttled DOM stats update
      const now = Date.now();
      if (state.statsEl && now - state.lastStatsDom > STATS_DOM_INTERVAL) {
        state.lastStatsDom = now;
        state.statsEl.textContent = formatStats();
      }
    },

    onMessage(msg) {
      if (msg.type === 'node_stats') {
        state.stats = msg.payload;
        state.lastStatsAt = Date.now();
      }
    },

    async teardown() {
      if (state.container?.parentNode) {
        state.container.parentNode.removeChild(state.container);
      }
      state.container = null;
      if (state.stageEl) {
        state.stageEl.style.display = '';
        state.stageEl = null;
      }
      if (state.styleEl) {
        state.styleEl.remove();
        state.styleEl = null;
      }
      if (state.client) {
        state.client.removeEventListener('open',       state.onOpen);
        state.client.removeEventListener('close',      state.onClose);
        state.client.removeEventListener('node_taken', state.onNodeTaken);
        state.client = null;
      }
    },
  };
}
