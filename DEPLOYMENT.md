# Deployment

Running `wallsync` against the physical 5-node videowall, plus the server-side knobs you'd want to touch in production. For getting-started / dev workflow (Codespaces, local tabs, admin UI), see [README.md](./README.md).

## Docker

```
docker compose up --build
```

The container serves on port 8080 with all frontend assets embedded.

Compose binds the uploaded-video directory to a named volume (`wallsync-media`), so uploads persist across container restarts. Inspect it with `docker volume inspect wallsync-media`.

### Releasing a new image

CI (`.github/workflows/build.yml`) runs `go vet` and `go test -race` on every pull request and on every `v*` tag. Each PR also builds the image. When the registry secrets are available, the image is pushed as `<last release>-pr<N>-<short hash>` (e.g. `1.1.2-pr12-9b141f4`). PR images never move `:latest` or `:1`.

To cut a release, tag `main`:

```
git tag v1.2.3
git push origin v1.2.3
```

The workflow runs the tests, builds the image, and pushes it with the tags `:1.2.3` (immutable), `:1.2`, `:1`, `:latest` and `:sha-<hash>`. Pre-release tags (e.g. `v1.2.3-rc1`) push only the exact version and don't move `:1.2`, `:1` or `:latest`.

Dependabot (`.github/dependabot.yml`) opens update PRs for Go modules and the Dockerfile base images every Monday, and for GitHub Actions monthly:

- **GitHub Actions updates** merge automatically once CI passes. They don't change the image, so no release is made.
- **Go module and Docker updates** wait for a person to review and merge them. On merge, `.github/workflows/dependabot-release.yml` tags the next patch release (e.g. `v1.1.2` → `v1.1.3`) and starts the build on that tag.

PRs from people never tag a release automatically.

The repo (or org) needs the secrets `DOCKER_REGISTRY`, `DOCKER_USERNAME` and `DOCKER_PASSWORD`, set for both Actions and Dependabot. Without the Dependabot copy, Dependabot PRs still build and test but don't push a preview image.

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
