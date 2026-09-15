package admin

import (
	"html/template"
	"net/http"
)

var adminTmpl = template.Must(template.New("admin").Parse(`<!doctype html>
<html><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="base-path" content="{{.BasePath}}">
<title>wallsync · admin</title>
<style>
  body { background:#0b0e12; color:#dfe6ef; font-family:system-ui,sans-serif; margin:0; padding:20px; }
  h1 { font-size:18px; margin:0 0 16px; letter-spacing:0.03em; }
  #status-bar { display:flex; align-items:center; gap:12px; margin-bottom:16px; flex-wrap:wrap; }
  #status-bar .now { font-family:ui-monospace,monospace; font-size:13px; }
  #countdown { color:#9ba7b6; font-family:ui-monospace,monospace; font-size:12px; }
  #cards { display:grid; grid-template-columns:repeat(auto-fit, minmax(200px, 1fr)); gap:12px; margin:0 0 16px; }
  #now-playing { display:flex; align-items:center; gap:12px; background:#141a22; border:1px solid #212a35; border-radius:8px; padding:12px 14px; margin:0 0 20px; flex-wrap:wrap; }
  #now-playing .np-label { flex:1; min-width:0; font-family:ui-monospace,monospace; font-size:13px; color:#dfe6ef; overflow-wrap:anywhere; }
  #now-playing .np-label .np-tag { color:#6b7684; letter-spacing:0.08em; text-transform:uppercase; font-size:11px; margin-right:8px; }
  #now-playing .np-buttons { display:flex; gap:6px; flex-wrap:wrap; }
  #now-playing button { font-size:13px; }
  #now-playing button.primary { background:#0f2a18; border-color:#2a6a35; color:#4ade80; }
  #now-playing button.primary:hover:not(:disabled) { background:#134a25; }
  #queue-strip { display:flex; gap:6px; margin:0 0 20px; overflow-x:auto; padding-bottom:4px; }
  #queue-strip.empty { display:none; }
  .queue-item { flex-shrink:0; background:#141a22; border:1px solid #212a35; border-radius:6px; padding:6px 10px; font-size:12px; display:flex; align-items:center; gap:8px; }
  .queue-item.current { border-color:#2a6a35; box-shadow:0 0 8px rgba(74,222,128,0.15); }
  .queue-item .qi-kind { color:#6b7684; font-size:10px; text-transform:uppercase; letter-spacing:0.08em; }
  .queue-item .qi-label { font-family:ui-monospace,monospace; min-width:0; overflow-wrap:anywhere; }
  .queue-item .qi-dur { color:#9ba7b6; font-family:ui-monospace,monospace; font-size:11px; }
  .queue-item .qi-remove { background:none; border:none; color:#6b7684; padding:0 4px; font-size:14px; cursor:pointer; }
  .queue-item .qi-remove:hover { color:#f87171; }
  .card { background:#141a22; border:1px solid #212a35; border-radius:8px; padding:12px; }
  .card h2 { margin:0 0 8px; font-size:14px; letter-spacing:0.05em; }
  .row { display:flex; justify-content:space-between; gap:8px; font-size:12px; padding:2px 0; font-family:ui-monospace,monospace; }
  .row > span:first-child { flex-shrink:0; }
  .playing-row { align-items:flex-start; }
  .playing-val { text-align:right; min-width:0; overflow-wrap:anywhere; }
  .dot { display:inline-block; width:10px; height:10px; border-radius:50%; margin-right:6px; vertical-align:middle; }
  .green { background:#4ade80; box-shadow:0 0 8px #4ade80aa; }
  .amber { background:#fbbf24; box-shadow:0 0 8px #fbbf24aa; }
  .red   { background:#f87171; box-shadow:0 0 8px #f87171aa; }
  .warn  { color:#fbbf24; }
  .err   { color:#f87171; }
  button { background:#1f2a38; color:#dfe6ef; border:1px solid #2c3a4a; padding:6px 14px; border-radius:6px; cursor:pointer; font:inherit; font-size:13px; }
  button:hover { background:#26344a; }
  button:disabled, button:disabled:hover { background:#1a222c; color:#4a5866; border-color:#232d3a; cursor:not-allowed; opacity:0.6; }
  #autorotate-btn.on { background:#0f2a18; border-color:#2a6a35; color:#4ade80; }
  #content { }
  #tab-bar { display:flex; gap:6px; margin-bottom:16px; border-bottom:1px solid #1a2535; padding-bottom:10px; }
  .tab { background:transparent; color:#6b7684; border:1px solid transparent; border-radius:4px; padding:5px 14px; font-size:12px; letter-spacing:0.05em; }
  .tab:hover { color:#dfe6ef; }
  .tab.active { background:#141a22; color:#dfe6ef; border-color:#212a35; }
  .tab-panel { display:none; }
  .tab-panel.active { display:block; }
  #scene-grid { display:flex; flex-wrap:wrap; gap:8px; padding:4px 0 16px; }
  .scene-btn { background:#1f2a38; color:#6b7684; border:1px solid #2c3a4a; padding:10px 22px; border-radius:6px; font-size:13px; letter-spacing:0.03em; transition:background 0.12s,color 0.12s,border-color 0.12s; }
  .scene-btn:hover { background:#26344a; color:#dfe6ef; }
  .scene-btn.active { background:#0f2a18; border-color:#2a6a35; color:#4ade80; box-shadow:0 0 10px rgba(74,222,128,0.12); }
  #image-list, #video-list { display:flex; flex-direction:column; gap:6px; margin:8px 0; }
  .media-row { display:flex; align-items:center; gap:8px; background:#141a22; border:1px solid #212a35; border-radius:6px; padding:8px 10px; font-size:13px; }
  .media-row .name { flex:1; min-width:0; font-family:ui-monospace,monospace; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
  .media-row .size { color:#9ba7b6; font-family:ui-monospace,monospace; font-size:11px; flex-shrink:0; }
  .row-buttons { display:flex; gap:6px; flex-shrink:0; flex-wrap:wrap; }
  .danger { background:#3a1e1e; border-color:#5a2c2c; }
  .danger:hover { background:#4a2828; }
  .upload-row { display:flex; align-items:center; gap:8px; margin-top:12px; }
  .upload-status { color:#9ba7b6; font-family:ui-monospace,monospace; font-size:12px; }
  input[type=file] { color:#dfe6ef; font:inherit; }
  input[type=text], input[type=number] { background:#0b0e12; color:#dfe6ef; border:1px solid #2c3a4a; border-radius:4px; padding:6px 8px; font:inherit; font-size:13px; }
  select { background:#0b0e12; color:#dfe6ef; border:1px solid #2c3a4a; border-radius:4px; padding:6px 8px; font:inherit; font-size:13px; }

  #playlist-list { display:flex; flex-direction:column; gap:6px; }
  .playlist-row { display:flex; align-items:center; gap:8px; background:#141a22; border:1px solid #212a35; border-radius:6px; padding:10px; font-size:13px; }
  .playlist-row .pl-name { flex:1; font-family:ui-monospace,monospace; }
  .playlist-row .pl-meta { color:#9ba7b6; font-family:ui-monospace,monospace; font-size:11px; }

  .playlist-editor { background:#141a22; border:1px solid #212a35; border-radius:8px; padding:14px; margin-top:14px; display:flex; flex-direction:column; gap:12px; }
  .playlist-editor .pe-row { display:flex; gap:8px; align-items:center; flex-wrap:wrap; }
  .playlist-editor .pe-row label { color:#9ba7b6; font-size:12px; letter-spacing:0.05em; }
  .playlist-editor .pe-items { display:flex; flex-direction:column; gap:6px; }
  .playlist-editor .pe-item { display:flex; gap:6px; align-items:center; background:#0b0e12; border:1px solid #212a35; border-radius:6px; padding:8px; flex-wrap:wrap; }
  .playlist-editor .pe-item .pe-idx { color:#6b7684; font-family:ui-monospace,monospace; font-size:11px; width:28px; text-align:right; }
  .playlist-editor .pe-item .pe-label { flex:1; min-width:0; font-family:ui-monospace,monospace; font-size:12px; overflow-wrap:anywhere; }
  .playlist-editor .pe-item input[type=number] { width:72px; }
  .playlist-editor .pe-actions { display:flex; gap:6px; align-items:center; flex-wrap:wrap; }

  @media (max-width: 640px) {
    body { padding:12px; }
    h1 { font-size:16px; }
    #status-bar { gap:8px; }
    #tab-bar { overflow-x:auto; -webkit-overflow-scrolling:touch; white-space:nowrap; flex-wrap:nowrap; }
    .tab { flex-shrink:0; padding:8px 14px; font-size:12px; }
    .tab, .scene-btn, button { min-height:40px; }
    .scene-btn { min-height:44px; padding:10px 16px; }
    .media-row { flex-direction:column; align-items:stretch; gap:6px; padding:10px; }
    .media-row .name { white-space:normal; overflow:visible; text-overflow:clip; }
    .media-row .size { align-self:flex-start; }
    .row-buttons { flex-wrap:wrap; }
    .media-row button, .playlist-row button { flex:1 1 auto; }
    .upload-row { flex-wrap:wrap; gap:8px; }
    .upload-row input[type=file] { flex:1 1 100%; }
    #now-playing { flex-direction:column; align-items:stretch; gap:10px; }
    #now-playing .np-label { text-align:center; white-space:normal; }
    #now-playing .np-buttons { justify-content:center; }
    #now-playing button { flex:0 1 auto; min-height:44px; }
    #now-playing .np-buttons { flex-wrap:wrap; justify-content:stretch; }
    #now-playing .np-buttons button { flex:1 1 40%; }
    #queue-strip { flex-direction:column; overflow-x:visible; }
    .queue-item { justify-content:space-between; }
    .playlist-editor input, .playlist-editor select { width:100%; box-sizing:border-box; }
  }
</style>
</head><body>
  <h1>wallsync</h1>
  <div id="status-bar">
    <span class="now">Scene: <b id="scene-name">–</b></span>
    <span id="countdown"></span>
    <button id="autorotate-btn">auto-rotate off</button>
  </div>
  <div id="cards"></div>
  <div id="now-playing">
    <div class="np-buttons">
      <button id="np-play"    class="primary" disabled title="no queue loaded">▶ play</button>
      <button id="np-pause"   disabled hidden title="pause on current item">⏸ pause</button>
      <button id="np-restart" disabled title="nothing playing">↻ restart</button>
      <button id="np-skip"    disabled hidden title="skip to next queue item">⏭ next</button>
      <button id="np-stop"    class="danger" disabled hidden title="stop queue">⏹ stop</button>
    </div>
    <div class="np-label"><span class="np-tag">now playing</span><span id="np-current">–</span></div>
  </div>
  <div id="queue-strip" class="empty"></div>
  <div id="content">
    <div id="tab-bar">
      <button class="tab active" data-tab="scenes">scenes</button>
      <button class="tab" data-tab="images">images</button>
      <button class="tab" data-tab="videos">videos</button>
      <button class="tab" data-tab="playlists">playlists</button>
    </div>
    <div id="tab-scenes" class="tab-panel active">
      <div id="scene-grid"></div>
    </div>
    <div id="tab-images" class="tab-panel">
      <div id="image-list"></div>
      <form id="img-upload-form">
        <div class="upload-row">
          <input type="file" name="image" id="img-upload-input" accept="image/jpeg,image/png,image/webp,image/gif,.jpg,.jpeg,.png,.webp,.gif" required>
          <button type="submit">upload image</button>
          <span id="img-upload-status" class="upload-status"></span>
        </div>
      </form>
    </div>
    <div id="tab-videos" class="tab-panel">
      <div id="video-list"></div>
      <form id="upload-form">
        <div class="upload-row">
          <input type="file" name="video" id="upload-input" accept="video/mp4,video/webm,video/quicktime,.mp4,.webm,.mov" required>
          <button type="submit">upload video</button>
          <span id="upload-status" class="upload-status"></span>
        </div>
      </form>
    </div>
    <div id="tab-playlists" class="tab-panel">
      <div id="playlist-list"></div>
      <div class="upload-row" style="margin-top:12px">
        <button id="pl-new-btn">+ new playlist</button>
        <button id="pl-save-queue-btn" title="save current queue as a new playlist">save current queue</button>
      </div>
      <div id="pl-editor" class="playlist-editor" hidden></div>
    </div>
  </div>
<script type="module">
const BASE = document.querySelector('meta[name="base-path"]')?.content || '';

// ── Node cards ──────────────────────────────────────────────────────────────
const cards      = document.getElementById('cards');
const sceneName  = document.getElementById('scene-name');
const countdown  = document.getElementById('countdown');

function classifyNode(n) {
  if (!n.connected) return 'red';
  if (Math.abs(n.offset_ms) > 25 || n.rtt_p95_ms > 50) return 'amber';
  return 'green';
}

function render(status) {
  sceneName.textContent = status.scene;
  const until = status.epoch_ms - status.now_ms;
  countdown.textContent = until > 0 ? 'epoch in ' + until + 'ms' : 't = ' + (-until) + 'ms';
  autoRotateOn = !!status.auto_rotate;
  updateAutoRotateBtn();
  renderSceneGrid(status.scene);
  renderNowPlaying(status);
  renderQueueStrip(status.queue);
  const playingLabel = status.playing || status.scene || '–';
  cards.innerHTML = status.nodes.map(n => {
    const cls = classifyNode(n);
    return "<div class='card'>" +
      "<h2><span class='dot " + cls + "'></span>node " + n.id + "</h2>" +
      "<div class='row'><span>state</span><b>" + (n.connected ? 'connected' : 'offline') + "</b></div>" +
      "<div class='row playing-row'><span>playing</span><b class='playing-val' title='" + escapeHtml(playingLabel) + "'>" + escapeHtml(playingLabel) + "</b></div>" +
      "<div class='row'><span>rtt med</span><b>" + (n.rtt_med_ms||0).toFixed(1) + " ms</b></div>" +
      "<div class='row'><span>rtt p95</span><b>" + (n.rtt_p95_ms||0).toFixed(1) + " ms</b></div>" +
      "<div class='row'><span>offset</span><b class='" + (Math.abs(n.offset_ms)>25?'warn':'') + "'>" + (n.offset_ms||0).toFixed(1) + " ms</b></div>" +
      "<div class='row'><span>last pong</span><b>" + (n.last_pong_ms_ago||0) + " ms</b></div>" +
      "<div class='row'><span>drops</span><b>" + (n.drops||0) + "</b></div>" +
    "</div>";
  }).join('');
}

async function pollOnce() {
  try {
    const r = await fetch(BASE + '/admin/status');
    render(await r.json());
  } catch(e) {}
}
setInterval(pollOnce, 500);
pollOnce();

// ── Now Playing ──────────────────────────────────────────────────────────────
const npCurrent = document.getElementById('np-current');
const npPlay    = document.getElementById('np-play');
const npPause   = document.getElementById('np-pause');
const npRestart = document.getElementById('np-restart');
const npSkip    = document.getElementById('np-skip');
const npStop    = document.getElementById('np-stop');

function renderNowPlaying(status) {
  const label = status.playing || status.scene || '–';
  npCurrent.textContent = label;

  const isIdle = status.scene === 'idle';
  npRestart.disabled = isIdle;
  npRestart.title = isIdle ? 'nothing playing' : 'restart current scene';

  const queue = status.queue;
  const hasQueue = queue && Array.isArray(queue.items) && queue.items.length > 0;
  const playing = hasQueue && queue.playing;
  const paused  = hasQueue && queue.paused;

  // Play button doubles as resume when paused; hide it while actively playing.
  npPlay.hidden = playing;
  npPlay.disabled = !hasQueue;
  npPlay.textContent = paused ? '▶ resume' : '▶ play';
  npPlay.title = hasQueue
    ? (paused ? 'resume on current item' : 'start queue')
    : 'no queue loaded';

  // Pause button only shown while actively playing.
  npPause.hidden = !playing;
  npPause.disabled = !playing;

  npSkip.hidden = !hasQueue;
  npStop.hidden = !hasQueue;
  npSkip.disabled = !hasQueue;
  npStop.disabled = !hasQueue;
}

npRestart.addEventListener('click', async () => {
  if (npRestart.disabled) return;
  try { await fetch(BASE + '/admin/restart', {method:'POST'}); } catch(e) {}
});
npPlay.addEventListener('click', async () => {
  if (npPlay.disabled) return;
  try { await fetch(BASE + '/admin/queue/play', {method:'POST'}); } catch(e) {}
});
npPause.addEventListener('click', async () => {
  if (npPause.disabled) return;
  try { await fetch(BASE + '/admin/queue/pause', {method:'POST'}); } catch(e) {}
});
npSkip.addEventListener('click', async () => {
  if (npSkip.disabled) return;
  try { await fetch(BASE + '/admin/queue/skip', {method:'POST'}); } catch(e) {}
});
npStop.addEventListener('click', async () => {
  if (npStop.disabled) return;
  try { await fetch(BASE + '/admin/queue/stop', {method:'POST'}); } catch(e) {}
});

// ── Queue strip ──────────────────────────────────────────────────────────────
const queueStrip = document.getElementById('queue-strip');

function renderQueueStrip(queue) {
  if (!queue || !queue.items || !queue.items.length) {
    queueStrip.classList.add('empty');
    queueStrip.innerHTML = '';
    return;
  }
  queueStrip.classList.remove('empty');
  queueStrip.innerHTML = queue.items.map((it, i) => {
    const current = i === queue.index && (queue.playing || queue.paused);
    const durLabel = (it.kind === 'video' && !it.duration_s) ? 'end' : (it.duration_s || 0) + 's';
    return "<div class='queue-item" + (current ? ' current' : '') + "' data-index='" + i + "'>" +
      "<span class='qi-kind'>" + escapeHtml(it.kind) + "</span>" +
      "<span class='qi-label'>" + escapeHtml(it.label || '') + "</span>" +
      "<span class='qi-dur'>" + durLabel + "</span>" +
      "<button class='qi-remove' data-remove='1' title='remove'>×</button>" +
    "</div>";
  }).join('');
}

queueStrip.addEventListener('click', async (e) => {
  const btn = e.target.closest('[data-remove]');
  if (!btn) return;
  const item = e.target.closest('.queue-item');
  const idx = item?.dataset.index;
  if (idx == null) return;
  try { await fetch(BASE + '/admin/queue/items/' + idx, {method:'DELETE'}); } catch(err) {}
});

// ── Auto-rotate ──────────────────────────────────────────────────────────────
let autoRotateOn = false;
const autorotateBtn = document.getElementById('autorotate-btn');

function updateAutoRotateBtn() {
  autorotateBtn.textContent = autoRotateOn ? 'auto-rotate on' : 'auto-rotate off';
  autorotateBtn.classList.toggle('on', autoRotateOn);
}

autorotateBtn.addEventListener('click', async () => {
  try {
    const r = await fetch(BASE + '/admin/autorotate', { method: 'POST' });
    const j = await r.json();
    autoRotateOn = j.auto_rotate;
    updateAutoRotateBtn();
  } catch(e) {}
});

// ── Scene grid ───────────────────────────────────────────────────────────────
const SCENES = [
  { id: 'idle',           label: 'idle' },
  { id: 'three-unified',  label: 'three-unified' },
  { id: 'solar-panorama', label: 'solar-panorama' },
  { id: 'warp-jump',      label: 'warp-jump' },
];
const sceneGrid = document.getElementById('scene-grid');

function renderSceneGrid(current) {
  sceneGrid.innerHTML = SCENES.map(s =>
    "<button class='scene-btn" + (s.id === current ? ' active' : '') + "' data-scene='" + s.id + "'>" + s.label + "</button>"
  ).join('');
}

sceneGrid.addEventListener('click', async (e) => {
  const btn = e.target.closest('.scene-btn');
  if (!btn) return;
  await fetch(BASE + '/admin/scene', {
    method: 'POST',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify({scene: btn.dataset.scene})
  });
});

// ── Tabs ─────────────────────────────────────────────────────────────────────
document.getElementById('tab-bar').addEventListener('click', (e) => {
  const tab = e.target.closest('.tab');
  if (!tab) return;
  const name = tab.dataset.tab;
  document.querySelectorAll('.tab').forEach(t => t.classList.toggle('active', t.dataset.tab === name));
  document.querySelectorAll('.tab-panel').forEach(p => p.classList.toggle('active', p.id === 'tab-' + name));
});

// ── Shared helpers ────────────────────────────────────────────────────────────
function fmtSize(bytes) {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024*1024) return (bytes/1024).toFixed(1) + ' kB';
  return (bytes/1024/1024).toFixed(1) + ' MB';
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}

// ── Images ────────────────────────────────────────────────────────────────────
const imageList      = document.getElementById('image-list');
const imgUploadForm  = document.getElementById('img-upload-form');
const imgUploadInput = document.getElementById('img-upload-input');
const imgUploadStatus = document.getElementById('img-upload-status');

function renderImages(images) {
  if (!images || !images.length) {
    imageList.innerHTML = "<div style='color:#9ba7b6;font-size:12px'>(no images uploaded)</div>";
    return;
  }
  imageList.innerHTML = images.map(v =>
    "<div class='media-row' data-id='" + v.id + "'>" +
      "<span class='name'>" + escapeHtml(v.filename) + "</span>" +
      "<span class='size'>" + fmtSize(v.size_bytes) + "</span>" +
      "<div class='row-buttons'>" +
        "<button data-play='mirror'>play mirrored</button>" +
        "<button data-play='span' disabled title='disabled: content is not authored for the full-wall aspect'>play stretched</button>" +
        "<button data-queue='1'>+ queue</button>" +
        "<button class='danger' data-delete='1'>delete</button>" +
      "</div>" +
    "</div>"
  ).join('');
}

async function refreshImages() {
  try {
    const r = await fetch(BASE + '/admin/images');
    renderImages((await r.json()).images || []);
  } catch(e) {
    imageList.innerHTML = "<div class='err'>image list failed: " + escapeHtml(e.message) + "</div>";
  }
}

imageList.addEventListener('click', async (e) => {
  const row = e.target.closest('.media-row');
  if (!row) return;
  const id = row.dataset.id;
  if (e.target.dataset.play) {
    await fetch(BASE + '/admin/play-image', {
      method: 'POST', headers: {'Content-Type':'application/json'},
      body: JSON.stringify({image_id: id, mode: e.target.dataset.play})
    });
  } else if (e.target.dataset.delete) {
    if (!confirm('Delete this image?')) return;
    const r = await fetch(BASE + '/admin/images/' + encodeURIComponent(id), {method:'DELETE'});
    if (r.ok) await refreshImages();
  }
});

imgUploadForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  const file = imgUploadInput.files[0];
  if (!file) return;
  const fd = new FormData();
  fd.append('image', file);
  imgUploadStatus.textContent = 'uploading ' + file.name + '…';
  try {
    const r = await fetch(BASE + '/admin/images', {method:'POST', body:fd});
    if (!r.ok) { imgUploadStatus.textContent = 'failed: ' + await r.text(); return; }
    imgUploadStatus.textContent = 'uploaded ' + file.name;
    imgUploadInput.value = '';
    await refreshImages();
  } catch(err) { imgUploadStatus.textContent = 'error: ' + err.message; }
});

refreshImages();
setInterval(refreshImages, 5000);

// ── Videos ────────────────────────────────────────────────────────────────────
const videoList     = document.getElementById('video-list');
const uploadForm    = document.getElementById('upload-form');
const uploadInput   = document.getElementById('upload-input');
const uploadStatus  = document.getElementById('upload-status');

function renderVideos(videos) {
  if (!videos || !videos.length) {
    videoList.innerHTML = "<div style='color:#9ba7b6;font-size:12px'>(no videos uploaded)</div>";
    return;
  }
  videoList.innerHTML = videos.map(v =>
    "<div class='media-row' data-id='" + v.id + "'>" +
      "<span class='name'>" + escapeHtml(v.filename) + "</span>" +
      "<span class='size'>" + fmtSize(v.size_bytes) + "</span>" +
      "<div class='row-buttons'>" +
        "<button data-play='mirror'>play mirrored</button>" +
        "<button data-play='stretch' disabled title='disabled: content is not authored for the full-wall aspect'>play stretched</button>" +
        "<button data-queue='1'>+ queue</button>" +
        "<button class='danger' data-delete='1'>delete</button>" +
      "</div>" +
    "</div>"
  ).join('');
}

async function refreshVideos() {
  try {
    const r = await fetch(BASE + '/admin/videos');
    renderVideos((await r.json()).videos || []);
  } catch(e) {
    videoList.innerHTML = "<div class='err'>video list failed: " + escapeHtml(e.message) + "</div>";
  }
}

videoList.addEventListener('click', async (e) => {
  const row = e.target.closest('.media-row');
  if (!row) return;
  const id = row.dataset.id;
  if (e.target.dataset.play) {
    await fetch(BASE + '/admin/play', {
      method: 'POST', headers: {'Content-Type':'application/json'},
      body: JSON.stringify({video_id: id, mode: e.target.dataset.play})
    });
  } else if (e.target.dataset.delete) {
    if (!confirm('Delete this video?')) return;
    const r = await fetch(BASE + '/admin/videos/' + encodeURIComponent(id), {method:'DELETE'});
    if (r.ok) await refreshVideos();
  }
});

uploadForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  const file = uploadInput.files[0];
  if (!file) return;
  const fd = new FormData();
  fd.append('video', file);
  uploadStatus.textContent = 'uploading ' + file.name + '…';
  try {
    const r = await fetch(BASE + '/admin/videos', {method:'POST', body:fd});
    if (!r.ok) { uploadStatus.textContent = 'failed: ' + await r.text(); return; }
    uploadStatus.textContent = 'uploaded ' + file.name;
    uploadInput.value = '';
    await refreshVideos();
  } catch(err) { uploadStatus.textContent = 'error: ' + err.message; }
});

refreshVideos();
setInterval(refreshVideos, 5000);

// ── Add-to-queue (from any media row) ────────────────────────────────────────
async function queueMedia(kind, id) {
  let dur;
  if (kind === 'video') {
    // Videos default to play-to-end; duration_s=0 tells the server to use the
    // natural duration reported by the client on loadedmetadata.
    dur = 0;
  } else {
    const durInput = prompt('Duration in seconds:', '15');
    if (durInput == null) return;
    dur = Number.parseInt(durInput, 10);
    if (!Number.isFinite(dur) || dur < 1) { alert('duration must be >= 1'); return; }
  }
  const item = { kind, ref_id: id, mode: 'mirror', duration_s: dur };
  const r = await fetch(BASE + '/admin/queue/items', {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({ item })
  });
  if (!r.ok) alert('add failed: ' + await r.text());
}

imageList.addEventListener('click', (e) => {
  if (e.target.dataset.queue) {
    const row = e.target.closest('.media-row');
    if (row) queueMedia('image', row.dataset.id);
  }
});
videoList.addEventListener('click', (e) => {
  if (e.target.dataset.queue) {
    const row = e.target.closest('.media-row');
    if (row) queueMedia('video', row.dataset.id);
  }
});

// ── Scenes: add-to-queue chip on each scene button ───────────────────────────
sceneGrid.addEventListener('contextmenu', async (e) => {
  const btn = e.target.closest('.scene-btn');
  if (!btn) return;
  e.preventDefault();
  const s = btn.dataset.scene;
  if (s === 'idle' || s === 'video' || s === 'image') return;
  const durInput = prompt('Duration in seconds for scene "' + s + '":', '20');
  if (durInput == null) return;
  const dur = Number.parseInt(durInput, 10);
  if (!Number.isFinite(dur) || dur < 1) { alert('duration must be >= 1'); return; }
  const item = { kind: 'scene', scene: s, duration_s: dur };
  const r = await fetch(BASE + '/admin/queue/items', {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({ item })
  });
  if (!r.ok) alert('add failed: ' + await r.text());
});

// ── Playlists tab ────────────────────────────────────────────────────────────
const playlistList = document.getElementById('playlist-list');
const plEditor = document.getElementById('pl-editor');
const plNewBtn = document.getElementById('pl-new-btn');
const plSaveQueueBtn = document.getElementById('pl-save-queue-btn');

const QUEUEABLE_SCENES = ['three-unified', 'solar-panorama', 'warp-jump'];
let cachedImages = [];
let cachedVideos = [];
let editorState = null; // { id?: string, name, loop, items }

async function refreshPlaylists() {
  try {
    const r = await fetch(BASE + '/admin/playlists');
    const j = await r.json();
    renderPlaylists(j.playlists || []);
  } catch(e) {
    playlistList.innerHTML = "<div class='err'>playlist list failed: " + escapeHtml(e.message) + "</div>";
  }
}

function renderPlaylists(items) {
  if (!items || !items.length) {
    playlistList.innerHTML = "<div style='color:#9ba7b6;font-size:12px'>(no playlists saved)</div>";
    return;
  }
  playlistList.innerHTML = items.map(p => {
    const plItems = p.items || [];
    return "<div class='playlist-row' data-id='" + p.id + "'>" +
      "<span class='pl-name'>" + escapeHtml(p.name) + "</span>" +
      "<span class='pl-meta'>" + plItems.length + " item" + (plItems.length===1?'':'s') + (p.loop ? ' · loop' : '') + "</span>" +
      "<div class='row-buttons'>" +
        "<button data-action='load'>load & play</button>" +
        "<button data-action='edit'>edit</button>" +
        "<button class='danger' data-action='delete'>delete</button>" +
      "</div>" +
    "</div>";
  }).join('');
}

playlistList.addEventListener('click', async (e) => {
  const row = e.target.closest('.playlist-row');
  if (!row) return;
  const id = row.dataset.id;
  const action = e.target.dataset.action;
  if (action === 'load') {
    await fetch(BASE + '/admin/queue/load', {
      method:'POST', headers:{'Content-Type':'application/json'},
      body: JSON.stringify({ playlist_id: id })
    });
    await fetch(BASE + '/admin/queue/play', {method:'POST'});
  } else if (action === 'delete') {
    if (!confirm('Delete this playlist?')) return;
    const r = await fetch(BASE + '/admin/playlists/' + encodeURIComponent(id), {method:'DELETE'});
    if (r.ok) await refreshPlaylists();
  } else if (action === 'edit') {
    const r = await fetch(BASE + '/admin/playlists');
    const j = await r.json();
    const p = (j.playlists || []).find(x => x.id === id);
    if (p) {
      if (!Array.isArray(p.items)) p.items = [];
      openEditor(p);
    }
  }
});

plNewBtn.addEventListener('click', () => openEditor(null));
plSaveQueueBtn.addEventListener('click', async () => {
  const name = prompt('Name for the new playlist:');
  if (!name) return;
  const loop = confirm('Should this playlist loop? OK for yes, Cancel for no.');
  const r = await fetch(BASE + '/admin/queue/save-as', {
    method:'POST', headers:{'Content-Type':'application/json'},
    body: JSON.stringify({ name, loop })
  });
  if (!r.ok) { alert('save failed: ' + await r.text()); return; }
  await refreshPlaylists();
});

function openEditor(existing) {
  editorState = existing
    ? { id: existing.id, name: existing.name, loop: existing.loop, items: existing.items.map(x => ({...x})) }
    : { name: '', loop: true, items: [] };
  plEditor.hidden = false;
  renderEditor();
}

function closeEditor() {
  editorState = null;
  plEditor.hidden = true;
  plEditor.innerHTML = '';
}

function renderEditor() {
  if (!editorState) return;
  const isEdit = !!editorState.id;
  const kindOpts = ['scene','image','video'].map(k => "<option value='" + k + "'>" + k + "</option>").join('');
  plEditor.innerHTML =
    "<div class='pe-row'>" +
      "<label>name</label>" +
      "<input type='text' id='pe-name' value='" + escapeHtml(editorState.name || '') + "' style='flex:1;min-width:180px'>" +
      "<label><input type='checkbox' id='pe-loop'" + (editorState.loop ? ' checked' : '') + "> loop</label>" +
    "</div>" +
    "<div class='pe-items' id='pe-items'></div>" +
    "<div class='pe-row'>" +
      "<label>add</label>" +
      "<select id='pe-add-kind'>" + kindOpts + "</select>" +
      "<select id='pe-add-ref'></select>" +
      "<input type='number' id='pe-add-dur' min='1' value='20' style='width:80px'>" +
      "<button id='pe-add-btn'>+ add item</button>" +
    "</div>" +
    "<div class='pe-actions'>" +
      "<button id='pe-save' class='primary'>" + (isEdit ? 'save changes' : 'create playlist') + "</button>" +
      "<button id='pe-cancel'>cancel</button>" +
    "</div>";

  const nameEl = document.getElementById('pe-name');
  const loopEl = document.getElementById('pe-loop');
  nameEl.addEventListener('input', () => { editorState.name = nameEl.value; });
  loopEl.addEventListener('change', () => { editorState.loop = loopEl.checked; });

  renderEditorItems();
  const addKind = document.getElementById('pe-add-kind');
  const addRef  = document.getElementById('pe-add-ref');
  const addDur  = document.getElementById('pe-add-dur');
  function reflectKindDefaults() {
    if (addKind.value === 'video') {
      addDur.min = '0';
      addDur.value = '0';
      addDur.title = '0 = play to natural end';
    } else {
      addDur.min = '1';
      if (Number.parseInt(addDur.value, 10) < 1) addDur.value = '20';
      addDur.title = '';
    }
    populateRefOptions(addKind.value, addRef);
  }
  addKind.addEventListener('change', reflectKindDefaults);
  reflectKindDefaults();

  document.getElementById('pe-add-btn').addEventListener('click', () => {
    const kind = addKind.value;
    const rawDur = document.getElementById('pe-add-dur').value;
    let durVal = Number.parseInt(rawDur, 10);
    if (!Number.isFinite(durVal) || durVal < 0) durVal = 0;
    if (kind === 'scene') {
      const scene = addRef.value;
      if (!scene) return;
      if (durVal < 1) { alert('scene items need a duration >= 1'); return; }
      editorState.items.push({ kind, scene, duration_s: durVal });
    } else if (kind === 'image') {
      const ref = addRef.value;
      if (!ref) { alert('no images uploaded'); return; }
      if (durVal < 1) { alert('image items need a duration >= 1'); return; }
      editorState.items.push({ kind, ref_id: ref, mode: 'mirror', duration_s: durVal });
    } else if (kind === 'video') {
      const ref = addRef.value;
      if (!ref) { alert('no videos uploaded'); return; }
      // duration_s = 0 means play to natural end.
      editorState.items.push({ kind, ref_id: ref, mode: 'mirror', duration_s: durVal });
    }
    renderEditorItems();
  });

  document.getElementById('pe-cancel').addEventListener('click', closeEditor);
  document.getElementById('pe-save').addEventListener('click', savePlaylist);
}

function populateRefOptions(kind, sel) {
  let opts = [];
  if (kind === 'scene') {
    opts = QUEUEABLE_SCENES.map(s => "<option value='" + s + "'>" + s + "</option>");
  } else if (kind === 'image') {
    opts = cachedImages.map(m => "<option value='" + m.id + "'>" + escapeHtml(m.filename) + "</option>");
    if (!opts.length) opts = ["<option value=''>(none uploaded)</option>"];
  } else if (kind === 'video') {
    opts = cachedVideos.map(m => "<option value='" + m.id + "'>" + escapeHtml(m.filename) + "</option>");
    if (!opts.length) opts = ["<option value=''>(none uploaded)</option>"];
  }
  sel.innerHTML = opts.join('');
}

function renderEditorItems() {
  const box = document.getElementById('pe-items');
  if (!box) return;
  if (!editorState.items.length) {
    box.innerHTML = "<div style='color:#9ba7b6;font-size:12px'>(no items yet)</div>";
    return;
  }
  box.innerHTML = editorState.items.map((it, i) => {
    let label, natural;
    if (it.kind === 'scene') label = it.scene;
    else {
      const src = it.kind === 'image' ? cachedImages : cachedVideos;
      const m = src.find(x => x.id === it.ref_id);
      label = m ? m.filename : (it.kind + ':' + it.ref_id);
      if (it.kind === 'video' && m && m.duration_ms) natural = (m.duration_ms / 1000).toFixed(1);
    }
    const isVideo = it.kind === 'video';
    const durMin = isVideo ? 0 : 1;
    const naturalHint = isVideo
      ? (it.duration_s === 0
          ? (natural ? '(play to end · ' + natural + 's)' : '(play to end)')
          : '')
      : '';
    return "<div class='pe-item' data-idx='" + i + "'>" +
      "<span class='pe-idx'>" + (i+1) + ".</span>" +
      "<span class='qi-kind'>" + escapeHtml(it.kind) + "</span>" +
      "<span class='pe-label'>" + escapeHtml(label) + "</span>" +
      "<input type='number' min='" + durMin + "' value='" + it.duration_s + "' data-role='dur'> s" +
      (naturalHint ? "<span class='qi-dur'>" + naturalHint + "</span>" : '') +
      "<button data-role='up' title='move up'>↑</button>" +
      "<button data-role='down' title='move down'>↓</button>" +
      "<button class='danger' data-role='remove'>×</button>" +
    "</div>";
  }).join('');

  box.querySelectorAll('.pe-item').forEach(el => {
    const idx = Number.parseInt(el.dataset.idx, 10);
    el.querySelector("[data-role='dur']").addEventListener('input', (e) => {
      const v = Number.parseInt(e.target.value, 10);
      const minVal = editorState.items[idx].kind === 'video' ? 0 : 1;
      if (Number.isFinite(v) && v >= minVal) editorState.items[idx].duration_s = v;
    });
    el.querySelector("[data-role='up']").addEventListener('click', () => moveItem(idx, idx-1));
    el.querySelector("[data-role='down']").addEventListener('click', () => moveItem(idx, idx+1));
    el.querySelector("[data-role='remove']").addEventListener('click', () => {
      editorState.items.splice(idx, 1);
      renderEditorItems();
    });
  });
}

function moveItem(from, to) {
  if (to < 0 || to >= editorState.items.length) return;
  const [item] = editorState.items.splice(from, 1);
  editorState.items.splice(to, 0, item);
  renderEditorItems();
}

async function savePlaylist() {
  const body = { name: editorState.name, loop: editorState.loop, items: editorState.items };
  const url  = editorState.id ? (BASE + '/admin/playlists/' + editorState.id) : (BASE + '/admin/playlists');
  const method = editorState.id ? 'PUT' : 'POST';
  const r = await fetch(url, { method, headers:{'Content-Type':'application/json'}, body: JSON.stringify(body) });
  if (!r.ok) { alert('save failed: ' + await r.text()); return; }
  closeEditor();
  await refreshPlaylists();
}

async function refreshMediaCaches() {
  try {
    const [imgR, vidR] = await Promise.all([
      fetch(BASE + '/admin/images'),
      fetch(BASE + '/admin/videos'),
    ]);
    cachedImages = (await imgR.json()).images || [];
    cachedVideos = (await vidR.json()).videos || [];
  } catch(e) {}
}

refreshMediaCaches();
setInterval(refreshMediaCaches, 5000);
refreshPlaylists();
setInterval(refreshPlaylists, 5000);
</script>
</body></html>`))

func (a *Admin) ServeAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = adminTmpl.Execute(w, map[string]string{"BasePath": a.BasePath})
}
