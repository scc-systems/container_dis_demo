import * as wallClock from './wallClock.js';
import { activeSceneName } from './sceneHost.js';

let el = null;
let visible = false;
let nodeId = 0;
let frames = 0;

export function init(id) {
  nodeId = id;
  el = document.getElementById('debug-hud');
  document.addEventListener('keydown', (e) => {
    if (e.key === 'd' || e.key === 'D') {
      visible = !visible;
      el.hidden = !visible;
    }
  });
}

export function update() {
  frames++;
  if (!visible || !el) return;
  const t = wallClock.wallClock();
  const t0 = wallClock.epochMs();
  const dpr = window.devicePixelRatio;
  const dprBad = dpr !== 1;
  const lines = [
    `node:     ${nodeId}`,
    `T0:       ${t0 !== null ? t0 : 'unset'}`,
    `t:        ${t.toFixed(3)}s`,
    `frames:   ${frames}`,
    `scene:    ${activeSceneName() || 'none'}`,
    `rtt:      ${wallClock.getRtt().toFixed(1)}ms (server-side)`,
    `offset:   ${wallClock.getOffset().toFixed(1)}ms`,
    `dpr:      1 (locked)`,
    dprBad ? `<span class="err">devicePixelRatio=${dpr} FIX --force-device-scale-factor=1</span>`
           : `devicePixelRatio: ${dpr}`,
    `viewport: ${window.innerWidth}x${window.innerHeight}`,
  ];
  el.innerHTML = lines.join('\n');
}
