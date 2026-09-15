const registry = new Map();
let active = null;
let activeName = null;
let pendingScene = null; // { name, epochMs, params, ctx, fadeStarted }
let mountingLock = false;

// Synchronized dip-to-black between scenes. Because every node cuts over at the
// same server epoch, running this fade on the same timeline on all 5 walls
// gives a lockstep transition — no protocol change required. Must be strictly
// less than the server's SceneSwitchLead (currently 2000ms) so the overlay is
// fully opaque when the cut happens; ~200ms of buffer covers WS latency + the
// first rAF after the scene_change arrives.
const FADE_MS = 1800;
let overlay = null;

function ensureOverlay() {
  if (overlay) return overlay;
  overlay = document.createElement('div');
  overlay.id = 'scene-fade';
  overlay.style.cssText = [
    'position:fixed', 'inset:0', 'background:#000',
    'opacity:0', 'pointer-events:none',
    'z-index:9998',
    `transition:opacity ${FADE_MS}ms ease-in-out`,
  ].join(';');
  document.body.appendChild(overlay);
  return overlay;
}

export function register(name, factory) {
  registry.set(name, factory);
}

export function hasActive() {
  return active !== null;
}

export function activeSceneName() {
  return activeName;
}

async function mount(name, ctx) {
  if (!registry.has(name)) {
    console.warn('unknown scene', name);
    return;
  }
  if (mountingLock) return;
  mountingLock = true;
  try {
    if (active && active.teardown) {
      try { await active.teardown(); } catch (e) { console.warn('teardown', e); }
    }
    const factory = registry.get(name);
    const scene = factory();
    await scene.init(ctx);
    active = scene;
    activeName = name;
  } finally {
    mountingLock = false;
  }
}

// Request a scene, honouring an epoch: if epochMs is in the future, delay mount.
export function requestScene(name, epochMs, params, ctx) {
  pendingScene = { name, epochMs, params, ctx, fadeStarted: false };
}

// Called each rAF frame from main.js. Returns nothing.
export function tick(t) {
  if (pendingScene) {
    const nowMs = Date.now();
    const hasEpoch = pendingScene.epochMs !== undefined;
    const untilMs = hasEpoch ? (pendingScene.epochMs - nowMs) : 0;

    // Fade-in phase: ramp the overlay to black over FADE_MS ms before the cut.
    // Skip on the very first mount — there is no prior scene to fade away from.
    if (hasEpoch && active && !pendingScene.fadeStarted && untilMs <= FADE_MS) {
      ensureOverlay().style.opacity = '1';
      pendingScene.fadeStarted = true;
    }

    if (!hasEpoch || untilMs <= 0) {
      const p = pendingScene;
      pendingScene = null;
      const ctx = { ...p.ctx, sceneParams: p.params };
      mount(p.name, ctx).then(() => {
        if (p.fadeStarted && overlay) {
          // Two rAFs so the newly-mounted DOM has painted at least one frame
          // fully behind the black overlay before we start fading it back out.
          requestAnimationFrame(() => requestAnimationFrame(() => {
            overlay.style.opacity = '0';
          }));
        }
      });
      return; // active may not exist yet — next frame will render
    }
  }
  if (active && typeof active.tick === 'function') {
    active.tick(t);
  }
}

export function onMessage(msg) {
  if (active && typeof active.onMessage === 'function') {
    active.onMessage(msg);
  }
}
