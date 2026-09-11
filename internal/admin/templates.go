package admin

import (
	"html/template"
	"net/http"
)

var adminTmpl = template.Must(template.New("admin").Parse(`<!doctype html>
<html><head><meta charset="utf-8">
<meta name="base-path" content="{{.BasePath}}">
<title>wallsync · admin</title>
<style>
  body { background:#0b0e12; color:#dfe6ef; font-family:system-ui,sans-serif; margin:0; padding:20px; }
  h1 { font-size:18px; margin:0 0 16px; letter-spacing:0.03em; }
  #status-bar { display:flex; align-items:center; gap:12px; margin-bottom:16px; flex-wrap:wrap; }
  #status-bar .now { font-family:ui-monospace,monospace; font-size:13px; }
  #countdown { color:#9ba7b6; font-family:ui-monospace,monospace; font-size:12px; }
  #cards { display:grid; grid-template-columns:repeat(5,1fr); gap:12px; margin:0 0 24px; }
  .card { background:#141a22; border:1px solid #212a35; border-radius:8px; padding:12px; }
  .card h2 { margin:0 0 8px; font-size:14px; letter-spacing:0.05em; }
  .row { display:flex; justify-content:space-between; font-size:12px; padding:2px 0; font-family:ui-monospace,monospace; }
  .dot { display:inline-block; width:10px; height:10px; border-radius:50%; margin-right:6px; vertical-align:middle; }
  .green { background:#4ade80; box-shadow:0 0 8px #4ade80aa; }
  .amber { background:#fbbf24; box-shadow:0 0 8px #fbbf24aa; }
  .red   { background:#f87171; box-shadow:0 0 8px #f87171aa; }
  .warn  { color:#fbbf24; }
  .err   { color:#f87171; }
  button { background:#1f2a38; color:#dfe6ef; border:1px solid #2c3a4a; padding:6px 14px; border-radius:6px; cursor:pointer; font:inherit; font-size:13px; }
  button:hover { background:#26344a; }
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
  .media-row .name { flex:1; font-family:ui-monospace,monospace; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
  .media-row .size { color:#9ba7b6; font-family:ui-monospace,monospace; font-size:11px; flex-shrink:0; }
  .danger { background:#3a1e1e; border-color:#5a2c2c; }
  .danger:hover { background:#4a2828; }
  .upload-row { display:flex; align-items:center; gap:8px; margin-top:12px; }
  .upload-status { color:#9ba7b6; font-family:ui-monospace,monospace; font-size:12px; }
  input[type=file] { color:#dfe6ef; font:inherit; }
</style>
</head><body>
  <h1>wallsync</h1>
  <div id="status-bar">
    <span class="now">Scene: <b id="scene-name">–</b></span>
    <span id="countdown"></span>
    <button id="autorotate-btn">auto-rotate off</button>
  </div>
  <div id="cards"></div>
  <div id="content">
    <div id="tab-bar">
      <button class="tab active" data-tab="scenes">scenes</button>
      <button class="tab" data-tab="images">images</button>
      <button class="tab" data-tab="videos">videos</button>
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
  cards.innerHTML = status.nodes.map(n => {
    const cls = classifyNode(n);
    return "<div class='card'>" +
      "<h2><span class='dot " + cls + "'></span>node " + n.id + "</h2>" +
      "<div class='row'><span>state</span><b>" + (n.connected ? 'connected' : 'offline') + "</b></div>" +
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
      "<button data-play='span'>play span</button>" +
      "<button data-play='mirror'>play mirror</button>" +
      "<button class='danger' data-delete='1'>delete</button>" +
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
      "<button data-play='mirror'>play mirrored</button>" +
      "<button data-play='stretch'>play stretched</button>" +
      "<button class='danger' data-delete='1'>delete</button>" +
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
</script>
</body></html>`))

func (a *Admin) ServeAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = adminTmpl.Execute(w, map[string]string{"BasePath": a.BasePath})
}
