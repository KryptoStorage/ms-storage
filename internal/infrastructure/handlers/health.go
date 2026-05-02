package handlers

import (
	"ms-storage/internal/application/ports"
	"ms-storage/pkg/logging"
	"ms-storage/pkg/response"
	"net/http"

	"github.com/rs/zerolog"
)

type HealthHandler struct {
	usecase ports.HealthPort
	logger  *zerolog.Logger
}

func NewHealthHandler(usecase ports.HealthPort, logger *zerolog.Logger) *HealthHandler {
	return &HealthHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	requestID := logging.GetRequestID(r.Context())

	log := h.logger.With().
		Str("request_id", requestID).
		Str("endpoint", "/health").
		Logger()

	log.Debug().
		Str("event", "health_check_requested").
		Msg("Health check endpoint called")

	output := h.usecase.GetHealth()

	log.Info().
		Str("event", "health_check_completed").
		Str("status", string(output.Status)).
		Str("version", output.Version).
		Msg("Health check response sent")

	response.JSON(w, http.StatusOK, output)
}

func (h *HealthHandler) Sync(w http.ResponseWriter, r *http.Request) {
	requestID := logging.GetRequestID(r.Context())

	log := h.logger.With().
		Str("request_id", requestID).
		Str("endpoint", "/health/sync").
		Logger()

	log.Debug().
		Str("event", "sync_status_requested").
		Msg("Sync status endpoint called")

	output := h.usecase.GetSync()

	log.Info().
		Str("event", "sync_status_completed").
		Str("status", string(output.Status)).
		Str("state", output.State).
		Msg("Sync status response sent")

	response.JSON(w, http.StatusOK, output)
}
