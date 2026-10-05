# ---- build ----
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
COPY . .
# go.sum is generated here if it was not committed
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- run ----
FROM alpine:3.20
RUN apk add --no-cache ffmpeg ca-certificates && adduser -D -u 10001 app
USER app
COPY --from=build /out/server /usr/local/bin/server
ENV PORT=8080
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:${PORT}/api/health || exit 1
CMD ["server"]
