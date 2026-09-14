FROM golang:1.27.0-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /my-app .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /my-app .
EXPOSE 8080
CMD ["./my-app"]