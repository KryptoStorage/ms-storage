package handlers

import (
	"net/http"

	"github.com/KryptoStorage/ms-storage/internal/application/ports"
	"github.com/KryptoStorage/ms-storage/internal/domain/health"
	"github.com/KryptoStorage/ms-storage/pkg/logging"
	"github.com/KryptoStorage/ms-storage/pkg/response"
	"github.com/rs/zerolog"
)

type HealthHandler struct {
	usecase ports.HealthPort
	logger  *zerolog.Logger
}

func NewHealthHandler(usecase ports.HealthPort, logger *zerolog.Logger) *HealthHandler {
	return &HealthHandler{usecase: usecase, logger: logger}
}

// Liveness is a cheap probe used by orchestrators (Kubernetes livenessProbe)
// to know whether the process is alive. It must not depend on external
// systems — those go in Readiness.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	out := h.usecase.GetHealth(r.Context())

	h.logger.Debug().
		Str("request_id", logging.GetRequestID(r.Context())).
		Str("event", "liveness_ok").
		Msg("liveness probe")

	response.JSON(w, http.StatusOK, out)
}

// Readiness probes downstream dependencies. Returns 503 when the service
// reports anything other than healthy so that load balancers can drain
// traffic away.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	out := h.usecase.GetReadiness(r.Context())

	status := http.StatusOK
	if out.Status != string(health.StatusHealthy) {
		status = http.StatusServiceUnavailable
	}

	h.logger.Info().
		Str("request_id", logging.GetRequestID(r.Context())).
		Str("event", "readiness_probe").
		Str("status", out.Status).
		Int("components", len(out.Components)).
		Msg("readiness probe")

	response.JSON(w, status, out)
}
