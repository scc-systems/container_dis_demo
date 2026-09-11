let T0Ms = null;
let lastRttMs = 0;
let lastOffsetMs = 0;

export function setEpoch(epochMs) {
  T0Ms = epochMs;
}

export function hasEpoch() {
  return T0Ms !== null;
}

export function epochMs() {
  return T0Ms;
}

// Wall-clock seconds since T0. Negative if we're pre-epoch.
export function wallClock() {
  if (T0Ms === null) return 0;
  return (Date.now() - T0Ms) / 1000;
}

// Wall-clock milliseconds since T0 (int).
export function wallClockMs() {
  if (T0Ms === null) return 0;
  return Date.now() - T0Ms;
}

export function setRtt(ms) { lastRttMs = ms; }
export function getRtt() { return lastRttMs; }
export function setOffset(ms) { lastOffsetMs = ms; }
export function getOffset() { return lastOffsetMs; }
