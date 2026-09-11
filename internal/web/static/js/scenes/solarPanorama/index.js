import * as THREE from 'three';

// Solar system ecliptic panorama, laid out horizontally across the whole
// 28800 × 3240 wall. Distances are TO SCALE (1 AU = 900 wall units).
// Planet sizes are power-compressed so terrestrial planets stay visible
// next to the gas giants.

const AU = 900;                // wall units per AU
const SUN_X = 500;
const CENTER_Y = 1620;
const SPOTLIGHT_INTERVAL = 8;  // seconds per planet in the auto-focus cycle

const PLANETS = [
  { name: 'Sun',     auDist: 0,     radiusKm: 695700, color: 0xffbb33, isStar: true },
  { name: 'Mercury', auDist: 0.39,  radiusKm: 2439.7, color: 0x8c8c8c, dayHours: 1407.6, yearDays: 88,      spinRate: 0.03 },
  { name: 'Venus',   auDist: 0.72,  radiusKm: 6051.8, color: 0xd8b171, dayHours: 5832,   yearDays: 224.7,   spinRate: 0.02 },
  { name: 'Earth',   auDist: 1.00,  radiusKm: 6371.0, color: 0x2a6fbc, dayHours: 23.9,   yearDays: 365.25,  spinRate: 0.30, hasClouds: true },
  { name: 'Mars',    auDist: 1.524, radiusKm: 3389.5, color: 0xa63e0c, dayHours: 24.6,   yearDays: 687,     spinRate: 0.28 },
  { name: 'Jupiter', auDist: 5.20,  radiusKm: 69911,  color: 0xc9a978, dayHours: 9.9,    yearDays: 4333,    spinRate: 0.55, bands: true },
  { name: 'Saturn',  auDist: 9.54,  radiusKm: 58232,  color: 0xd4b877, dayHours: 10.7,   yearDays: 10759,   spinRate: 0.50, hasRings: true },
  { name: 'Uranus',  auDist: 19.19, radiusKm: 25362,  color: 0x89d3d6, dayHours: 17.2,   yearDays: 30687,   spinRate: 0.20 },
  { name: 'Neptune', auDist: 30.07, radiusKm: 24622,  color: 0x3d68d9, dayHours: 16.1,   yearDays: 60190,   spinRate: 0.22 },
];

function planetDisplayRadius(km) {
  return Math.pow(km, 0.42) * 0.62;
}
for (const p of PLANETS) {
  p.x = SUN_X + p.auDist * AU;
  p.y = CENTER_Y;
  p.displayRadius = p.isStar ? 260 : planetDisplayRadius(p.radiusKm);
}

// ---- Planet shader: colour + soft terminator + fresnel rim ----
const PLANET_VERT = `
varying vec3 vNormal;
varying vec3 vView;
varying vec3 vPos;
void main() {
  vec4 mv = modelViewMatrix * vec4(position, 1.0);
  vNormal = normalize(normalMatrix * normal);
  vView = normalize(-mv.xyz);
  vPos = position;
  gl_Position = projectionMatrix * mv;
}
`;
const PLANET_FRAG = `
precision highp float;
uniform vec3 uColor;
uniform vec3 uLightDir;   // world-space light direction (toward sun)
uniform float uBands;     // 1.0 → Jupiter banding
uniform float uTime;
uniform mat3 uNormalMatrix;

float hash(vec3 p) { return fract(sin(dot(p, vec3(127.1, 311.7, 74.7))) * 43758.5453); }
float vnoise(vec3 p) {
  vec3 i = floor(p); vec3 f = fract(p); vec3 u = f*f*(3.0-2.0*f);
  return mix(mix(mix(hash(i),hash(i+vec3(1,0,0)),u.x), mix(hash(i+vec3(0,1,0)),hash(i+vec3(1,1,0)),u.x), u.y),
             mix(mix(hash(i+vec3(0,0,1)),hash(i+vec3(1,0,1)),u.x), mix(hash(i+vec3(0,1,1)),hash(i+vec3(1,1,1)),u.x), u.y), u.z);
}

varying vec3 vNormal;
varying vec3 vView;
varying vec3 vPos;

void main() {
  vec3 nrm = normalize(vPos);
  vec3 col = uColor;
  if (uBands > 0.5) {
    float lat = nrm.y;
    float band = 0.5 + 0.5 * sin(lat * 22.0 + vnoise(vec3(lat * 6.0, 0.0, 0.0)) * 1.5);
    col = mix(uColor * 0.65, uColor * 1.2, band);
    col += vec3(0.08, 0.03, 0.0) * (0.5 + 0.5 * sin(lat * 40.0));
  } else {
    // Subtle surface variation.
    float n = vnoise(nrm * 4.0);
    col = mix(uColor * 0.85, uColor * 1.15, n);
  }
  // Terminator: light direction transformed into view space by uNormalMatrix.
  vec3 lightN = normalize(uNormalMatrix * uLightDir);
  float shade = 0.28 + 0.85 * max(0.0, dot(vNormal, lightN));
  float rim = pow(1.0 - max(0.0, dot(vNormal, vView)), 2.0);
  col = col * shade + vec3(0.4, 0.6, 0.9) * rim * 0.22;
  gl_FragColor = vec4(col, 1.0);
}
`;

// ---- Sun shader: emissive + noisy surface + slow flicker ----
const SUN_FRAG = `
precision highp float;
uniform float uTime;
varying vec3 vPos;
varying vec3 vNormal;
varying vec3 vView;

float hash(vec3 p) { return fract(sin(dot(p, vec3(127.1, 311.7, 74.7))) * 43758.5453); }
float vnoise(vec3 p) {
  vec3 i = floor(p); vec3 f = fract(p); vec3 u = f*f*(3.0-2.0*f);
  return mix(mix(mix(hash(i),hash(i+vec3(1,0,0)),u.x), mix(hash(i+vec3(0,1,0)),hash(i+vec3(1,1,0)),u.x), u.y),
             mix(mix(hash(i+vec3(0,0,1)),hash(i+vec3(1,0,1)),u.x), mix(hash(i+vec3(0,1,1)),hash(i+vec3(1,1,1)),u.x), u.y), u.z);
}
float fbm(vec3 p) { float v=0.0, a=0.5, f=1.0; for(int i=0;i<4;i++){ v+=a*vnoise(p*f); f*=2.0; a*=0.5; } return v; }

void main() {
  vec3 nrm = normalize(vPos);
  float n = fbm(nrm * 4.0 + vec3(0.0, uTime * 0.15, 0.0));
  vec3 hot = mix(vec3(1.0, 0.75, 0.15), vec3(1.0, 1.0, 0.85), pow(n, 1.5));
  float rim = pow(1.0 - max(0.0, dot(vNormal, vView)), 1.7);
  hot += vec3(1.0, 0.55, 0.1) * rim * 0.9;
  gl_FragColor = vec4(hot, 1.0);
}
`;

// ---- Corona sprite shader ----
const CORONA_FRAG = `
precision highp float;
varying vec2 vUv;
void main() {
  vec2 uv = vUv - 0.5;
  float d = length(uv) * 2.0;
  if (d > 1.0) discard;
  float glow = pow(1.0 - d, 2.2);
  vec3 col = mix(vec3(1.0, 0.5, 0.1), vec3(1.0, 0.85, 0.4), pow(glow, 0.5));
  gl_FragColor = vec4(col, glow * 0.85);
}
`;
const CORONA_VERT = `
varying vec2 vUv;
void main() {
  vUv = uv;
  gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
`;

// ---- Star shader with twinkling ----
const STAR_VERT = `
attribute float aSeed;
uniform float uAspect;
uniform float uTime;
varying float vTwinkle;
void main() {
  gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
  float rate = 0.4 + fract(aSeed * 3.7) * 3.5;
  float phase = aSeed * 40.0;
  vTwinkle = pow(0.55 + 0.45 * sin(uTime * rate + phase), 4.0);
  float baseSize = mix(2.0, 5.0, fract(aSeed * 7.7));
  gl_PointSize = baseSize * uAspect * mix(0.6, 1.4, vTwinkle);
}
`;
const STAR_FRAG = `
precision highp float;
varying float vTwinkle;
void main() {
  vec2 uv = gl_PointCoord - 0.5;
  float d = length(uv);
  if (d > 0.5) discard;
  float soft = smoothstep(0.5, 0.0, d);
  float a = soft * mix(0.4, 1.0, vTwinkle);
  gl_FragColor = vec4(vec3(1.0, 0.97, 0.92) * a, a);
}
`;

// ---- Small-body point shader (asteroid & Kuiper belts) ----
const DUST_VERT = `
attribute float aSeed;
uniform float uAspect;
void main() {
  gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
  gl_PointSize = mix(2.0, 5.5, fract(aSeed * 7.7)) * uAspect;
}
`;
const DUST_FRAG = `
precision highp float;
void main() {
  vec2 uv = gl_PointCoord - 0.5;
  float d = length(uv);
  if (d > 0.5) discard;
  float a = smoothstep(0.5, 0.0, d) * 0.7;
  gl_FragColor = vec4(vec3(0.75, 0.7, 0.6) * a, a);
}
`;

// ---- Spotlight ring shader ----
const RING_VERT = `
varying vec2 vUv;
void main() {
  vUv = uv;
  gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
`;
const RING_FRAG = `
precision highp float;
uniform float uPulse;
varying vec2 vUv;
void main() {
  vec2 uv = vUv - 0.5;
  float d = length(uv) * 2.0;
  if (d > 1.0 || d < 0.86) discard;
  float t = 1.0 - abs(d - 0.93) / 0.07;
  vec3 col = vec3(0.15, 1.0, 0.85);
  float a = t * uPulse;
  gl_FragColor = vec4(col, a);
}
`;

// Deterministic PRNG helper.
function mulberry(seed) {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6D2B79F5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

// Label / callout canvas helpers.
function makeLabel(text, sub) {
  const canvas = document.createElement('canvas');
  canvas.width = 640; canvas.height = 140;
  const ctx = canvas.getContext('2d');
  ctx.textAlign = 'center'; ctx.textBaseline = 'top';
  ctx.font = '700 44px ui-monospace, "SF Mono", monospace';
  ctx.fillStyle = '#0dfab8';
  ctx.fillText(text, canvas.width / 2, 4);
  if (sub) {
    ctx.font = '400 26px ui-monospace, "SF Mono", monospace';
    ctx.fillStyle = '#7fd3c1';
    ctx.fillText(sub, canvas.width / 2, 62);
  }
  return canvas;
}
function makeCallout(planet) {
  const canvas = document.createElement('canvas');
  canvas.width = 1100; canvas.height = 380;
  const ctx = canvas.getContext('2d');
  const pad = 28;
  ctx.strokeStyle = '#0dfab8';
  ctx.lineWidth = 4;
  ctx.strokeRect(pad, pad, canvas.width - pad * 2, canvas.height - pad * 2);
  ctx.font = '700 88px ui-monospace, "SF Mono", monospace';
  ctx.fillStyle = '#ffffff';
  ctx.textAlign = 'left'; ctx.textBaseline = 'top';
  ctx.fillText(planet.name.toUpperCase(), pad + 40, pad + 30);
  ctx.font = '400 34px ui-monospace, "SF Mono", monospace';
  ctx.fillStyle = '#0dfab8';
  const rows = [];
  if (planet.isStar) {
    rows.push(`DIAMETER   ${(planet.radiusKm*2).toLocaleString()} km`);
    rows.push(`TYPE       G-TYPE MAIN SEQUENCE`);
    rows.push(`AGE        4.6 GYr`);
  } else {
    rows.push(`DIST       ${planet.auDist.toFixed(3)} AU  ·  ${(planet.auDist * 149.6e6).toExponential(2)} km`);
    rows.push(`DIAMETER   ${(planet.radiusKm*2).toLocaleString()} km`);
    rows.push(`ORBIT      ${planet.yearDays.toLocaleString()} d`);
    rows.push(`DAY        ${planet.dayHours.toFixed(1)} h`);
  }
  rows.forEach((r, i) => ctx.fillText(r, pad + 40, pad + 130 + i * 52));
  return canvas;
}

// ---- Belt point generation ----
function makeBelt(seed, xMin, xMax, ySpread, count, zSpread) {
  const geo = new THREE.BufferGeometry();
  const positions = new Float32Array(count * 3);
  const seeds = new Float32Array(count);
  const rnd = mulberry(seed);
  for (let i = 0; i < count; i++) {
    positions[i*3+0] = xMin + rnd() * (xMax - xMin);
    positions[i*3+1] = CENTER_Y + (rnd() - 0.5) * 2 * ySpread;
    positions[i*3+2] = (rnd() - 0.5) * 2 * zSpread;
    seeds[i] = rnd();
  }
  geo.setAttribute('position', new THREE.BufferAttribute(positions, 3));
  geo.setAttribute('aSeed', new THREE.BufferAttribute(seeds, 1));
  geo.boundingSphere = new THREE.Sphere(new THREE.Vector3((xMin+xMax)/2, CENTER_Y, 0), Math.max(xMax-xMin, ySpread*2, zSpread*2));
  return geo;
}

function makeStars(wallW, wallH, count, seed) {
  const geo = new THREE.BufferGeometry();
  const positions = new Float32Array(count * 3);
  const seeds = new Float32Array(count);
  const rnd = mulberry(seed);
  for (let i = 0; i < count; i++) {
    positions[i*3+0] = -1000 + rnd() * (wallW + 2000);
    positions[i*3+1] = -500 + rnd() * (wallH + 1000);
    positions[i*3+2] = -3000 + rnd() * 2200; // behind planets
    seeds[i] = rnd();
  }
  geo.setAttribute('position', new THREE.BufferAttribute(positions, 3));
  geo.setAttribute('aSeed', new THREE.BufferAttribute(seeds, 1));
  geo.boundingSphere = new THREE.Sphere(new THREE.Vector3(wallW/2, wallH/2, -2000), wallW);
  return geo;
}

export default function SolarPanoramaFactory() {
  const state = {
    renderer: null, scene: null, camera: null,
    ctx: null,
    planetMeshes: [], planetMats: [],
    labelSprites: [], calloutSprite: null, calloutTex: null, ringMesh: null,
    ringMat: null,
    starMat: null,
    dustMats: [],
    sunMat: null,
    coronaMat: null,
    currentSpotlight: -1,
  };

  function buildPlanet(p, wallH) {
    // Sun uses its own shader; planets share PLANET_FRAG.
    const geo = new THREE.SphereGeometry(p.displayRadius, 48, 32);
    let mat;
    if (p.isStar) {
      mat = new THREE.ShaderMaterial({
        uniforms: { uTime: { value: 0 } },
        vertexShader: PLANET_VERT,
        fragmentShader: SUN_FRAG,
      });
    } else {
      const lightDir = new THREE.Vector3(SUN_X - p.x, 0, 0).normalize();
      // Flip so light comes FROM sun TO planet's near side.
      lightDir.multiplyScalar(-1);
      mat = new THREE.ShaderMaterial({
        uniforms: {
          uColor: { value: new THREE.Color(p.color) },
          uLightDir: { value: lightDir },
          uBands: { value: p.bands ? 1.0 : 0.0 },
          uTime: { value: 0 },
          uNormalMatrix: { value: new THREE.Matrix3() },
        },
        vertexShader: PLANET_VERT,
        fragmentShader: PLANET_FRAG,
      });
    }
    const mesh = new THREE.Mesh(geo, mat);
    mesh.position.set(p.x, p.y, 0);
    state.scene.add(mesh);
    state.planetMeshes.push(mesh);
    state.planetMats.push(mat);

    if (p.hasRings) {
      const r0 = p.displayRadius * 1.35;
      const r1 = p.displayRadius * 2.2;
      const ringGeo = new THREE.RingGeometry(r0, r1, 96, 1);
      // UV mapping so radial gradient works.
      const uv = ringGeo.attributes.uv;
      for (let i = 0; i < uv.count; i++) {
        const x = ringGeo.attributes.position.getX(i);
        const y = ringGeo.attributes.position.getY(i);
        const r = Math.sqrt(x*x + y*y);
        uv.setXY(i, (r - r0) / (r1 - r0), 0);
      }
      uv.needsUpdate = true;
      const ringMat = new THREE.ShaderMaterial({
        uniforms: {},
        vertexShader: `varying float vR; void main() { vR = uv.x; gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0); }`,
        fragmentShader: `
          precision highp float; varying float vR;
          void main() {
            float band1 = 1.0 - smoothstep(0.10, 0.14, vR) + smoothstep(0.30, 0.35, vR) * 0.4;
            float band2 = 1.0 - smoothstep(0.55, 0.60, vR) + smoothstep(0.75, 0.80, vR) * 0.5;
            float a = clamp((band1 * 0.7 + band2 * 0.6 + 0.35) * (1.0 - smoothstep(0.9, 1.0, vR)) - 0.05, 0.0, 1.0);
            vec3 col = mix(vec3(0.55, 0.42, 0.28), vec3(0.95, 0.86, 0.68), vR);
            gl_FragColor = vec4(col, a * 0.9);
          }
        `,
        transparent: true, side: THREE.DoubleSide, depthWrite: false,
      });
      const ringMesh = new THREE.Mesh(ringGeo, ringMat);
      ringMesh.rotation.x = -Math.PI / 2 + 0.28; // tilt
      ringMesh.rotation.z = 0.12;
      mesh.add(ringMesh);
    }

    return mesh;
  }

  function buildLabels() {
    for (let i = 0; i < PLANETS.length; i++) {
      const p = PLANETS[i];
      const sub = p.isStar ? 'STAR' : `${p.auDist.toFixed(2)} AU`;
      const canvas = makeLabel(p.name.toUpperCase(), sub);
      const tex = new THREE.CanvasTexture(canvas);
      tex.minFilter = THREE.LinearFilter;
      const mat = new THREE.SpriteMaterial({ map: tex, transparent: true, depthTest: false, depthWrite: false });
      const sp = new THREE.Sprite(mat);
      sp.scale.set(canvas.width, canvas.height, 1);
      // Below the planet, with clearance for the display radius.
      sp.position.set(p.x, p.y - Math.max(150, p.displayRadius + 90), 0);
      state.scene.add(sp);
      state.labelSprites.push(sp);
    }
  }

  function buildCallout() {
    // Blank canvas → texture, redrawn on spotlight change.
    const canvas = document.createElement('canvas');
    canvas.width = 1100; canvas.height = 380;
    const tex = new THREE.CanvasTexture(canvas);
    tex.minFilter = THREE.LinearFilter;
    const mat = new THREE.SpriteMaterial({ map: tex, transparent: true, depthTest: false, depthWrite: false });
    const sp = new THREE.Sprite(mat);
    sp.scale.set(canvas.width, canvas.height, 1);
    sp.visible = false;
    state.scene.add(sp);
    state.calloutSprite = sp;
    state.calloutTex = tex;
    state.calloutCanvas = canvas;
  }

  function buildRing() {
    // A camera-facing ring for the spotlight halo. Plane with a discard shader.
    const geo = new THREE.PlaneGeometry(1, 1);
    const mat = new THREE.ShaderMaterial({
      uniforms: { uPulse: { value: 0 } },
      vertexShader: RING_VERT, fragmentShader: RING_FRAG,
      transparent: true, depthTest: false, depthWrite: false,
    });
    const mesh = new THREE.Mesh(geo, mat);
    mesh.visible = false;
    state.scene.add(mesh);
    state.ringMesh = mesh;
    state.ringMat = mat;
  }

  function updateSpotlight(t) {
    const idx = Math.floor(t / SPOTLIGHT_INTERVAL) % PLANETS.length;
    const localT = (t % SPOTLIGHT_INTERVAL) / SPOTLIGHT_INTERVAL;  // 0..1
    // Pulse: quick ease-in, gentle hold, ease-out at end.
    let pulse;
    if (localT < 0.15) pulse = localT / 0.15;
    else if (localT > 0.85) pulse = (1 - localT) / 0.15;
    else pulse = 1;

    if (idx !== state.currentSpotlight) {
      state.currentSpotlight = idx;
      // Redraw callout.
      const p = PLANETS[idx];
      const src = makeCallout(p);
      const ctx = state.calloutCanvas.getContext('2d');
      ctx.clearRect(0, 0, state.calloutCanvas.width, state.calloutCanvas.height);
      ctx.drawImage(src, 0, 0);
      state.calloutTex.needsUpdate = true;
    }

    // Grow current planet slightly.
    for (let i = 0; i < state.planetMeshes.length; i++) {
      const target = (i === idx) ? (1.0 + 0.35 * pulse) : 1.0;
      const cur = state.planetMeshes[i].scale.x;
      const next = cur + (target - cur) * 0.15;
      state.planetMeshes[i].scale.setScalar(next);
    }

    // Position + show ring around current spotlight.
    const p = PLANETS[idx];
    const scaled = p.displayRadius * (1.0 + 0.35 * pulse);
    const ringSize = scaled * 4.5;
    state.ringMesh.position.set(p.x, p.y, 5);
    state.ringMesh.scale.set(ringSize, ringSize, 1);
    state.ringMesh.visible = pulse > 0.02;
    state.ringMat.uniforms.uPulse.value = pulse;

    // Callout above planet.
    state.calloutSprite.position.set(
      Math.min(28800 - 600, Math.max(600, p.x)),
      p.y + Math.max(340, scaled + 240),
      6,
    );
    state.calloutSprite.visible = pulse > 0.02;
    state.calloutSprite.material.opacity = pulse;
  }

  return {
    name: 'solar-panorama',

    async init(ctx) {
      state.ctx = ctx;
      const { canvas, nodeRect, wallSize, perNodeSize } = ctx;

      const renderer = new THREE.WebGLRenderer({
        canvas, antialias: true, powerPreference: 'high-performance',
      });
      renderer.setPixelRatio(1);
      renderer.setSize(perNodeSize.w, perNodeSize.h, false);
      renderer.setClearColor(0x02050e, 1);
      state.renderer = renderer;

      const scene = new THREE.Scene();
      state.scene = scene;

      // Perspective camera pushed way back → near-orthographic feel over
      // the panorama, but planets still shade correctly in 3D.
      const wallW = wallSize.w, wallH = wallSize.h;
      const aspect = wallW / wallH;
      const vFov = 12;
      const dist = wallH / (2 * Math.tan((vFov * Math.PI) / 180 / 2));
      const cam = new THREE.PerspectiveCamera(vFov, aspect, 100, dist * 4);
      cam.position.set(wallW / 2, CENTER_Y, dist);
      cam.lookAt(wallW / 2, CENTER_Y, 0);
      cam.setViewOffset(wallW, wallH, nodeRect.x, nodeRect.y, nodeRect.w, nodeRect.h);
      state.camera = cam;

      // Starfield.
      const starMat = new THREE.ShaderMaterial({
        uniforms: { uAspect: { value: 1.0 }, uTime: { value: 0 } },
        vertexShader: STAR_VERT, fragmentShader: STAR_FRAG,
        transparent: true, depthWrite: false, blending: THREE.AdditiveBlending,
      });
      state.starMat = starMat;
      scene.add(new THREE.Points(makeStars(wallW, wallH, 6000, 91827), starMat));

      // Asteroid belt (2.2 – 3.2 AU).
      const beltA = new THREE.ShaderMaterial({
        uniforms: { uAspect: { value: 1.0 } },
        vertexShader: DUST_VERT, fragmentShader: DUST_FRAG,
        transparent: true, depthWrite: false,
      });
      state.dustMats.push(beltA);
      const beltAX0 = SUN_X + 2.15 * AU;
      const beltAX1 = SUN_X + 3.25 * AU;
      scene.add(new THREE.Points(makeBelt(1234, beltAX0, beltAX1, 220, 1600, 60), beltA));

      // Kuiper belt (30 – 55 AU, clipped by wall width).
      const beltK = new THREE.ShaderMaterial({
        uniforms: { uAspect: { value: 1.0 } },
        vertexShader: DUST_VERT, fragmentShader: DUST_FRAG,
        transparent: true, depthWrite: false,
      });
      state.dustMats.push(beltK);
      const beltKX0 = SUN_X + 30 * AU;
      const beltKX1 = Math.min(28800 - 100, SUN_X + 55 * AU);
      if (beltKX1 > beltKX0) {
        scene.add(new THREE.Points(makeBelt(5678, beltKX0, beltKX1, 320, 500, 90), beltK));
      }

      // Planets.
      for (const p of PLANETS) buildPlanet(p, wallH);

      // Sun corona (large sprite behind the sun sphere).
      const coronaGeo = new THREE.PlaneGeometry(1400, 1400);
      const coronaMat = new THREE.ShaderMaterial({
        uniforms: {},
        vertexShader: CORONA_VERT, fragmentShader: CORONA_FRAG,
        transparent: true, depthWrite: false, blending: THREE.AdditiveBlending,
      });
      state.coronaMat = coronaMat;
      const corona = new THREE.Mesh(coronaGeo, coronaMat);
      corona.position.set(SUN_X, CENTER_Y, -20);
      scene.add(corona);

      // Sun lens flare — three additional glowing sprites lined up horizontally.
      const flareGeo = new THREE.PlaneGeometry(220, 220);
      const flareOffsets = [200, 600, 1200];
      for (const off of flareOffsets) {
        const m = new THREE.ShaderMaterial({
          uniforms: {},
          vertexShader: CORONA_VERT, fragmentShader: CORONA_FRAG,
          transparent: true, depthWrite: false, blending: THREE.AdditiveBlending,
        });
        const flare = new THREE.Mesh(flareGeo, m);
        flare.position.set(SUN_X + off, CENTER_Y, -18);
        scene.add(flare);
      }

      buildLabels();
      buildCallout();
      buildRing();
    },

    tick(t) {
      if (!state.renderer) return;
      const aspectFactor = state.ctx.wallSize.h / state.ctx.perNodeSize.h;

      state.starMat.uniforms.uAspect.value = aspectFactor;
      state.starMat.uniforms.uTime.value = t;
      for (const m of state.dustMats) m.uniforms.uAspect.value = aspectFactor;

      // Spin planets on their axes + gentle orbital wobble.
      for (let i = 0; i < PLANETS.length; i++) {
        const p = PLANETS[i];
        const mesh = state.planetMeshes[i];
        const mat = state.planetMats[i];
        if (p.isStar) {
          mat.uniforms.uTime.value = t;
          continue;
        }
        mesh.rotation.y = t * (p.spinRate || 0.2);
        // Small orbital wobble in Z (in/out of page) and slight X drift.
        const period = Math.min(240, 12 + p.yearDays * 0.02); // capped for visible motion
        const phase = t * (2 * Math.PI / period);
        mesh.position.z = 8 * Math.sin(phase);
        mesh.position.x = p.x + 12 * Math.cos(phase * 0.7);
        // Update normalMatrix for shader lighting.
        mesh.updateMatrixWorld();
        mat.uniforms.uNormalMatrix.value.getNormalMatrix(mesh.matrixWorld);
      }

      // Update sun shader time.
      // (sun mat is index 0)
      if (state.planetMats[0]) state.planetMats[0].uniforms.uTime.value = t;

      updateSpotlight(t);

      // Refresh camera frustum offset (in case per-node rect changed).
      const nr = state.ctx.nodeRect, ws = state.ctx.wallSize;
      state.camera.setViewOffset(ws.w, ws.h, nr.x, nr.y, nr.w, nr.h);

      state.renderer.render(state.scene, state.camera);
    },

    onMessage() { /* fully client-deterministic */ },

    async teardown() {
      if (state.renderer) {
        state.renderer.dispose();
        state.renderer.forceContextLoss();
        state.renderer = null;
      }
      if (state.scene) {
        state.scene.traverse(o => {
          if (o.geometry) o.geometry.dispose();
          if (o.material) {
            const mats = Array.isArray(o.material) ? o.material : [o.material];
            for (const m of mats) {
              if (m.map) m.map.dispose();
              m.dispose();
            }
          }
        });
      }
      state.scene = null;
      state.camera = null;
      state.planetMeshes = []; state.planetMats = [];
      state.labelSprites = []; state.calloutSprite = null;
      state.ringMesh = null; state.starMat = null; state.dustMats = [];
    }
  };
}
