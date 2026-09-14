FROM golang:1.27.0-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /my-app ./cmd/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /my-app .
EXPOSE 8080
CMD ["./my-app"]