import { wsUrl, LOCKED_DPR } from './shell/config.js';
import { WsClient } from './shell/wsClient.js';
import * as wallClock from './shell/wallClock.js';
import * as sceneHost from './shell/sceneHost.js';
import * as geom from './shell/wallGeometry.js';
import * as debugHud from './shell/debugHud.js';
import ThreeUnified from './scenes/threeUnified/index.js';
import SolarPanorama from './scenes/solarPanorama/index.js';
import WarpJump from './scenes/warpJump/index.js';
import VideoScene from './scenes/video/index.js';
import ImageScene from './scenes/image/index.js';
import IdleScene from './scenes/idle/index.js';

const params = new URLSearchParams(location.search);
const nodeIdRaw = params.get('node');
const nodeId = Number.parseInt(nodeIdRaw, 10);
if (!Number.isInteger(nodeId) || nodeId < 0 || nodeId >= 32) {
  document.body.innerHTML = `<div style="color:#f87171;padding:40px;font-size:24px">Bad or missing ?node=&lt;id&gt;</div>`;
  throw new Error('invalid node id');
}

sceneHost.register('three-unified',   ThreeUnified);
sceneHost.register('solar-panorama',  SolarPanorama);
sceneHost.register('warp-jump',       WarpJump);
sceneHost.register('idle',            IdleScene);
sceneHost.register('video',           VideoScene);
sceneHost.register('image',           ImageScene);

const splash = document.getElementById('splash');
debugHud.init(nodeId);

// Wire up WS.
const client = new WsClient(wsUrl(nodeId), nodeId);

function replaceStageCanvas() {
  const old = document.getElementById('stage');
  const fresh = document.createElement('canvas');
  fresh.id = 'stage';
  if (old && old.parentNode) old.parentNode.replaceChild(fresh, old);
  else document.body.insertBefore(fresh, document.body.firstChild);
  return fresh;
}

let sceneCtxCache = null;

function buildCtx(seed) {
  const nodeRect = geom.nodeRect(nodeId);
  const wallSize = geom.wallSize();
  const perNode = geom.perNodeSize();
  return {
    canvas: replaceStageCanvas(),
    nodeId,
    nodeRect,
    wallSize,
    perNodeSize: perNode,
    dpr: LOCKED_DPR,
    seed,
    client,
    config: {
      lockedDpr: LOCKED_DPR
    }
  };
}

client.addEventListener('node_taken', () => {
  splash.classList.remove('hidden');
  splash.classList.add('conflict');
  splash.textContent = 'Node taken — waiting';
});

client.addEventListener('welcome', (ev) => {
  const p = ev.detail;
  geom.updateFromWelcome(p);
  sceneCtxCache = buildCtx(p.seed);
  client.backoff = 500; // reset backoff now that we have a good connection
  splash.classList.remove('conflict');
  splash.textContent = 'Connecting…';
  splash.classList.add('hidden');
  sceneHost.requestScene(p.current_scene, p.scene_epoch_ms, { seed: p.seed }, sceneCtxCache);
});

client.addEventListener('scene_change', (ev) => {
  const p = ev.detail;
  const seed = p.params && p.params.seed ? p.params.seed : (sceneCtxCache ? sceneCtxCache.seed : 0);
  sceneCtxCache = buildCtx(seed);
  sceneHost.requestScene(p.scene, p.epoch_ms, p.params || {}, sceneCtxCache);
});

client.addEventListener('tick', (ev) => {
  sceneHost.onMessage({ type: 'tick', payload: ev.detail });
});

client.addEventListener('node_stats', (ev) => {
  sceneHost.onMessage({ type: 'node_stats', payload: ev.detail });
});

client.connect();

function raf() {
  const t = wallClock.wallClock();
  sceneHost.tick(t);
  debugHud.update();
  requestAnimationFrame(raf);
}
requestAnimationFrame(raf);
