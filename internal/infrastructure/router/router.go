package router

import (
	"ms-storage/internal/infrastructure/handlers"
	"net/http"

	"github.com/gorilla/mux"
)

type Router struct {
	*mux.Router
}

func NewRouter(healthHandler *handlers.HealthHandler) *Router {
	r := mux.NewRouter()

	r.HandleFunc("/health", healthHandler.Health).Methods(http.MethodGet)
	r.HandleFunc("/health/sync", healthHandler.Sync).Methods(http.MethodGet)

	return &Router{Router: r}
}
