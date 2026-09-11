import * as THREE from 'three';

// The wall is 28800 x 3240 by default (aspect ≈ 8.888).
// Content lives centered at (0,0,0) in world units where 1 unit = 1 wall pixel / 30 or so.
// We use signed coords for float precision at large offsets.

const TERRAIN_COLS = 300;
const TERRAIN_ROWS = 60;
const TERRAIN_WORLD_W = 12000; // world units, x
const TERRAIN_WORLD_D = 2400;  // world units, z
const FLYER_COUNT = 16;

const VERT = `
uniform float uTime;
uniform float uAmp;
uniform float uKx;
uniform float uKz;
varying float vHeight;
void main() {
  vec3 p = position;
  float h = sin(p.x * uKx + uTime * 0.9) * cos(p.z * uKz + uTime * 0.7)
          + 0.4 * sin(p.x * uKx * 2.7 - uTime * 1.6)
          + 0.3 * cos(p.z * uKz * 3.1 + uTime * 1.2);
  p.y += h * uAmp;
  vHeight = h;
  gl_Position = projectionMatrix * modelViewMatrix * vec4(p, 1.0);
}
`;

const FRAG = `
precision highp float;
varying float vHeight;
void main() {
  float g = clamp(0.5 + 0.5 * vHeight, 0.0, 1.0);
  vec3 col = mix(vec3(0.04, 0.28, 0.55), vec3(0.55, 0.9, 1.0), g);
  gl_FragColor = vec4(col, 1.0);
}
`;

export default function ThreeUnifiedFactory() {
  const state = {
    renderer: null,
    scene: null,
    camera: null,
    terrainMat: null,
    flyers: [],
    animFrame: null,
    ctx: null,
  };

  function buildTerrain() {
    const geo = new THREE.PlaneGeometry(TERRAIN_WORLD_W, TERRAIN_WORLD_D, TERRAIN_COLS, TERRAIN_ROWS);
    geo.rotateX(-Math.PI / 2);
    const mat = new THREE.ShaderMaterial({
      uniforms: {
        uTime: { value: 0 },
        uAmp:  { value: 90 },
        uKx:   { value: 0.0022 },
        uKz:   { value: 0.006 },
      },
      vertexShader: VERT,
      fragmentShader: FRAG,
      wireframe: true,
      transparent: false,
    });
    state.terrainMat = mat;
    const mesh = new THREE.Mesh(geo, mat);
    mesh.position.y = -200;
    state.scene.add(mesh);
  }

  function buildFlyers() {
    const geo = new THREE.IcosahedronGeometry(60, 0);
    for (let i = 0; i < FLYER_COUNT; i++) {
      const hue = (i * 0.11) % 1;
      const col = new THREE.Color().setHSL(hue, 0.65, 0.6);
      const mat = new THREE.MeshBasicMaterial({ color: col, wireframe: true });
      const mesh = new THREE.Mesh(geo, mat);
      state.scene.add(mesh);
      state.flyers.push({ mesh, phase: i * 0.7, ySwing: 300 + (i % 5) * 40, speed: 0.35 + i * 0.02 });
    }
  }

  function positionFlyer(f, t) {
    // Trajectory that crosses full wall x range (-7000..7000) parametrically from t.
    const period = 22.0 - 0.6 * (f.speed - 0.35);
    const cycle = ((t * f.speed) % period) / period; // 0..1
    const x = -7500 + 15000 * cycle;
    const y = 400 + f.ySwing * Math.sin(t * 0.8 + f.phase);
    const z = 700 * Math.sin(t * 0.3 + f.phase * 1.7);
    f.mesh.position.set(x, y, z);
    f.mesh.rotation.set(t * 0.7 + f.phase, t * 0.5, 0);
  }

  return {
    name: 'three-unified',

    async init(ctx) {
      state.ctx = ctx;
      const { canvas, nodeRect, wallSize, perNodeSize } = ctx;

      const renderer = new THREE.WebGLRenderer({
        canvas,
        antialias: true,
        powerPreference: 'high-performance',
        preserveDrawingBuffer: false,
      });
      renderer.setPixelRatio(1);
      renderer.setSize(perNodeSize.w, perNodeSize.h, false);
      renderer.setClearColor(0x02060c, 1);
      state.renderer = renderer;

      const scene = new THREE.Scene();
      scene.fog = new THREE.Fog(0x02060c, 4000, 12000);
      state.scene = scene;

      // Full-wall camera; frustum offset crops to this node.
      const aspect = wallSize.w / wallSize.h;
      const camera = new THREE.PerspectiveCamera(35, aspect, 10, 20000);
      camera.position.set(0, 900, 3500);
      camera.lookAt(0, 0, 0);
      camera.setViewOffset(wallSize.w, wallSize.h, nodeRect.x, nodeRect.y, nodeRect.w, nodeRect.h);
      state.camera = camera;

      const dir = new THREE.DirectionalLight(0xffffff, 0.6);
      dir.position.set(2, 4, 2);
      scene.add(dir);
      scene.add(new THREE.AmbientLight(0x223344, 0.6));

      buildTerrain();
      buildFlyers();
    },

    tick(t) {
      if (!state.renderer || !state.terrainMat) return;
      state.terrainMat.uniforms.uTime.value = t;

      // Slow global camera orbit anchored on t (all nodes agree).
      const orbit = t * 0.06;
      state.camera.position.x = Math.sin(orbit) * 400;
      state.camera.position.z = 3400 + Math.cos(orbit) * 400;
      state.camera.lookAt(Math.sin(orbit * 1.4) * 100, 100, 0);
      // Re-apply the frustum offset (lookAt resets the view matrix but not the offset).

      for (const f of state.flyers) positionFlyer(f, t);
      state.renderer.render(state.scene, state.camera);
    },

    onMessage(msg) { /* no per-tick server state needed for this scene */ },

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
            if (Array.isArray(o.material)) o.material.forEach(m => m.dispose());
            else o.material.dispose();
          }
        });
      }
      state.scene = null;
      state.camera = null;
      state.terrainMat = null;
      state.flyers = [];
    }
  };
}
