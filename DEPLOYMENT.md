# Deployment

Running `wallsync` against the physical 5-node videowall, plus the server-side knobs you'd want to touch in production. For getting-started / dev workflow (Codespaces, local tabs, admin UI), see [README.md](./README.md).

## Docker

```
docker compose up --build
```

The container serves on port 8080 with all frontend assets embedded.

Compose binds the uploaded-video directory to a named volume (`wallsync-media`), so uploads persist across container restarts. Inspect it with `docker volume inspect wallsync-media`.

### Releasing a new image

CI (`.github/workflows/build.yml`) only fires on a semver tag matching `v*.*.*`. Pushing to a branch does nothing. To cut a release:

```
git tag v1.2.3
git push origin v1.2.3
```

The workflow runs tests, builds the image, and pushes it to the registry with four tags: `:1.2.3` (immutable), plus the moving tags `:1.2`, `:1`, and `:latest`. Pre-release tags (e.g. `v1.2.3-rc1`) build and push only the exact version and don't move `:latest`.

## Pointing the 5 remote Chromium hosts at the server

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

`--force-device-scale-factor=1` is essential: DPR mismatches across hosts produce visible seam shimmer. The client-side debug HUD flashes red if `window.devicePixelRatio !== 1`.

Ensure all 5 hosts sync to the same campus NTP source. The admin page's per-node "offset" column shows warnings above +/- 25 ms. If you're serving from a Codespace instead of a campus host, the hosts inherit the Codespace URL as their origin; the setup works the same way, just over the public internet.

## Server config (environment variables)

| var                        | default          | meaning                                   |
| -------------------------- | ---------------- | ----------------------------------------- |
| `WALLSYNC_ADDR`            | `:8080`          | HTTP bind address                          |
| `WALLSYNC_TICK_HZ`         | `30`             | broadcast rate                            |
| `WALLSYNC_PING_MS`         | `1000`           | heartbeat interval (ms)                   |
| `WALLSYNC_INITIAL_SCENE`   | `idle`           | scene to boot into                        |
| `WALLSYNC_AUTOROTATE_S`    | `0` (off)        | if >0, rotate scenes every N seconds      |
| `WALLSYNC_MEDIA_DIR`       | `./data/videos`  | on-disk root for uploaded videos          |
| `WALLSYNC_MAX_UPLOAD_MB`   | `500`            | per-upload size cap (megabytes)           |
