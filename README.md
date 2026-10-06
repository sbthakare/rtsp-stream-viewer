# RTSP Stream Viewer

Add RTSP stream URLs in a web page and watch them live in the browser, one or many at once.

- **Frontend:** React 18 + TypeScript (Vite)
- **Backend:** Go, FFmpeg, WebSockets
- **Decoder in the browser:** [JSMpeg](https://github.com/phoboslab/jsmpeg) (MPEG-1 video over MPEG-TS), no plugins

```
 RTSP camera ──► FFmpeg (one per URL) ──► Go hub ──► WebSocket ──► JSMpeg canvas (browser)
                 transcode to MPEG-TS       fan-out   binary        one tile per stream
```

Browsers cannot play RTSP, so the Go server runs FFmpeg to convert each stream to MPEG-1 video in an MPEG-TS
container, then pushes the bytes to the page over a WebSocket where JSMpeg decodes them onto a `<canvas>`.

## Features

- Add streams by URL (`rtsp://` or `rtsps://`, credentials supported). The list is remembered in the browser.
- Grid layout with 1 to 4 columns or auto-fit, responsive down to phones.
- Per-stream Play/Pause, Fullscreen and Remove. Pause disconnects fully, so the server stops FFmpeg when nobody is watching.
- Clear status for each tile: connecting, live, reconnecting, paused, error.
- Graceful failures: wrong password, wrong path, refused connection, timeouts and dropped streams each produce a plain-language message. Dropped streams reconnect automatically with exponential backoff (5 tries), then offer "Try again".
- Efficient: ten viewers of the same camera share one FFmpeg process. Slow viewers drop frames instead of slowing everyone down.

## Run it locally

Prerequisites: Node 18+, and either Docker **or** Go 1.22+ with FFmpeg installed.

### 1. Start test cameras and the backend (Docker)

```bash
docker compose up --build
```

This starts MediaMTX (RTSP server on `:8554`), three FFmpeg test publishers (`cam1`, `cam2`, `cam3`) and the Go backend on `:8080`. The Compose backend uses a `VIDEO_BITRATE` of `4000k` for higher-quality local testing.

The test publishers intentionally show generated video rather than real cameras: `cam1` is an animated test pattern, `cam2` is a Mandelbrot pattern, and `cam3` is SMPTE colour bars. Add these local Docker URLs in the viewer: `rtsp://mediamtx:8554/cam1`, `rtsp://mediamtx:8554/cam2`, and `rtsp://mediamtx:8554/cam3`.

### 2. Start the frontend

```bash
cd frontend
cp .env.example .env     # optional: adds one-click test stream chips
npm install
npm run dev              # http://localhost:5173
```

Click a **cam1 / cam2 / cam3** chip, or paste `rtsp://mediamtx:8554/cam1`. When the backend runs in Docker, use the host
name `mediamtx`; when the backend runs on your machine, use `localhost` (see below).

### Without Docker for the backend

```bash
# terminal 1: RTSP server
docker run --rm -p 8554:8554 bluenviron/mediamtx:latest
# terminal 2: test camera (needs ffmpeg)
./scripts/publish-test-stream.sh cam1
# terminal 3: backend
cd backend && go mod tidy && go run ./cmd/server
```

Stream URL for this setup: `rtsp://localhost:8554/cam1`. To loop your own file: `./scripts/publish-test-stream.sh cam2 clip.mp4`.

### Tests

```bash
cd backend && go test ./...      # URL validation + error mapping
cd frontend && npm run build     # type-checks and builds
```

## Configuration

Backend environment variables (see `backend/.env.example`):

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP port (hosting platforms set this) |
| `ALLOWED_ORIGINS` | `*` | Comma-separated frontend origins allowed to call the API/WebSocket. **Set this in production.** |
| `MAX_STREAMS` | `8` | Max simultaneous FFmpeg processes |
| `VIDEO_BITRATE` / `MAX_WIDTH` / `FPS` | `1500k` / `960` / `25` | Output quality vs CPU/bandwidth. `FPS` must be 24, 25, 30, 50, or 60 because the browser pipeline uses MPEG-1; an unsupported value falls back to 25. `docker-compose.yml` uses `4000k` and 25 fps for local testing. |
| `ALLOWED_RTSP_HOSTS` | empty | Optional allow-list of RTSP hosts |
| `BLOCK_PRIVATE_HOSTS` | `false` | Refuse loopback/private addresses (turn on for public deployments) |
| `IDLE_GRACE_SECONDS` | `5` | Keep FFmpeg alive briefly after the last viewer leaves |
| `START_TIMEOUT_SECONDS` | `20` | Give up if no video arrives |

Frontend (`frontend/.env`): `VITE_API_URL` (backend base URL, empty in dev) and `VITE_DEMO_STREAMS` (optional test chips).

## Deploy

The backend needs FFmpeg, so deploy it from `backend/Dockerfile` (Alpine + FFmpeg). The frontend is a static site.

**Backend (Render or Railway)**
1. Create a web service from this repo with root directory `backend` (Docker build is detected automatically).
2. Set `ALLOWED_ORIGINS` to your frontend URL and `BLOCK_PRIVATE_HOSTS=true`.
3. Check `https://<backend>/api/health` returns `{"status":"ok","ffmpeg":true}`.

`render.yaml` in the repo root sets up both services on Render as a Blueprint.

**Frontend (Vercel)**
1. Import the repo, set **Root Directory** to `frontend`.
2. Add the environment variable `VITE_API_URL=https://<backend-url>`.
3. Deploy.

**A camera the cloud backend can reach.** The deployed backend connects to RTSP URLs from the internet, so `localhost` and
Docker hostnames will not work there. For the live demo, run MediaMTX plus the test publishers on a small VPS (or a Railway
service with a TCP proxy for port 8554, or any public RTSP stream) and add `rtsp://<public-host>:8554/cam1`.

When using a VPS as the RTSP host, deploy the `mediamtx`, `cam1`, `cam2`, and `cam3` services from `docker-compose.yml` on that VPS. The Render backend does not create those streams; it only reads them. Verify that every requested path has an active publisher:

```bash
docker compose up -d mediamtx cam1 cam2 cam3
docker compose ps
docker compose logs cam3
```

If `rtsp://<public-host>:8554/cam3` returns `404 Not Found`, no publisher is sending video to `/cam3`; start or fix the `cam3` service on the RTSP host. Replace the test publishers with your own camera publishers to show real footage.

## API

| Endpoint | Description |
|---|---|
| `GET /ws?url=<rtsp-url>` | Upgrades to WebSocket and streams binary MPEG-TS. Failures arrive as close code `4001` (do not retry) or `4002` (stream dropped, retry) with a readable reason. |
| `POST /api/validate` `{ "url": "..." }` | Returns `{ "ok": true }` or `{ "ok": false, "error": "..." }` |
| `GET /api/health` | Liveness plus FFmpeg availability |
| `GET /api/stats` | Active streams and viewers |

## Project layout

```
backend/
  cmd/server/main.go            entry point, graceful shutdown
  internal/config               environment configuration
  internal/stream               FFmpeg process manager, fan-out hub, URL validation
  internal/api                  HTTP routes, WebSocket handler, CORS
  Dockerfile                    Alpine + FFmpeg image
frontend/src
  components/                   AddStreamForm, StreamGrid, StreamTile
  hooks/                        usePlayer (connect, retry, pause), useStreams (state + persistence)
  lib/                          config, RTSP parsing, API client, JSMpeg WebSocket source
docker-compose.yml              MediaMTX + test cameras + backend
```

## Design decisions and trade-offs

- **JSMpeg over WebSocket** keeps latency low (about a second) and works in every browser without plugins. MPEG-1 is less
  efficient than H.264, so quality per bit is lower. The alternative is HLS/WebRTC, which needs more infrastructure or adds latency.
- **One FFmpeg per URL, many viewers.** Cost scales with distinct cameras, not viewers. `MAX_STREAMS` bounds CPU. Horizontal
  scaling: run several backend instances behind a load balancer with sticky routing by URL.
- **Backpressure:** each viewer has a bounded queue; when it fills, chunks are dropped for that viewer only.
- **Security:** only `rtsp(s)://` URLs are accepted and passed to FFmpeg as a single argument (no shell, no option injection).
  Optional host allow-list and private-address blocking guard against server-side request forgery. Credentials are redacted in logs
  and masked in the UI.
- **Audio is not forwarded** (video only) to keep the pipeline simple and light.

## Troubleshooting

- *"Cannot reach the stream server"*: the backend is not running, or `VITE_API_URL` is wrong.
- *"The host could not be found" / "connection refused"*: the **backend** cannot reach that address. From Docker use `mediamtx`, not `localhost`.
- *Stays on "Connecting"*: check the backend logs; the camera may be slow to answer (timeout is 20 s by default).
- *Choppy with many streams*: lower `MAX_WIDTH`, `FPS` or `VIDEO_BITRATE`.
