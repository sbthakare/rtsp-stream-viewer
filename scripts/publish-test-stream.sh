#!/usr/bin/env bash
# Publish a looping test pattern (or your own video file) to a local MediaMTX server.
#   ./scripts/publish-test-stream.sh            -> rtsp://localhost:8554/cam1
#   ./scripts/publish-test-stream.sh cam2       -> rtsp://localhost:8554/cam2
#   ./scripts/publish-test-stream.sh cam3 my.mp4
# Start MediaMTX first:  docker run --rm -p 8554:8554 bluenviron/mediamtx:latest
set -euo pipefail
NAME="${1:-cam1}"
FILE="${2:-}"
TARGET="rtsp://localhost:8554/${NAME}"

if [[ -n "$FILE" ]]; then
  exec ffmpeg -re -stream_loop -1 -i "$FILE" -c:v libx264 -preset ultrafast -tune zerolatency -pix_fmt yuv420p -an \
    -f rtsp -rtsp_transport tcp "$TARGET"
fi
exec ffmpeg -re -f lavfi -i "testsrc2=size=1280x720:rate=25" -c:v libx264 -preset ultrafast -tune zerolatency \
  -pix_fmt yuv420p -g 50 -f rtsp -rtsp_transport tcp "$TARGET"
