export default function ImageSceneFactory() {
  const state = {
    container: null,
    stageEl: null,
  };

  return {
    name: 'image',

    async init(ctx) {
      const params = ctx.sceneParams || {};
      const url = params.url;
      const mode = params.mode === 'mirror' ? 'mirror' : 'span';
      if (!url) {
        console.warn('image scene: missing url in params', params);
        return;
      }

      state.stageEl = document.getElementById('stage');
      if (state.stageEl) state.stageEl.style.display = 'none';

      const container = document.createElement('div');
      container.id = 'image-scene';
      container.style.cssText = 'position:fixed;inset:0;overflow:hidden;background:#000;z-index:1;';
      state.container = container;

      const img = document.createElement('img');
      img.src = url;
      img.alt = '';
      img.style.display = 'block';

      if (mode === 'span') {
        const { wallSize, perNodeSize, nodeRect } = ctx;
        const widthPct   = (wallSize.w / perNodeSize.w) * 100;
        const heightPct  = (wallSize.h / perNodeSize.h) * 100;
        const offsetXPct = (nodeRect.x / perNodeSize.w) * 100;
        const offsetYPct = (nodeRect.y / perNodeSize.h) * 100;
        img.style.position  = 'absolute';
        img.style.width     = widthPct + '%';
        img.style.height    = heightPct + '%';
        img.style.left      = '0';
        img.style.top       = '0';
        img.style.transform = 'translate(' + (-offsetXPct) + '%, ' + (-offsetYPct) + '%)';
        img.style.objectFit = 'fill';
      } else {
        img.style.width     = '100%';
        img.style.height    = '100%';
        img.style.objectFit = 'contain';
      }

      container.appendChild(img);
      document.body.appendChild(container);
    },

    tick(_t) {},

    onMessage(_msg) {},

    async teardown() {
      if (state.container && state.container.parentNode) {
        state.container.parentNode.removeChild(state.container);
      }
      state.container = null;
      if (state.stageEl) {
        state.stageEl.style.display = '';
        state.stageEl = null;
      }
    },
  };
}
