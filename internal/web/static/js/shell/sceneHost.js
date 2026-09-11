const registry = new Map();
let active = null;
let activeName = null;
let pendingScene = null; // { name, epochMs, params }
let mountingLock = false;

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
  pendingScene = { name, epochMs, params, ctx };
}

// Called each rAF frame from main.js. Returns nothing.
export function tick(t) {
  if (pendingScene) {
    const nowMs = Date.now();
    if (pendingScene.epochMs === undefined || nowMs >= pendingScene.epochMs) {
      const p = pendingScene;
      pendingScene = null;
      const ctx = { ...p.ctx, sceneParams: p.params };
      mount(p.name, ctx);
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
