# syntax=docker/dockerfile:1.7

FROM golang:1.22-alpine AS builder

WORKDIR /src

# Cache module downloads in their own layer.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Static, stripped binary. -trimpath removes local paths from the binary.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20 AS runtime

RUN apk --no-cache add ca-certificates tzdata wget \
 && addgroup -S app && adduser -S -G app -u 10001 app

WORKDIR /app
COPY --from=builder /out/server /app/server

USER app
EXPOSE 8080

# Container-level healthcheck mirrors the Kubernetes liveness probe.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/livez >/dev/null || exit 1

ENTRYPOINT ["/app/server"]
