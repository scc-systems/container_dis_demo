# wallsync — 45-display videowall sync

A Go WebSocket server that orchestrates 5 remote Chromium instances driving a **15 × 3 = 45-display** videowall. All timing is anchored to a server-published epoch `T₀`; clients render frames from `t = (Date.now() − T₀) / 1000` — never local delta time — so all 5 walls stay in lockstep across a campus LAN with trusted NTP.

Three switchable scenes:

- **three-unified** — a single Three.js world (wireframe terrain + orbiting flyers) split across nodes with `camera.setViewOffset`.
- **solar-panorama** — the whole solar system laid out horizontally across all 15 tiles, distances to true scale (1 AU = 900 wall units), with an auto-cycling spotlight that grows each planet in turn and shows a sci-fi callout of its physical stats. Includes the asteroid belt, Kuiper belt, Saturn's rings, and a twinkling starfield.
- **warp-jump** — radial star streaks exploding out from a vanishing point at the wall centre, with periodic shockwave rings, exploiting the 9:1 aspect for pure warp-speed feel.

Plus a **video** scene: upload an `.mp4`, `.webm`, or `.mov` from the admin page and play it back synchronised across every node. Two layouts are supported per play — **mirrored** (the whole video on every node) or **stretched** (one video spanning the full 28800 × 3240 wall, each node showing its slice). Videos loop until you hit **stop video** on the admin page.

## Running locally (single machine, 5 tabs)

```
go build ./cmd/wallsync
./wallsync
```

Then open these 6 tabs in Chrome / Chromium at fullscreen 100% zoom:

- `http://localhost:8080/wall?node=0`
- `http://localhost:8080/wall?node=1`
- `http://localhost:8080/wall?node=2`
- `http://localhost:8080/wall?node=3`
- `http://localhost:8080/wall?node=4`
- `http://localhost:8080/admin`

Press `d` on any wall tab to toggle the debug HUD (frame count, T₀, current t, DPR check).

Use the buttons on the admin page to switch scenes. Each switch broadcasts a `scene_change` with an epoch ~800 ms in the future — all tabs cut over on that instant.

## Docker

```
docker compose up --build
```

The container serves on port 8080 with all frontend assets embedded.

## Deploying to the 5 remote Chromium hosts

On each of the 5 hosts, install Chromium and launch pointed at the server (replace `videowall-server.campus.local:8080` and set `NODE_ID` per host):

```
NODE_ID=0  # 0..4 per host

chromium \
  --kiosk \
  --start-fullscreen \
  --window-size=5760,3240 \
  --window-position=0,0 \
  --force-device-scale-factor=1 \
  --autoplay-policy=no-user-gesture-required \
  --disable-pinch \
  --overscroll-history-navigation=0 \
  --noerrdialogs \
  --disable-session-crashed-bubble \
  --incognito \
  "http://videowall-server.campus.local:8080/wall?node=${NODE_ID}"
```

`--force-device-scale-factor=1` is essential — DPR mismatches across hosts will produce visible seam shimmer. The client-side debug HUD flashes red if `window.devicePixelRatio !== 1`.

Ensure all 5 hosts sync to the same campus NTP source. The admin page's per-node "offset" column shows warnings above ±25 ms.

## Config

Environment variables:

| var                        | default          | meaning                                   |
| -------------------------- | ---------------- | ----------------------------------------- |
| `WALLSYNC_ADDR`            | `:8080`          | HTTP bind address                          |
| `WALLSYNC_TICK_HZ`         | `30`             | broadcast rate                            |
| `WALLSYNC_PING_MS`         | `1000`           | heartbeat interval (ms)                   |
| `WALLSYNC_INITIAL_SCENE`   | `idle`           | scene to boot into                        |
| `WALLSYNC_AUTOROTATE_S`    | `0` (off)        | if >0, rotate scenes every N seconds      |
| `WALLSYNC_MEDIA_DIR`       | `./data/videos`  | on-disk root for uploaded videos          |
| `WALLSYNC_MAX_UPLOAD_MB`   | `500`            | per-upload size cap (megabytes)           |

### Uploading a video

From `/admin`, pick a file (`.mp4`, `.webm`, `.mov`), click **upload**, then choose **play mirrored** or **play stretched** on its row. **stop video** returns the wall to `three-unified`. Files land in `WALLSYNC_MEDIA_DIR` alongside a `.json` sidecar. In Docker Compose that path is backed by the named volume `wallsync-media`, so uploads persist across container restarts (inspect via `docker volume inspect wallsync-media`). Video sync uses the same server-published epoch as the procedural scenes — each node seeks `<video>.currentTime` to `wallClock() % duration` and drift-corrects (nudged `playbackRate` for small skew, hard-seek beyond 250 ms).

## Architecture

- `cmd/wallsync/main.go` — HTTP mux + wiring
- `internal/config` — env-driven config struct (wall geometry, timings)
- `internal/protocol` — JSON envelope + typed messages (hello, welcome, ping, pong, scene_change, tick, admin_status)
- `internal/timing` — clock helpers, PingBook + rolling RTT/offset stats
- `internal/hub` — WS accept, node registry (5 slots), supersede semantics on reconnect, non-blocking broadcast fan-out
- `internal/scene` — Director + Switch (future-epoch + broadcast) + optional auto-rotate
- `internal/feeds` — single 30 Hz ticker; per-scene payload (Three camera, Matter spawner, Pixi FFT/radar)
- `internal/media` — on-disk video store (Save/List/Delete) backing the video scene
- `internal/admin` — status snapshot, admin HTML page, scene switch + video upload/list/delete/play/stop handlers
- `internal/web` — `embed.FS` for the frontend

Frontend (all under `internal/web/static/`):

- `js/main.js` — bootstrap, master `requestAnimationFrame` loop
- `js/shell/wallClock.js` — `wallClock()`, `setEpoch`, `hasEpoch`
- `js/shell/wsClient.js` — WS handshake, reconnect, ping/pong
- `js/shell/sceneHost.js` — scene registry, mount at `epoch_ms`, dispatch messages
- `js/shell/wallGeometry.js` — wall constants, `nodeRect(N)`, `nodeRectForCamera(N)`
- `js/shell/debugHud.js` — toggleable overlay (press `d`)
- `js/scenes/{threeUnified,solarPanorama,warpJump,video}/index.js` — the four scenes
