FROM golang:1.27.0-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /my-app ./cmd/main.go

FROM mwader/static-ffmpeg:6.1 AS ffmpeg

FROM alpine:latest
COPY --from=ffmpeg --chmod=755 /ffprobe /usr/local/bin/
WORKDIR /root/
COPY --from=builder /my-app .
EXPOSE 8080
ENTRYPOINT ["./my-app"]