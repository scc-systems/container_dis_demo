let cfg = {
  wallWidth: 28800,
  wallHeight: 3240,
  perNodeWidth: 5760,
  perNodeHeight: 3240,
  nodeCount: 5,
};

export function updateFromWelcome(payload) {
  if (payload.wall_width)     cfg.wallWidth = payload.wall_width;
  if (payload.wall_height)    cfg.wallHeight = payload.wall_height;
  if (payload.per_node_width) cfg.perNodeWidth = payload.per_node_width;
  if (payload.per_node_height)cfg.perNodeHeight = payload.per_node_height;
  if (payload.node_count)     cfg.nodeCount = payload.node_count;
}

export function wallSize() {
  return { w: cfg.wallWidth, h: cfg.wallHeight };
}

export function perNodeSize() {
  return { w: cfg.perNodeWidth, h: cfg.perNodeHeight };
}

export function nodeRect(nodeId) {
  return {
    x: nodeId * cfg.perNodeWidth,
    y: 0,
    w: cfg.perNodeWidth,
    h: cfg.perNodeHeight,
  };
}

// Convenience for THREE.PerspectiveCamera.setViewOffset.
export function nodeRectForCamera(nodeId) {
  return {
    fullW: cfg.wallWidth,
    fullH: cfg.wallHeight,
    xOffset: nodeId * cfg.perNodeWidth,
    yOffset: 0,
    w: cfg.perNodeWidth,
    h: cfg.perNodeHeight,
  };
}
