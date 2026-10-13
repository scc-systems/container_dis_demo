# wallsync: a reference server for a 5-node, 45-display videowall

A Go + WebSocket server that drives the videowall: five Chromium instances, five URLs, and a shared clock over WebSockets keeping all five walls in lockstep. Fork it, copy ideas out of it, or just run it as-is.

The wall is **15 displays wide x 3 tall (45 total)**, driven by **5 Chromium instances** each rendering a **3x3** slice. The server broadcasts a shared epoch `T0` over WebSockets; every client renders from `t = (Date.now() - T0) / 1000` so all five walls stay in lockstep.

> **Live demo:** <https://dis-client-manager.scc.lancs.ac.uk/wallsync/admin>. Switch scenes and upload videos; the physical wall updates in real time.

## The wall, physically

```
                     Physical wall: 15 displays wide × 3 tall  (45 total)

       ┌─────────────────┬─────────────────┬─────────────────┬─────────────────┬─────────────────┐
       │  Chromium  #1   │  Chromium  #2   │  Chromium  #3   │  Chromium  #4   │  Chromium  #5   │
       │  /wall?node=0   │  /wall?node=1   │  /wall?node=2   │  /wall?node=3   │  /wall?node=4   │
       │                 │                 │                 │                 │                 │
       │  ┌───┬───┬───┐  │  ┌───┬───┬───┐  │  ┌───┬───┬───┐  │  ┌───┬───┬───┐  │  ┌───┬───┬───┐  │
       │  │   │   │   │  │  │   │   │   │  │  │   │   │   │  │  │   │   │   │  │  │   │   │   │  │
       │  ├───┼───┼───┤  │  ├───┼───┼───┤  │  ├───┼───┼───┤  │  ├───┼───┼───┤  │  ├───┼───┼───┤  │
       │  │   │   │   │  │  │   │   │   │  │  │   │   │   │  │  │   │   │   │  │  │   │   │   │  │
       │  ├───┼───┼───┤  │  ├───┼───┼───┤  │  ├───┼───┼───┤  │  ├───┼───┼───┤  │  ├───┼───┼───┤  │
       │  │   │   │   │  │  │   │   │   │  │  │   │   │   │  │  │   │   │   │  │  │   │   │   │  │
       │  └───┴───┴───┘  │  └───┴───┴───┘  │  └───┴───┴───┘  │  └───┴───┴───┘  │  └───┴───┴───┘  │
       │                 │                 │                 │                 │  <<< AUDIO >>>  │
       │  USB-IP KB/M    │  USB-IP KB/M    │  USB-IP KB/M    │  USB-IP KB/M    │  USB-IP KB/M    │
       └─────────────────┴─────────────────┴─────────────────┴─────────────────┴─────────────────┘
                  ▲                 ▲                 ▲                 ▲                 ▲
                  │                 │                 │                 │                 │
                  │                 │                 │                 │                 │
                  └─────────────────┴─── Go server ───┴─── WebSocket ───┴─────────────────┘
                                   (shared epoch T₀, 30 Hz ticks, scene_change)
```

Hardware notes:

- Each of the 5 Chromium instances drives a 3x3 block of displays (9 physical panels, usually 1920x1080 each, giving a 5760x3240 Chromium window per node).
- Keyboard and mouse are available on every block over USB-IP. Plug a USB keyboard/mouse into any node's input server and it shows up to the Chromium instance as a normal HID device. Treat the wall as five big interactive surfaces.
- Only Chromium #5 (`node=4`) outputs audio. The other four nodes have no speakers wired in, so sound-reactive scenes should gate audio on `node === 4`.

## Running it in GitHub Codespaces

A Go 1.23 devcontainer is included (`.devcontainer/devcontainer.json`), so:

1. Click **Code > Codespaces > Create codespace on main**.
2. Once the container is up, run:

   ```
   go run ./cmd/wallsync
   ```

3. Codespaces will forward port **8080**. The devcontainer sets its visibility to **public**, so you get a URL of the form `https://<codespace>-8080.app.github.dev` that anyone (including the physical wall's Chromium hosts) can reach.
4. Open five tabs, one per node, at:

   ```
   https://<codespace>-8080.app.github.dev/wall?node=0
   https://<codespace>-8080.app.github.dev/wall?node=1
   https://<codespace>-8080.app.github.dev/wall?node=2
   https://<codespace>-8080.app.github.dev/wall?node=3
   https://<codespace>-8080.app.github.dev/wall?node=4
   ```

   Plus `/admin` for the control UI. Point the physical wall at the same URLs by setting each Chromium host's launch URL to the matching `?node=N`.

To keep the forward private, change `"visibility": "public"` to `"private"` in `.devcontainer/devcontainer.json` and sign the wall's Chromium hosts in to your GitHub account; Codespaces will issue them a session cookie.

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

Press `d` on any wall tab to toggle the debug HUD (frame count, T0, current t, DPR check).

Use the buttons on the admin page to switch scenes. Each switch broadcasts a `scene_change` with an epoch ~800 ms in the future, so all tabs cut over on the same instant.

## Modifying it

- **Add a new scene.** Scenes live under `internal/web/static/js/scenes/<name>/index.js` and register themselves with the shell. Each one gets `t` (seconds since `T0`), its node index, and the wall geometry, then draws its slice. Copy `warpJump` as a starting point.
- **Point the wall at a different server.** The 5 Chromium hosts aren't tied to this repo. Serve a `/wall?node=N` page from any WebSocket-speaking web server and point them at that. The protocol lives in `internal/protocol/` if you want to speak the same `hello` / `welcome` / `tick` / `scene_change` dialect, or invent your own.
- **Make it interactive.** USB-IP keyboard/mouse input lands in the Chromium window like any laptop. Register normal DOM `keydown` / `pointermove` listeners inside your scene.
- **Use node 4 for audio.** `WebAudio` or an `<audio>` element gated on `nodeId === 4` is enough. For tight sync against the visual nodes, schedule the audio start on a `T0`-aligned future epoch the same way scene switches work.

Three scenes ship with the repo:

- **three-unified**: a single Three.js world (wireframe terrain + orbiting flyers) split across nodes with `camera.setViewOffset`.
- **solar-panorama**: the solar system laid out horizontally across all 15 tiles, distances to true scale (1 AU = 900 wall units), with an auto-cycling spotlight that grows each planet in turn and shows a callout of its physical stats. Includes the asteroid belt, Kuiper belt, Saturn's rings, and a starfield.
- **warp-jump**: radial star streaks from a vanishing point at the wall centre, with periodic shockwave rings.

Plus a **video** scene: upload an `.mp4`, `.webm`, or `.mov` from the admin page and play it back synchronised across every node. Two layouts per play: **mirrored** (the whole video on every node) or **stretched** (one video spanning the full 28800 x 3240 wall, each node showing its slice). Videos loop until you hit **stop video**.

## Uploading a video

From `/admin`, pick a file (`.mp4`, `.webm`, `.mov`), click **upload**, then choose **play mirrored** or **play stretched** on its row. **stop video** returns the wall to `three-unified`. Files land in `./data/videos` by default (configurable via `WALLSYNC_MEDIA_DIR`; see [DEPLOYMENT.md](./DEPLOYMENT.md)) alongside a `.json` sidecar. Video sync uses the same server-published epoch as the procedural scenes: each node seeks `<video>.currentTime` to `wallClock() % duration` and drift-corrects (nudged `playbackRate` for small skew, hard-seek beyond 250 ms).

## Deployment

Running against the physical wall, Docker Compose, and server env vars live in [DEPLOYMENT.md](./DEPLOYMENT.md).

## Architecture

- `cmd/wallsync/main.go`: HTTP mux + wiring
- `internal/config`: env-driven config struct (wall geometry, timings)
- `internal/protocol`: JSON envelope + typed messages (hello, welcome, ping, pong, scene_change, tick, admin_status)
- `internal/timing`: clock helpers, PingBook + rolling RTT/offset stats
- `internal/hub`: WS accept, node registry (5 slots), supersede semantics on reconnect, non-blocking broadcast fan-out
- `internal/scene`: Director + Switch (future-epoch + broadcast) + optional auto-rotate
- `internal/feeds`: single 30 Hz ticker; per-scene payload (Three camera, Matter spawner, Pixi FFT/radar)
- `internal/media`: on-disk video store (Save/List/Delete) backing the video scene
- `internal/admin`: status snapshot, admin HTML page, scene switch + video upload/list/delete/play/stop handlers
- `internal/web`: `embed.FS` for the frontend

Frontend (all under `internal/web/static/`):

- `js/main.js`: bootstrap, master `requestAnimationFrame` loop
- `js/shell/wallClock.js`: `wallClock()`, `setEpoch`, `hasEpoch`
- `js/shell/wsClient.js`: WS handshake, reconnect, ping/pong
- `js/shell/sceneHost.js`: scene registry, mount at `epoch_ms`, dispatch messages
- `js/shell/wallGeometry.js`: wall constants, `nodeRect(N)`, `nodeRectForCamera(N)`
- `js/shell/debugHud.js`: toggleable overlay (press `d`)
- `js/scenes/{threeUnified,solarPanorama,warpJump,video}/index.js`: the four scenes

## License

**Released into the Public Domain under the [BSD Zero Clause License](./LICENSE) (0BSD).**

Do whatever you want with it: copy, modify, distribute, use commercially, no attribution required, no warranty.

Bundled third-party code maintains their own license:

- **three.js** (`internal/web/static/js/vendor/three.module.js`) — MIT, Copyright 2010-2024 Three.js Authors. Keep the license header intact if you redistribute the file.
- **github.com/coder/websocket** — ISC. Pulled at build time via Go modules; include its license in binary distributions if you want to be thorough.
