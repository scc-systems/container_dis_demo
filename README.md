# wallsync: a reference server for a 5-node, 45-display videowall

A Go + WebSocket server that drives the videowall from five Chromium instances, each at its own URL. Fork it, copy ideas out of it, or just run it as-is.

The wall is 15 displays wide and 3 tall (45 in total). Each of the 5 Chromium instances renders a 3x3 slice. The server broadcasts a shared epoch `T0` over WebSockets. Every client renders from `t = (Date.now() - T0) / 1000`, so all five walls stay in lockstep.

> **Live demo:** <https://dis-client-manager.scc.lancs.ac.uk/wallsync/admin>. Switch scenes and upload videos there, and the physical wall updates in real time.

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

- Each of the 5 Chromium instances drives a 3x3 block of displays. That is 9 physical panels, usually 1920x1080 each, so each node has a 5760x3240 Chromium window.
- Every block has a keyboard and mouse over USB-IP. Plug a USB keyboard or mouse into any node's input server, and the Chromium instance sees it as a normal HID device. You can treat the wall as five big interactive surfaces.
- Only Chromium #5 (`node=4`) outputs audio. The other four nodes have no speakers wired in, so sound-reactive scenes should gate audio on `node === 4`.

## Running it in GitHub Codespaces

The repo includes a Go 1.23 devcontainer (`.devcontainer/devcontainer.json`). To use it:

1. Click **Code > Codespaces > Create codespace on main**.
2. When the container is up, run:

   ```
   go run ./cmd/wallsync
   ```

3. Codespaces forwards port 8080. The devcontainer makes the port public, so you get a URL of the form `https://<codespace>-8080.app.github.dev`. Anyone can reach it, including the Chromium hosts of the physical wall.
4. Open five tabs, one per node, at:

   ```
   https://<codespace>-8080.app.github.dev/wall?node=0
   https://<codespace>-8080.app.github.dev/wall?node=1
   https://<codespace>-8080.app.github.dev/wall?node=2
   https://<codespace>-8080.app.github.dev/wall?node=3
   https://<codespace>-8080.app.github.dev/wall?node=4
   ```

   Also open `/admin` for the control UI. To point the physical wall at the same URLs, set the launch URL of each Chromium host to the matching `?node=N`.

To keep the forwarded port private, change `"visibility": "public"` to `"private"` in `.devcontainer/devcontainer.json`. Then sign the wall's Chromium hosts in to your GitHub account. Codespaces then gives them a session cookie.

## Running locally (single machine, 5 wall tabs)

```
go build ./cmd/wallsync
./wallsync
```

Then open these 6 tabs in Chrome or Chromium, full screen at 100% zoom:

- `http://localhost:8080/wall?node=0`
- `http://localhost:8080/wall?node=1`
- `http://localhost:8080/wall?node=2`
- `http://localhost:8080/wall?node=3`
- `http://localhost:8080/wall?node=4`
- `http://localhost:8080/admin`

Press `d` on any wall tab to show or hide the debug HUD (frame count, T0, current t, DPR check).

Use the buttons on the admin page to switch scenes. Each switch broadcasts a `scene_change` with an epoch ~800 ms in the future, so all tabs change scene at the same instant.

## Modifying it

- Add a new scene. Scenes live under `internal/web/static/js/scenes/<name>/index.js` and register themselves with the shell. Each scene gets `t` (seconds since `T0`), its node index and the wall geometry, then draws its slice. Copy `warpJump` as a starting point.
- Point the wall at a different server. The 5 Chromium hosts aren't tied to this repo. Any WebSocket-speaking web server that serves a `/wall?node=N` page will do. The protocol is in `internal/protocol/`. Use it to speak the same `hello` / `welcome` / `tick` / `scene_change` dialect, or invent your own.
- Make it interactive. USB-IP keyboard and mouse input reaches the Chromium window as it would on any laptop. Register normal DOM `keydown` / `pointermove` listeners inside your scene.
- Use node 4 for audio. `WebAudio` or an `<audio>` element gated on `nodeId === 4` is enough. For tight sync with the visual nodes, schedule the audio start on a `T0`-aligned future epoch, the same way scene switches work.

Three scenes ship with the repo:

- `three-unified`: one Three.js world (wireframe terrain and orbiting flyers), split across the nodes with `camera.setViewOffset`.
- `solar-panorama`: the solar system laid out horizontally across all 15 tiles, with distances to true scale (1 AU = 900 wall units). An auto-cycling spotlight enlarges each planet in turn and shows a callout of its physical stats. The scene also has the asteroid belt, the Kuiper belt, Saturn's rings and a starfield.
- `warp-jump`: radial star streaks from a vanishing point at the wall centre, with periodic shockwave rings.

There is also a video scene. Upload an `.mp4`, `.webm` or `.mov` file from the admin page, and every node plays it back in sync. Each play uses one of two layouts:

- mirrored: the whole video on every node.
- stretched: one video across the full 28800 x 3240 wall, with each node showing its slice.

Videos loop until you click **stop video**.

## Uploading a video

On `/admin`, pick a file (`.mp4`, `.webm`, `.mov`) and click **upload**. Then click **play mirrored** or **play stretched** on its row. **stop video** returns the wall to `three-unified`.

Files go to `./data/videos` by default, each with a `.json` sidecar. To use a different directory, set `WALLSYNC_MEDIA_DIR` (see [DEPLOYMENT.md](./DEPLOYMENT.md)).

Video sync uses the same server-published epoch as the procedural scenes. Each node seeks `<video>.currentTime` to `wallClock() % duration` and corrects drift. For small skew it nudges `playbackRate`. Beyond 250 ms it does a hard seek.

## Deployment

[DEPLOYMENT.md](./DEPLOYMENT.md) covers running against the physical wall, Docker Compose and the server env vars.

## Architecture

- `cmd/wallsync/main.go`: HTTP mux + wiring
- `internal/config`: env-driven config struct (wall geometry, timings)
- `internal/protocol`: JSON envelope + typed messages (hello, welcome, ping, pong, scene_change, tick, admin_status)
- `internal/timing`: clock helpers, PingBook + rolling RTT/offset stats
- `internal/hub`: WS accept, node registry (5 slots), supersede semantics on reconnect, non-blocking broadcast fan-out
- `internal/scene`: Director + Switch (future-epoch + broadcast) + optional auto-rotate
- `internal/feeds`: single 30 Hz ticker, per-scene payload (Three camera, Matter spawner, Pixi FFT/radar)
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

Released into the public domain under the [BSD Zero Clause License](./LICENSE) (0BSD).

Do whatever you want with it: copy, modify, distribute, use commercially, no attribution required, no warranty.

Bundled third-party code keeps its own license:

- three.js (`internal/web/static/js/vendor/three.module.js`): MIT, Copyright 2010-2024 Three.js Authors. Keep the license header intact if you redistribute the file.
- github.com/coder/websocket: ISC. Go modules pull it in at build time. If you want to be thorough, include its license in binary distributions.
