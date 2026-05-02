# AGENTS.md

## Build & Run

```bash
make run                # dev (console logs, debug level)
make build              # builds ./bin/ms-storage
make test               # go test -race ./...
make cover              # coverage summary
make lint               # golangci-lint
make docker-up          # docker compose up -d --build
```

## Module path

`github.com/KryptoStorage/ms-storage` — change in `go.mod` and rerun
`make tidy` if you fork.

## Architecture (Clean Architecture)

```
cmd/server/                 — entry point, wires everything
internal/
  domain/                   — pure business types and rules (no external deps)
    health/                 — Status, HealthCheck, Readiness, Component
    errors/                 — typed Error with Kind for HTTP mapping
  application/              — use cases, DTOs, ports
    dto/                    — output payloads (no domain leak: plain types only)
    ports/                  — interfaces consumed by use cases (e.g. ReadinessProbe)
    health/                 — use case implementation
  infrastructure/           — adapters
    handlers/               — HTTP handlers
    middleware/             — RequestLogger, Recover, SecurityHeaders, CORS, RateLimiter, Metrics, Timeout
    router/                 — gorilla/mux registration
  config/                   — env loading + validation
pkg/                        — sharable helpers
  logging/                  — zerolog setup, request_id ctx helpers
  response/                 — JSON / Error writers; domain.Error → HTTP status
  shutdown/                 — Closer registry (LIFO, error aggregation)
  validator/                — fluent validator (Required, MinLength, …)
```

Direction of dependencies: `infrastructure → application → domain`. The
domain package never imports outwards.

## Operational endpoints (root, unversioned)

| Path | Purpose |
|------|---------|
| `GET /livez`   | Liveness probe — process up. Cheap, no deps. |
| `GET /readyz`  | Readiness probe — runs every registered `ReadinessProbe`; returns 503 when any reports degraded/unhealthy. |
| `GET /metrics` | Prometheus exposition. |

Business endpoints hang off `/api/v1/...`. Add new routes in
`internal/infrastructure/router/router.go` inside the `v1` subrouter.

## Adding a feature

1. `internal/domain/<feature>/`        — entity + invariants + domain errors
2. `internal/application/dto/`         — input/output DTOs (plain types)
3. `internal/application/ports/`       — repository / external interfaces
4. `internal/application/<feature>/`   — use case
5. `internal/infrastructure/handlers/` — HTTP handler
6. `internal/infrastructure/router/`   — register on `v1` subrouter
7. `cmd/server/main.go`                — wire dependencies; register shutdown hooks
8. Tests next to each layer (`*_test.go`)

## Adding a readiness probe

Implement `ports.ReadinessProbe` (Name, Probe(ctx)). Append to the `probes`
slice in `cmd/server/main.go` before constructing `health.New(...)`.

## Errors

Return `*errors.Error` from use cases with the appropriate `Kind`. Handlers
call `response.Error(w, err)` which maps Kind → HTTP status. For ad-hoc
responses use `response.ErrorWith(w, status, code, msg, details)`.

## Env vars

See `.env.example`. Highlights:

| Var | Default | Notes |
|-----|---------|-------|
| `APP_ENV` | `development` | one of development/staging/production/test |
| `HTTP_PORT` | `8080` | validated as TCP port |
| `HTTP_HANDLER_TIMEOUT` | `15s` | wraps router with `http.TimeoutHandler` |
| `HTTP_SHUTDOWN_TIMEOUT` | `30s` | drain budget for the closer registry |
| `LOG_LEVEL` / `LOG_FORMAT` | `info` / `json` | console for dev |
| `CORS_ALLOWED_ORIGINS` | empty | leave empty to disable; comma-separated |
| `RATE_LIMIT_ENABLED` | `false` | per-IP token bucket; in-process only |

`config.Load()` returns an error on invalid values — fail fast at startup.

## Shutdown

Resources register a Hook with `shutdown.NewRegistry().Register(name, fn)`.
On SIGINT/SIGTERM the server drains in two phases sharing
`HTTP_SHUTDOWN_TIMEOUT`:

1. `srv.Shutdown(ctx)` — stops accepting new connections and waits for
   in-flight requests to finish.
2. `registry.Run(ctx)` — runs hooks in LIFO order to close downstream
   pools (DB, queues, rate limiter, …).

Register pools as you open them in `run()`; they will close after the HTTP
server has drained, so in-flight requests still see a live pool.
