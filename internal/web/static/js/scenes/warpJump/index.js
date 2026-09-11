import * as THREE from 'three';

// Warp / hyperspace jump: radial star streaks from a central vanishing
// point exploding outward across the whole wall. Instanced thin quads
// so streaks aren't limited by driver 1-pixel line width.

const STREAK_COUNT = 1400;
const RING_COUNT = 3;
const N_STARS = 3200;

const STREAK_VERT = `
attribute float aSeed;
uniform float uTime;
uniform vec2 uCenter;      // wall coords of the vanishing point
uniform vec2 uWall;        // wallW, wallH
uniform float uWarp;       // 0..1 intensity ramp

// pseudo-random from seed
float sfract(float x) { return fract(sin(x * 91.13) * 43758.5453); }

void main() {
  // Two vertices per instance encode the streak tail (uv.x=0) and head (uv.x=1).
  float side = uv.x;

  // Per-streak lifecycle: each has a lifespan (1..2s) offset by seed.
  float life = 1.0 + fract(aSeed * 7.31) * 1.6;
  float t0 = aSeed * 40.0;
  float phase = fract((uTime + t0) / life);         // 0..1

  // Direction from center: random angle by seed.
  float angle = fract(aSeed * 91.7) * 6.2831853;
  vec2 dir = vec2(cos(angle), sin(angle));

  // Radius from center grows with phase; head leads, tail follows.
  float radiusHead = phase * (uWall.x * 0.9);
  float streakLen  = mix(30.0, 320.0, fract(aSeed * 3.41)) + phase * 900.0;
  float radiusTail = max(0.0, radiusHead - streakLen);

  float r = mix(radiusTail, radiusHead, side);
  vec2 posXY = uCenter + dir * r;

  // Perpendicular offset for the quad thickness (billboard in view plane).
  vec2 perp = vec2(-dir.y, dir.x);
  float thickness = mix(4.0, 12.0, fract(aSeed * 5.7)) * (0.4 + uWarp * 1.4);
  posXY += perp * (uv.y - 0.5) * 2.0 * thickness;

  gl_Position = projectionMatrix * modelViewMatrix * vec4(posXY, 0.0, 1.0);
}
`;

const STREAK_FRAG = `
precision highp float;
uniform float uWarp;
varying vec2 vUv;
void main() {
  // uv.x = 0 at tail, 1 at head. Fade tail out; hot at head.
  float t = vUv.x;
  float thick = 1.0 - abs(vUv.y - 0.5) * 2.0;
  float alpha = smoothstep(0.0, 0.35, t) * pow(thick, 1.3);
  vec3 cold = vec3(0.25, 0.55, 1.0);
  vec3 hot  = vec3(1.0, 0.95, 0.85);
  vec3 col = mix(cold, hot, pow(t, 2.0));
  col *= (0.9 + uWarp * 1.6);
  gl_FragColor = vec4(col, alpha * uWarp);
}
`;

// Passthrough that also forwards uv.
const STREAK_VERT_WRAPPED = `
varying vec2 vUv;
${STREAK_VERT.replace('void main() {', 'void main() { vUv = uv;')}
`;

const RING_VERT = `
varying vec2 vUv;
void main() {
  vUv = uv;
  gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
`;
const RING_FRAG = `
precision highp float;
uniform float uProgress;   // 0..1, 0 spawn (small), 1 dissipate (big)
uniform float uAlpha;
varying vec2 vUv;
void main() {
  vec2 uv = vUv - 0.5;
  float d = length(uv) * 2.0;
  if (d > 1.0) discard;
  float ringW = 0.06;
  float mask = smoothstep(1.0 - ringW * 2.0, 1.0 - ringW, d) - smoothstep(1.0 - ringW, 1.0, d);
  float col = mask * uAlpha;
  vec3 c = mix(vec3(0.15, 0.6, 1.0), vec3(1.0, 0.95, 0.9), 0.6);
  gl_FragColor = vec4(c * col, col);
}
`;

const STAR_VERT = `
attribute float aSeed;
uniform float uAspect;
uniform float uWarp;
uniform vec2 uCenter;
uniform float uTime;
void main() {
  // Static-ish stars that begin to drift outward as uWarp ramps up.
  vec4 mv = modelViewMatrix * vec4(position, 1.0);
  vec2 posXY = position.xy;
  float dx = posXY.x - uCenter.x;
  float dy = posXY.y - uCenter.y;
  float r = length(vec2(dx, dy)) + 0.001;
  vec2 dir = vec2(dx, dy) / r;
  float outward = uWarp * uWarp * 400.0 * (0.5 + fract(aSeed * 3.7));
  posXY += dir * outward;

  gl_Position = projectionMatrix * modelViewMatrix * vec4(posXY, position.z, 1.0);
  gl_PointSize = mix(1.8, 4.4, fract(aSeed * 7.7)) * uAspect;
}
`;
const STAR_FRAG = `
precision highp float;
void main() {
  vec2 uv = gl_PointCoord - 0.5;
  float d = length(uv);
  if (d > 0.5) discard;
  float a = smoothstep(0.5, 0.0, d) * 0.6;
  gl_FragColor = vec4(vec3(1.0, 0.97, 0.92) * a, a);
}
`;

function buildStreakGeometry(count, seed) {
  // One quad (4 verts, 2 tris) per streak.
  const geo = new THREE.InstancedBufferGeometry();
  const positions = new Float32Array([
    0, 0, 0,
    1, 0, 0,
    1, 1, 0,
    0, 1, 0,
  ]);
  const uvs = new Float32Array([
    0, 0,
    1, 0,
    1, 1,
    0, 1,
  ]);
  const index = new Uint16Array([0, 1, 2, 0, 2, 3]);
  geo.setAttribute('position', new THREE.BufferAttribute(positions, 3));
  geo.setAttribute('uv', new THREE.BufferAttribute(uvs, 2));
  geo.setIndex(new THREE.BufferAttribute(index, 1));
  const seeds = new Float32Array(count);
  let s = (seed >>> 0) || 7;
  for (let i = 0; i < count; i++) {
    s = (s * 1664525 + 1013904223) >>> 0;
    seeds[i] = (s >>> 0) / 4294967296;
  }
  geo.setAttribute('aSeed', new THREE.InstancedBufferAttribute(seeds, 1));
  geo.instanceCount = count;
  geo.boundingSphere = new THREE.Sphere(new THREE.Vector3(0, 0, 0), 1e7);
  return geo;
}

function buildStarGeometry(count, wallW, wallH, seed) {
  const geo = new THREE.BufferGeometry();
  const positions = new Float32Array(count * 3);
  const seeds = new Float32Array(count);
  let s = (seed >>> 0) || 3;
  const rnd = () => { s = (s * 1664525 + 1013904223) >>> 0; return s / 4294967296; };
  for (let i = 0; i < count; i++) {
    positions[i*3+0] = -800 + rnd() * (wallW + 1600);
    positions[i*3+1] = -400 + rnd() * (wallH + 800);
    positions[i*3+2] = -100;
    seeds[i] = rnd();
  }
  geo.setAttribute('position', new THREE.BufferAttribute(positions, 3));
  geo.setAttribute('aSeed', new THREE.BufferAttribute(seeds, 1));
  geo.boundingSphere = new THREE.Sphere(new THREE.Vector3(wallW/2, wallH/2, 0), wallW);
  return geo;
}

export default function WarpJumpFactory() {
  const state = {
    renderer: null, scene: null, camera: null, ctx: null,
    streakMat: null, starMat: null, rings: [],
  };

  return {
    name: 'warp-jump',

    async init(ctx) {
      state.ctx = ctx;
      const { canvas, nodeRect, wallSize, perNodeSize, seed } = ctx;

      const renderer = new THREE.WebGLRenderer({
        canvas, antialias: true, powerPreference: 'high-performance',
      });
      renderer.setPixelRatio(1);
      renderer.setSize(perNodeSize.w, perNodeSize.h, false);
      renderer.setClearColor(0x03040c, 1);
      state.renderer = renderer;

      const scene = new THREE.Scene();
      state.scene = scene;

      // Orthographic camera in wall coords so streaks map 1:1 to wall pixels.
      const cam = new THREE.OrthographicCamera(0, wallSize.w, wallSize.h, 0, -1000, 1000);
      cam.position.z = 100;
      cam.setViewOffset(wallSize.w, wallSize.h, nodeRect.x, nodeRect.y, nodeRect.w, nodeRect.h);
      state.camera = cam;

      const center = new THREE.Vector2(wallSize.w / 2, wallSize.h / 2);

      // Background stars.
      const starMat = new THREE.ShaderMaterial({
        uniforms: {
          uAspect: { value: 1.0 },
          uWarp:   { value: 0.0 },
          uCenter: { value: center },
          uTime:   { value: 0.0 },
        },
        vertexShader: STAR_VERT, fragmentShader: STAR_FRAG,
        transparent: true, depthWrite: false, blending: THREE.AdditiveBlending,
      });
      state.starMat = starMat;
      scene.add(new THREE.Points(
        buildStarGeometry(N_STARS, wallSize.w, wallSize.h, (seed || 1) ^ 0xa5a5),
        starMat,
      ));

      // Streaks.
      const streakMat = new THREE.ShaderMaterial({
        uniforms: {
          uTime:   { value: 0 },
          uCenter: { value: center },
          uWall:   { value: new THREE.Vector2(wallSize.w, wallSize.h) },
          uWarp:   { value: 0.0 },
        },
        vertexShader: STREAK_VERT_WRAPPED,
        fragmentShader: STREAK_FRAG,
        transparent: true, depthWrite: false, blending: THREE.AdditiveBlending,
      });
      state.streakMat = streakMat;
      const streakGeo = buildStreakGeometry(STREAK_COUNT, seed || 1);
      const streakMesh = new THREE.Mesh(streakGeo, streakMat);
      scene.add(streakMesh);

      // Shockwave rings.
      const ringGeo = new THREE.PlaneGeometry(1, 1);
      for (let i = 0; i < RING_COUNT; i++) {
        const mat = new THREE.ShaderMaterial({
          uniforms: {
            uProgress: { value: i / RING_COUNT },
            uAlpha:    { value: 0.0 },
          },
          vertexShader: RING_VERT, fragmentShader: RING_FRAG,
          transparent: true, depthWrite: false, blending: THREE.AdditiveBlending,
        });
        const mesh = new THREE.Mesh(ringGeo, mat);
        mesh.position.set(center.x, center.y, 0);
        scene.add(mesh);
        state.rings.push({ mesh, mat, offset: i * (5.0 / RING_COUNT) });
      }
    },

    tick(t) {
      if (!state.renderer) return;
      const aspectFactor = state.ctx.wallSize.h / state.ctx.perNodeSize.h;

      // Warp intensity ramps in over the first 3 seconds, then holds.
      const warp = Math.min(1.0, t / 3.0);
      state.streakMat.uniforms.uTime.value = t;
      state.streakMat.uniforms.uWarp.value = warp;
      state.starMat.uniforms.uWarp.value = warp;
      state.starMat.uniforms.uAspect.value = aspectFactor;
      state.starMat.uniforms.uTime.value = t;

      // Shockwave rings: each cycles every 5s, staggered.
      const maxR = Math.max(state.ctx.wallSize.w, state.ctx.wallSize.h) * 1.3;
      for (const r of state.rings) {
        const local = ((t + r.offset) % 5.0) / 5.0; // 0..1
        const size = 200 + local * maxR;
        r.mesh.scale.set(size, size, 1);
        // Alpha: fades in first 15%, out last 40%, hidden before warp ramps.
        const fadeIn  = Math.min(1, local / 0.15);
        const fadeOut = 1 - Math.max(0, (local - 0.6) / 0.4);
        r.mat.uniforms.uAlpha.value = fadeIn * fadeOut * warp * 0.75;
        r.mat.uniforms.uProgress.value = local;
      }

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
            for (const m of mats) m.dispose();
          }
        });
      }
      state.scene = null; state.camera = null;
      state.streakMat = null; state.starMat = null; state.rings = [];
    }
  };
}
