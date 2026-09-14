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
      r: 1.4 + Math.random() * 1.6,
      a: 0.35 + Math.random() * 0.35,
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

      // Inject keyframe animation. Palette is tuned for a back-projected
      // portrait rectangle — bright, high-contrast, glow-heavy so it survives
      // the loss in projection.
      const style = document.createElement('style');
      style.textContent = `
        @keyframes idle-pulse {
          0%,100% { color:#4a9fe6; text-shadow:0 0 10px rgba(120,180,255,0.35); }
          50%     { color:#ffffff; text-shadow:0 0 40px rgba(120,180,255,0.9), 0 0 8px rgba(255,255,255,0.6); }
        }
        .idle-ready { animation: idle-pulse 3.5s ease-in-out infinite; }
        #idle-safe {
          position:absolute;
          left:50%; top:50%;
          transform:translate(-50%,-50%);
          width:min(40vw, 1600px);
          height:min(80vh, 2400px);
          display:flex; flex-direction:column; align-items:center; justify-content:center;
          gap:6%;
          padding:6% 0;
          box-sizing:border-box;
        }
        #idle-safe .idle-stats-wrap {
          margin-top:auto;
        }
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

      // Central portrait safe-area wrapper — everything visible on the projected
      // surface lives inside this. Particles fill the full canvas behind.
      const safe = document.createElement('div');
      safe.id = 'idle-safe';
      safe.style.cssText = 'pointer-events:none;font-family:ui-monospace,"SF Mono",Menlo,monospace;';
      container.appendChild(safe);

      // NODE N — single line, never wraps. Letter-spacing is tight enough that
      // the whole label fits inside the portrait safe-area on any viewport.
      const nodeLabel = document.createElement('div');
      nodeLabel.textContent = `NODE ${ctx.nodeId}`;
      nodeLabel.style.cssText = [
        'width:100%', 'text-align:center', 'white-space:nowrap',
        'font-size:clamp(2.4rem,8vw,7rem)', 'letter-spacing:0.15em',
        'padding-left:0.15em', // compensate trailing letter-spacing so it visually centres
        'font-weight:700',
        'color:#e6f2ff', 'text-transform:uppercase',
        'text-shadow:0 0 22px rgba(120,180,255,0.65)',
        'line-height:1',
      ].join(';');
      safe.appendChild(nodeLabel);

      // SYSTEM READY
      const readyLabel = document.createElement('div');
      readyLabel.textContent = 'SYSTEM READY';
      readyLabel.className = 'idle-ready';
      readyLabel.style.cssText = [
        'width:100%', 'text-align:center', 'white-space:nowrap',
        'font-size:clamp(1.4rem,3.8vw,3rem)', 'letter-spacing:0.28em',
        'padding-left:0.28em', // compensate trailing letter-spacing so it visually centres
        'font-weight:600', 'text-transform:uppercase',
        'line-height:1',
        'margin-top:0.6em', // push down a smidge below NODE N
      ].join(';');
      state.readyEl = readyLabel;
      safe.appendChild(readyLabel);

      // Stats card — pinned to bottom of the safe area so it stays projected.
      const statsWrap = document.createElement('div');
      statsWrap.className = 'idle-stats-wrap';
      const statsCard = document.createElement('div');
      statsCard.style.cssText = [
        'padding:16px 24px',
        'background:rgba(9,13,18,0.85)', 'border:1px solid #3a5a80',
        'border-radius:8px', 'font-size:clamp(16px,1.8vw,26px)', 'line-height:1.7',
        'color:#c5d9ee', 'white-space:pre',
      ].join(';');
      statsCard.textContent = formatStats();
      state.statsEl = statsCard;
      statsWrap.appendChild(statsCard);
      safe.appendChild(statsWrap);

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
        pctx.fillStyle = `rgba(140,190,240,${p.a})`;
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
