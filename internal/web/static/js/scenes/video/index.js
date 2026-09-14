import * as wallClock from '../../shell/wallClock.js';

// Drift thresholds (seconds). Below the deadband we do nothing; between the
// deadband and the hard-seek limit we nudge playbackRate; above we hard-seek.
const DRIFT_DEADBAND_S = 0.03;
const DRIFT_HARD_SEEK_S = 0.25;
const DRIFT_CHECK_INTERVAL_MS = 500;

export default function VideoSceneFactory() {
  const state = {
    ctx: null,
    videoEl: null,
    container: null,
    stageEl: null,
    driftTimer: null,
    duration: 0,
    startedAt: null,
  };

  function expectedPosition() {
    if (!state.duration) return 0;
    const t = wallClock.wallClock();
    if (t < 0) return 0;
    return t % state.duration;
  }

  function correctDrift() {
    const v = state.videoEl;
    if (!v || !state.duration || v.paused || v.readyState < 2) return;
    const expected = expectedPosition();
    const drift = v.currentTime - expected;
    const mag = Math.abs(drift);
    if (mag > DRIFT_HARD_SEEK_S) {
      v.playbackRate = 1.0;
      v.currentTime = expected;
      return;
    }
    if (mag > DRIFT_DEADBAND_S) {
      // Ahead → slow down; behind → speed up. Clamp to a gentle range so
      // playback stays perceptually normal.
      const rate = 1 - Math.max(-0.1, Math.min(0.1, drift * 0.5));
      v.playbackRate = rate;
    } else if (v.playbackRate !== 1.0) {
      v.playbackRate = 1.0;
    }
  }

  return {
    name: 'video',

    async init(ctx) {
      state.ctx = ctx;
      const params = ctx.sceneParams || {};
      const url = params.url;
      const mode = params.mode === 'stretch' ? 'stretch' : 'mirror';
      if (!url) {
        console.warn('video scene: missing url in params', params);
        return;
      }

      // Hide the shared canvas and mount a full-viewport container to hold
      // the <video>. Teardown restores it.
      state.stageEl = document.getElementById('stage');
      if (state.stageEl) state.stageEl.style.display = 'none';

      const container = document.createElement('div');
      container.id = 'video-scene';
      container.style.cssText = 'position:fixed;inset:0;overflow:hidden;background:#000;z-index:1;';
      state.container = container;

      const v = document.createElement('video');
      v.src = url;
      v.muted = true;
      v.autoplay = false;
      v.playsInline = true;
      v.loop = true;
      v.preload = 'auto';
      v.controls = false;
      v.style.display = 'block';

      if (mode === 'stretch') {
        const { wallSize, perNodeSize, nodeRect } = ctx;
        const widthPct = (wallSize.w / perNodeSize.w) * 100;
        const heightPct = (wallSize.h / perNodeSize.h) * 100;
        const offsetXPct = (nodeRect.x / perNodeSize.w) * 100;
        const offsetYPct = (nodeRect.y / perNodeSize.h) * 100;
        v.style.position = 'absolute';
        v.style.width = widthPct + '%';
        v.style.height = heightPct + '%';
        v.style.left = '0';
        v.style.top = '0';
        v.style.transform = 'translate(' + (-offsetXPct) + '%, ' + (-offsetYPct) + '%)';
        v.style.objectFit = 'fill';
      } else {
        v.style.width = '100%';
        v.style.height = '100%';
        v.style.objectFit = 'cover';
      }

      container.appendChild(v);
      document.body.appendChild(container);
      state.videoEl = v;

      const startPlayback = () => {
        state.duration = v.duration || 0;
        if (state.duration > 0) {
          v.currentTime = expectedPosition();
          // Report natural duration to the server so the queue player can
          // advance at the true end of the video instead of a fallback timer.
          const videoID = params.video_id;
          if (videoID && ctx.client && typeof ctx.client.send === 'function') {
            ctx.client.send({
              type: 'video_duration',
              payload: { video_id: videoID, duration_s: state.duration }
            });
          }
        }
        const attempt = v.play();
        if (attempt && typeof attempt.catch === 'function') {
          attempt.catch(err => console.warn('video autoplay blocked:', err));
        }
        state.startedAt = Date.now();
      };
      if (v.readyState >= 1 && v.duration && !Number.isNaN(v.duration)) {
        startPlayback();
      } else {
        v.addEventListener('loadedmetadata', startPlayback, { once: true });
      }

      state.driftTimer = setInterval(correctDrift, DRIFT_CHECK_INTERVAL_MS);
    },

    tick(_t) { /* video renders itself; nothing per-frame */ },

    onMessage(_msg) { /* video scene ignores per-tick server messages */ },

    async teardown() {
      if (state.driftTimer) {
        clearInterval(state.driftTimer);
        state.driftTimer = null;
      }
      if (state.videoEl) {
        try { state.videoEl.pause(); } catch (_) {}
        state.videoEl.removeAttribute('src');
        try { state.videoEl.load(); } catch (_) {}
        state.videoEl = null;
      }
      if (state.container && state.container.parentNode) {
        state.container.parentNode.removeChild(state.container);
      }
      state.container = null;
      if (state.stageEl) {
        state.stageEl.style.display = '';
        state.stageEl = null;
      }
      state.duration = 0;
      state.startedAt = null;
    },
  };
}
