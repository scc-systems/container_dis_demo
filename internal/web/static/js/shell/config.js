export const LOCKED_DPR = 1;

const BASE_PATH = document.querySelector('meta[name="base-path"]')?.content || '';

// Where to connect the WebSocket, given the current page origin and node id.
export function wsUrl(nodeId) {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${location.host}${BASE_PATH}/ws?node=${nodeId}`;
}
