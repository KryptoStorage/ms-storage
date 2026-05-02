# MS-Storage Microservice

Microservicio Golang escalable basado en **Clean Architecture** con **Gorilla Toolkit**, **Zerolog** y **Docker**.

## Tecnologías

- **Golang 1.22+**
- **Gorilla Mux** - Router HTTP
- **Zerolog** - Logging estructurado
- **Docker** - Contenedor

---

## Arquitectura

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         CLEAN ARCHITECTURE                             │
│                                                                         │
│   ┌────────────────────────────────────────────────────────────────┐  │
│   │                      PRIMARY ADAPTERS                           │  │
│   │                         (cmd/server)                            │  │
│   │                    Entry point, bootstrap                       │  │
│   └────────────────────────────────────────────────────────────────┘  │
│                                 │                                     │
│                                 ▼                                     │
│   ┌────────────────────────────────────────────────────────────────┐  │
│   │                      APPLICATION LAYER                         │  │
│   │              (application/health, application/dto)             │  │
│   │                 Use cases, ports, DTOs                         │  │
│   └────────────────────────────────────────────────────────────────┘  │
│                                 │                                     │
│                                 ▼                                     │
│   ┌────────────────────────────────────────────────────────────────┐  │
│   │                         DOMAIN LAYER                            │  │
│   │                  (domain/health, domain/shared)                 │  │
│   │            Entities, business rules, interfaces                 │  │
│   └────────────────────────────────────────────────────────────────┘  │
│                                 │                                     │
│                                 ▼                                     │
│   ┌────────────────────────────────────────────────────────────────┐  │
│   │                    INFRASTRUCTURE LAYER                        │  │
│   │     (infrastructure/router, middleware, handlers)              │  │
│   │              HTTP adapters, external services                   │  │
│   └────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

### Flujo de una Solicitud

```
HTTP Request
      │
      ▼
cmd/server/main.go           # Punto de entrada
      │
      ▼
infrastructure/middleware    # RequestLogger (logs + request_id)
      │
      ▼
infrastructure/router        # Gorilla Mux (ruteo)
      │
      ▼
infrastructure/handlers      # HealthHandler (HTTP adapter)
      │
      ▼
application/health/usecase   # Lógica de negocio
      │
      ▼
application/dto              # Data Transfer Objects
      │
      ▼
HTTP Response
```

---

## Estructura del Proyecto

```
ms-storage/
├── cmd/
│   └── server/
│       └── main.go                    # Entry point + Graceful Shutdown
│
├── internal/
│   ├── config/
│   │   └── config.go                  # Configuración con env vars
│   │
│   ├── domain/                        # CORE - Lógica pura
│   │   └── health/
│   │       ├── status.go              # Constantes (StatusHealthy, etc)
│   │       ├── entity.go              # Entidad HealthCheck
│   │       └── sync.go                # Entidad SyncStatus
│   │
│   ├── application/                   # CASOS DE USO
│   │   ├── health/
│   │   │   └── usecase.go            # GetHealth(), GetSync()
│   │   ├── dto/
│   │   │   └── health.go             # HealthOutput, SyncOutput
│   │   └── ports/
│   │       └── health.go            # HealthPort interface
│   │
│   └── infrastructure/               # ADAPTADORES
│       ├── router/
│       │   └── router.go             # Gorilla Mux setup
│       ├── middleware/
│       │   └── middleware.go         # RequestLogger, CORS, Security
│       └── handlers/
│           └── health.go            # HTTP Handlers
│
├── pkg/
│   ├── response/
│   │   └── response.go              # Helpers HTTP
│   ├── validator/
│   │   └── validator.go             # Validaciones reutilizables
│   └── logging/
│       └── logger.go                # Wrapper Zerolog
│
├── configs/
│   └── docker/
│       └── .env                      # Variables Docker
│
├── .env                              # Variables locales
├── .gitignore                         # Archivos ignorados por Git
├── .dockerignore                      # Archivos ignorados por Docker
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── README.md
```

---

## Variables de Entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| `APP_NAME` | `ms-storage` | Nombre del microservicio |
| `APP_VERSION` | `1.0.0` | Versión actual |
| `APP_ENV` | `development` | Entorno (development/production) |
| `HTTP_HOST` | `0.0.0.0` | Host donde escucha el servidor |
| `HTTP_PORT` | `8080` | Puerto del servidor |
| `LOG_LEVEL` | `info` | Nivel de log (debug/info/warn/error) |
| `LOG_FORMAT` | `console` | Formato (json para prod, console para dev) |

---

## Primeros Pasos

### Requisitos

- Go 1.22+
- Docker (opcional)

### Instalación

```bash
# Descargar dependencias
go mod tidy

# Compilar
go build -o server ./cmd/server

# Ejecutar (desarrollo con logs legibles)
LOG_FORMAT=console LOG_LEVEL=debug ./server

# Ejecutar (producción con JSON)
LOG_FORMAT=json LOG_LEVEL=info ./server
```

### Con Docker

```bash
# Build y ejecutar
docker-compose up --build

# Ver logs
docker-compose logs -f

# Detener
docker-compose down
```

---

## Logging

El microservicio usa **Zerolog** para logging estructurado JSON.

### Formato Console (Desarrollo)

```bash
LOG_FORMAT=console ./server
```

Salida legible:
```
INF  Starting ms-storage  env=development event=startup service=ms-storage version=1.0.0
DBG  Health check endpoint called  endpoint=/health event=health_check_requested request_id=
INF  Health check response sent  endpoint=/health event=health_check_completed status=healthy
```

### Formato JSON (Producción)

```bash
LOG_FORMAT=json ./server
```

Salida para ELK/Datadog/CloudWatch:
```json
{"level":"info","service":"ms-storage","event":"health_check_completed","request_id":"abc-123","endpoint":"/health","status":"healthy","time":"2026-05-02T06:00:00Z"}
```

### Niveles de Log

| Nivel | Uso |
|-------|-----|
| `debug` | Desarrollo, información detallada |
| `info` | Producción por defecto |
| `warn` | Solo advertencias y errores |
| `error` | Solo errores |

---

## Endpoints

### GET /health

Estado de vida del microservicio.

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "5m30s",
  "service_name": "ms-storage"
}
```

### GET /health/sync

Estado de sincronización.

```bash
curl http://localhost:8080/health/sync
```

**Response:**
```json
{
  "status": "healthy",
  "last_sync": "2026-05-02T06:00:00Z",
  "next_sync": "2026-05-02T06:00:30Z",
  "state": "synced",
  "description": "Service is synchronized and operational"
}
```

---

## Graceful Shutdown

El servidor maneja优雅ly el cierre:

1. Recibe señal `SIGINT` o `SIGTERM`
2. Deja de aceptar nuevas conexiones
3. Espera a que conexiones activas terminen (timeout 30s)
4. Cierra conexión a base de datos si existe
5. Termina proceso

```bash
# Ctrl+C o
kill -SIGINT $PID
kill -SIGTERM $PID
```

---

## Capas y Responsabilidades

### 1. `cmd/server` - Primary Adapter

Punto de entrada. Ensambla dependencias, configura servidor y maneja Graceful Shutdown.

### 2. `internal/domain` - Domain Layer

Lógica de negocio pura. **Sin dependencias externas.**

- `status.go`: Constantes de status
- `entity.go`: Entidades (HealthCheck)
- `sync.go`: Entidades (SyncStatus)

### 3. `internal/application` - Application Layer

Casos de uso y DTOs.

- `usecase.go`: Lógica de negocio
- `dto/`: Data Transfer Objects
- `ports/`: Interfaces (HealthPort)

### 4. `internal/infrastructure` - Infrastructure Layer

Implementaciones concretas.

- `handlers/`: HTTP Handlers
- `router/`: Gorilla Mux
- `middleware/`: RequestLogger, CORS, Security

### 5. `pkg` - Paquetes Compartidos

Código reutilizable.

- `logging/`: Wrapper Zerolog
- `response/`: Helpers HTTP
- `validator/`: Validaciones

---

## Agregar un Nuevo Endpoint

### Paso 1: Crear dominio (`internal/domain/users/`)

```go
// internal/domain/users/user.go
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

### Paso 2: Crear DTOs (`internal/application/dto/`)

```go
// internal/application/dto/user.go
type UserOutput struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

### Paso 3: Crear Puerto (`internal/application/ports/`)

```go
// internal/application/ports/user.go
type UserPort interface {
    GetUser(id string) (dto.UserOutput, error)
}
```

### Paso 4: Crear Caso de Uso (`internal/application/users/`)

```go
// internal/application/users/usecase.go
type UserUseCase struct{}

func (u *UserUseCase) GetUser(id string) (dto.UserOutput, error) {
    return dto.UserOutput{ID: id, Name: "Test"}, nil
}
```

### Paso 5: Crear Handler (`internal/infrastructure/handlers/`)

```go
// internal/infrastructure/handlers/user.go
type UserHandler struct {
    usecase ports.UserPort
    logger  *zerolog.Logger
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
    output, err := h.usecase.GetUser("123")
    if err != nil {
        response.Error(w, http.StatusNotFound, "not_found", err.Error())
        return
    }
    response.JSON(w, http.StatusOK, output)
}
```

### Paso 6: Registrar Ruta (`internal/infrastructure/router/`)

```go
// internal/infrastructure/router/router.go
r.HandleFunc("/users/{id}", userHandler.Get).Methods(http.MethodGet)
```

### Paso 7: Conectar en `cmd/server/main.go`

```go
userUseCase := users.NewUserUseCase()
userHandler := handlers.NewUserHandler(userUseCase, log)
r := router.NewRouter(userHandler)
```

---

## Docker

### Dockerfile (Multi-stage)

```dockerfile
# Builder
FROM golang:1.22-alpine AS builder
RUN go build -o server ./cmd/server

# Runtime
FROM alpine:3.19
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

### docker-compose.yml

```yaml
services:
  ms-storage:
    build: .
    ports:
      - "8080:8080"
    environment:
      - LOG_FORMAT=json
      - LOG_LEVEL=info
    healthcheck:
      test: ["CMD", "wget", "-q", "http://localhost:8080/health"]
```

---

## Testing

```bash
go test ./...
go test -cover ./...
go test -v ./internal/domain/...
```

---

## Principios Aplicados

1. **Dependency Rule**: Dependencias solo hacia el core
2. **Interface Separation**: Puertos definen abstracciones
3. **Single Responsibility**: Cada capa tiene una responsabilidad
4. **Testability**: Core testable sin infraestructura
5. **Scalability**: Estructura por feature, no por tipo
6. **Zero Allocation**: Zerolog para logging performant

---

## Recursos

- [Clean Architecture - Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [Gorilla Mux](https://github.com/gorilla/mux)
- [Zerolog](https://github.com/rs/zerolog)
