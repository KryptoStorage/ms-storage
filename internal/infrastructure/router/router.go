package router

import (
	"net/http"

	"github.com/KryptoStorage/ms-storage/internal/infrastructure/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Deps struct {
	Health *handlers.HealthHandler
}

// New builds the HTTP router. Operational endpoints (/livez, /readyz,
// /metrics) live at the root so that probes and scrapers do not need to know
// the API version. Business endpoints hang off /api/v1.
func New(d Deps) *mux.Router {
	r := mux.NewRouter()
	r.StrictSlash(true)

	r.HandleFunc("/livez", d.Health.Liveness).Methods(http.MethodGet)
	r.HandleFunc("/readyz", d.Health.Readiness).Methods(http.MethodGet)
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)

	v1 := r.PathPrefix("/api/v1").Subrouter()
	_ = v1 // reserved; register feature handlers here as the service grows.

	return r
}
