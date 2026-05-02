# ms-storage

Microservicio Go con **Clean Architecture**, listo para producción: logging
estructurado, probes de Kubernetes, métricas Prometheus, configuración
validada, shutdown ordenado, errores tipados y tests.

## Stack

- **Go 1.22+**
- [`gorilla/mux`](https://github.com/gorilla/mux) — router HTTP
- [`rs/zerolog`](https://github.com/rs/zerolog) — logging estructurado zero-alloc
- [`prometheus/client_golang`](https://github.com/prometheus/client_golang) — métricas
- [`golang.org/x/time/rate`](https://pkg.go.dev/golang.org/x/time/rate) — rate limiting
- [`joho/godotenv`](https://github.com/joho/godotenv) — `.env` para desarrollo

---

## Estructura del proyecto

```
ms-storage/
├── cmd/
│   └── server/main.go                     # Entry point: bootstrap, shutdown
│
├── internal/
│   ├── config/
│   │   ├── config.go                      # Carga + validación de env vars
│   │   └── config_test.go
│   │
│   ├── domain/                            # CORE — sin dependencias externas
│   │   ├── health/
│   │   │   └── entity.go                  # HealthCheck, Readiness, Component
│   │   ├── health/status.go               # StatusHealthy, Degraded, Unhealthy
│   │   └── errors/errors.go               # *Error con Kind tipado
│   │
│   ├── application/                       # CASOS DE USO
│   │   ├── dto/health.go                  # DTOs de salida (tipos planos)
│   │   ├── ports/health.go                # HealthPort, ReadinessProbe
│   │   └── health/
│   │       ├── usecase.go                 # GetHealth, GetReadiness
│   │       └── usecase_test.go
│   │
│   └── infrastructure/                    # ADAPTADORES
│       ├── handlers/health.go             # Liveness, Readiness
│       ├── handlers/health_test.go
│       ├── middleware/middleware.go       # Logger, CORS, RateLimit, Security
│       └── router/router.go               # gorilla/mux + Prometheus
│
├── pkg/                                   # Helpers reutilizables
│   ├── logging/logger.go                  # Wrapper Zerolog + ctx helpers
│   ├── response/response.go               # JSON + Error → HTTP status mapping
│   ├── shutdown/shutdown.go               # Closer registry (LIFO + join errors)
│   ├── shutdown/shutdown_test.go
│   ├── validator/validator.go             # Validaciones fluidas
│   └── validator/validator_test.go
│
├── .env.example
├── .golangci.yml
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

---

## Arquitectura

Clean Architecture con regla de dependencia estricta: las flechas siempre
apuntan **hacia adentro**. El dominio no sabe que existen HTTP, JSON ni
bases de datos.

```
┌──────────────────────────────────────────────────────────────────┐
│                       INFRASTRUCTURE                             │
│   handlers · middleware · router  ◄── adaptadores HTTP           │
│            │                                                     │
│            ▼ depende de                                          │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                    APPLICATION                          │   │
│   │   use cases · ports · dto                               │   │
│   │            │                                            │   │
│   │            ▼ depende de                                 │   │
│   │   ┌────────────────────────────────────────────────┐   │   │
│   │   │                  DOMAIN                        │   │   │
│   │   │   entities · value objects · domain errors    │   │   │
│   │   │   (sin dependencias externas)                  │   │   │
│   │   └────────────────────────────────────────────────┘   │   │
│   └─────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘

           cmd/server: punto de entrada que ensambla todo
              (primary adapter, conoce todas las capas)
```

**Por qué importa:**
- El dominio se puede testear sin levantar HTTP ni DB.
- Cambiar la base de datos, el framework HTTP o el formato de log no
  toca la lógica de negocio.
- Los `ports` (interfaces) definen lo que la aplicación necesita; la
  infraestructura provee implementaciones.

---

## Endpoints

El servicio expone **dos APIs sobre el mismo puerto**, con audiencias
distintas:

### API operacional (root, sin versionar)

Consumidores: Kubernetes, Prometheus, load balancer. Estos paths son
**estables de por vida del servicio** — la plataforma no debe conocer
versiones de tu API de negocio.

| Método | Path | Propósito |
|--------|------|-----------|
| GET | `/livez` | Liveness — el proceso está vivo. Cheap, sin dependencias. |
| GET | `/readyz` | Readiness — corre cada `ReadinessProbe` registrada; devuelve **503** si algo no está sano para drenar tráfico. |
| GET | `/metrics` | Exposición Prometheus. |

Ejemplo:

```bash
curl -s http://localhost:8080/livez
# {"status":"healthy","version":"1.0.0","uptime":"5m30s","service_name":"ms-storage","timestamp":"2026-05-02T07:00:00Z"}

curl -s http://localhost:8080/readyz
# {"status":"healthy","components":[],"timestamp":"2026-05-02T07:00:00Z"}
```

### API de negocio (versionada)

Bajo `/api/v1/...`. Hoy está vacía: agrega rutas en
`internal/infrastructure/router/router.go` dentro del subrouter `v1`.

**Por qué la separación:** los endpoints operacionales no son contrato
hacia clientes; son contrato hacia tu plataforma. Versionarlos los ata
artificialmente al ciclo de vida de tu API de negocio. La convención
estándar en Kubernetes y Prometheus es root.

---

## Variables de entorno

Todas tienen valor por defecto y se validan al arrancar. Si algo es
inválido, el proceso falla con un error explícito (fail-fast).

### Aplicación

| Variable | Default | Notas |
|----------|---------|-------|
| `APP_NAME` | `ms-storage` | |
| `APP_VERSION` | `1.0.0` | |
| `APP_ENV` | `development` | `development \| staging \| production \| test` |

### HTTP

| Variable | Default | Notas |
|----------|---------|-------|
| `HTTP_HOST` | `0.0.0.0` | |
| `HTTP_PORT` | `8080` | validado como puerto TCP |
| `HTTP_READ_TIMEOUT` | `10s` | duración Go (`5s`, `1m`, …) |
| `HTTP_WRITE_TIMEOUT` | `10s` | |
| `HTTP_IDLE_TIMEOUT` | `60s` | |
| `HTTP_HANDLER_TIMEOUT` | `15s` | envuelve el router con `http.TimeoutHandler` |
| `HTTP_SHUTDOWN_TIMEOUT` | `30s` | budget para drenar conexiones y cerrar pools |

### Logging

| Variable | Default | Notas |
|----------|---------|-------|
| `LOG_LEVEL` | `info` | `debug \| info \| warn \| error` |
| `LOG_FORMAT` | `json` | `json` (prod) o `console` (dev) |

### CORS

Vacío por defecto (CORS deshabilitado). Habilítalo solo cuando lo
necesites; valores como `*` están permitidos pero **no para endpoints
autenticados**.

| Variable | Default | Notas |
|----------|---------|-------|
| `CORS_ALLOWED_ORIGINS` | _(vacío)_ | CSV: `https://app.example.com,https://admin.example.com` |
| `CORS_ALLOWED_METHODS` | `GET,POST,PUT,PATCH,DELETE,OPTIONS` | |
| `CORS_ALLOWED_HEADERS` | `Content-Type,Authorization,X-Request-ID` | |
| `CORS_MAX_AGE_SECONDS` | `600` | |

### Rate limiting (in-process, por IP)

| Variable | Default | Notas |
|----------|---------|-------|
| `RATE_LIMIT_ENABLED` | `false` | |
| `RATE_LIMIT_RPS` | `50` | |
| `RATE_LIMIT_BURST` | `100` | |

> Para una flota multi-réplica usa un limiter centralizado (Redis,
> Envoy, API gateway). El de aquí cubre un solo nodo o tráfico sticky.

---

## Primeros pasos

### Requisitos

- Go 1.22+
- Docker (opcional)
- `make` (opcional, todo lo de abajo se puede correr a mano)

### Local

```bash
cp .env.example .env
make run                          # console logs, debug level
# o sin make:
LOG_FORMAT=console LOG_LEVEL=debug go run ./cmd/server
```

### Build

```bash
make build                        # produce ./bin/ms-storage
./bin/ms-storage
```

### Docker

```bash
make docker-up                    # docker compose up -d --build
make docker-down
```

El contenedor corre como usuario no privilegiado (uid 10001), con
`read_only: true`, `cap_drop: ALL` y `no-new-privileges`.

---

## Comandos del Makefile

```bash
make help          # lista todos los targets
make run           # arranca con logs en consola
make build         # compila a ./bin/ms-storage
make test          # go test -race ./...
make cover         # cobertura + resumen
make cover-html    # abre el HTML
make vet           # go vet
make lint          # golangci-lint
make fmt           # gofmt + goimports
make docker-up     # docker compose up -d --build
make docker-down
make tidy          # go mod tidy
make clean
```

---

## Logging

Zerolog estructurado. Cada request recibe un `request_id` (uuid o el
header `X-Request-ID` entrante) que se propaga al contexto y se devuelve
en la respuesta.

### Console (desarrollo)

```bash
LOG_FORMAT=console LOG_LEVEL=debug go run ./cmd/server
```

```
2026-05-02T07:00:00Z INF Starting ms-storage env=development version=1.0.0
2026-05-02T07:00:01Z INF Incoming request method=GET path=/livez request_id=8a61...
2026-05-02T07:00:01Z INF Request completed status=200 duration=0.29ms request_id=8a61...
```

### JSON (producción)

```json
{"level":"info","service":"ms-storage","event":"request_completed","request_id":"8a61...","method":"GET","path":"/livez","status":200,"duration":290857,"time":"2026-05-02T07:00:01Z"}
```

Listo para Loki, Datadog, CloudWatch o cualquier ingestor de JSON.

### Recuperar el request_id en código

```go
import "github.com/KryptoStorage/ms-storage/pkg/logging"

func (h *MyHandler) Do(w http.ResponseWriter, r *http.Request) {
    rid := logging.GetRequestID(r.Context())
    h.logger.Info().Str("request_id", rid).Msg("doing something")
}
```

---

## Manejo de errores

Los use cases retornan `*errors.Error` con un `Kind` tipado. El handler
no decide el HTTP status — eso lo hace `pkg/response.Error`, que mapea
`Kind` → status code. Esto centraliza la traducción y evita que cada
handler reinvente la rueda.

```go
import (
    derrors "github.com/KryptoStorage/ms-storage/internal/domain/errors"
    "github.com/KryptoStorage/ms-storage/pkg/response"
)

// En el use case:
return derrors.New(derrors.KindNotFound, "user_not_found", "user does not exist")

// En el handler:
if err != nil {
    response.Error(w, err)         // 404 + JSON normalizado
    return
}
```

Mapeo `Kind` → HTTP:

| Kind | Status |
|------|--------|
| `KindValidation` | 400 |
| `KindUnauthorized` | 401 |
| `KindForbidden` | 403 |
| `KindNotFound` | 404 |
| `KindConflict` | 409 |
| `KindUnavailable` | 503 |
| `KindInternal` y desconocido | 500 |

Formato de respuesta:

```json
{ "error": { "code": "user_not_found", "message": "user does not exist" } }
```

---

## Graceful shutdown

Al recibir `SIGINT` o `SIGTERM`:

1. El servidor HTTP deja de aceptar conexiones nuevas.
2. Se ejecutan los hooks del `shutdown.Registry` en **orden LIFO** (lo
   último que se abrió se cierra primero).
3. Los errores se agregan con `errors.Join` — un hook que falla no
   impide que los demás corran.
4. Si se excede `HTTP_SHUTDOWN_TIMEOUT`, el contexto se cancela.

Registrar un recurso:

```go
registry := shutdown.NewRegistry()
registry.Register("http_server", srv.Shutdown)
registry.Register("postgres", pool.Close)        // se cierra primero
registry.Register("kafka", producer.Close)       // se cierra antes que postgres
```

---

## Observabilidad

`/metrics` expone las métricas estándar de la runtime de Go (GC,
goroutines, memoria, etc.) más cualquier métrica que registres en
`prometheus.DefaultRegisterer`.

Ejemplo de métrica de negocio:

```go
var requestsTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "ms_storage_requests_total",
        Help: "Total HTTP requests por endpoint y status.",
    },
    []string{"endpoint", "status"},
)
```

Tracing (OpenTelemetry) **no** está cableado, pero el `context.Context`
ya se propaga end-to-end, así que agregarlo es aditivo.

---

## Agregar un feature nuevo

Sigue las capas de Clean Architecture (de adentro hacia afuera):

### 1. Dominio — `internal/domain/<feature>/`

Tipos puros y reglas de invariantes. Sin imports de `net/http`, ni de
ninguna librería externa.

```go
package users

type User struct {
    ID    string
    Name  string
    Email string
}
```

### 2. DTO de salida — `internal/application/dto/`

Tipos planos que se serializan a JSON. **No importan dominio.**

```go
type UserOutput struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

### 3. Puertos — `internal/application/ports/`

Interfaces que el use case necesita (repositorio, cache, etc.).

```go
type UserRepository interface {
    FindByID(ctx context.Context, id string) (*users.User, error)
}
```

### 4. Use case — `internal/application/<feature>/`

```go
type UseCase struct {
    repo ports.UserRepository
}

func (u *UseCase) GetUser(ctx context.Context, id string) (dto.UserOutput, error) {
    user, err := u.repo.FindByID(ctx, id)
    if err != nil {
        return dto.UserOutput{}, derrors.Wrap(derrors.KindNotFound, "user_not_found", "not found", err)
    }
    return dto.UserOutput{ID: user.ID, Name: user.Name, Email: user.Email}, nil
}
```

### 5. Handler — `internal/infrastructure/handlers/`

```go
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
    id := mux.Vars(r)["id"]
    out, err := h.usecase.GetUser(r.Context(), id)
    if err != nil {
        response.Error(w, err)
        return
    }
    response.JSON(w, http.StatusOK, out)
}
```

### 6. Ruta — `internal/infrastructure/router/router.go`

Dentro del subrouter `v1`:

```go
v1.HandleFunc("/users/{id}", d.User.Get).Methods(http.MethodGet)
```

### 7. Wiring — `cmd/server/main.go`

```go
userRepo := postgres.NewUserRepository(db)
userUC   := users.New(userRepo)
userHandler := handlers.NewUserHandler(userUC, log)

r := router.New(router.Deps{
    Health: healthHandler,
    User:   userHandler,
})
```

### 8. Tests

Coloca `*_test.go` junto al código de cada capa. Los use cases se
testean con fakes que implementan el puerto; los handlers con
`httptest`.

---

## Agregar una readiness probe

Implementa `ports.ReadinessProbe` y agrégala al slice `probes` antes de
construir el use case de health:

```go
type pgProbe struct{ pool *pgxpool.Pool }

func (p *pgProbe) Name() string { return "postgres" }
func (p *pgProbe) Probe(ctx context.Context) dto.ComponentOutput {
    start := time.Now()
    if err := p.pool.Ping(ctx); err != nil {
        return dto.ComponentOutput{
            Name: "postgres", Status: "unhealthy", Message: err.Error(),
            Latency: time.Since(start).String(),
        }
    }
    return dto.ComponentOutput{
        Name: "postgres", Status: "healthy",
        Latency: time.Since(start).String(),
    }
}
```

En `main.go`:

```go
probes := []ports.ReadinessProbe{
    &pgProbe{pool: pool},
}
healthUC := health.New(health.Options{Version: ..., Probes: probes})
```

`/readyz` devolverá:

- `200` si todas las probes son `healthy`.
- `503` si alguna está `degraded` o `unhealthy`.

---

## Testing

```bash
make test                # go test -race -count=1 ./...
make cover               # con coverage
make cover-html          # abre el HTML
```

Convenciones:

- Tests junto al código (`*_test.go` en el mismo paquete).
- Race detector siempre activo (`-race`).
- Use cases con fakes que implementan los puertos; nada de mocks
  pesados.
- Handlers con `net/http/httptest`.

---

## Linting

```bash
make lint                # golangci-lint
```

Configurado en `.golangci.yml` con: `errcheck`, `govet`, `staticcheck`,
`gosec`, `revive`, `misspell`, `bodyclose`, `gocyclo`, `goimports`,
`unconvert`, `unparam`, `ineffassign`, `gosimple`, `unused`.

---

## Principios aplicados

1. **Dependency Rule** — dependencias siempre hacia el core.
2. **Explicit boundaries** — `ports` definen lo que la aplicación
   necesita; la infraestructura provee.
3. **Fail fast** — config inválida aborta el arranque.
4. **Errors carry meaning** — `Kind` tipado, mapeo único a HTTP.
5. **Operational vs business APIs** — endpoints de plataforma en root,
   endpoints de negocio versionados.
6. **Liveness ≠ Readiness** — probes con responsabilidades distintas.
7. **Graceful shutdown ordenado** — closer registry, no `os.Exit`
   directo.
8. **Zero-alloc logging** — Zerolog, JSON estructurado, request_id
   propagado.
9. **Tests por capa, sin mocks pesados** — interfaces pequeñas y fakes.

---

## Recursos

- [Clean Architecture — Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Kubernetes — Configure Liveness, Readiness and Startup Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Prometheus — Instrumenting a Go application](https://prometheus.io/docs/guides/go-application/)
- [zerolog](https://github.com/rs/zerolog) · [gorilla/mux](https://github.com/gorilla/mux)
