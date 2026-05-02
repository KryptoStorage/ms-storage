package main

import (
	"context"
	"fmt"
	"ms-storage/internal/application/health"
	"ms-storage/internal/config"
	"ms-storage/internal/infrastructure/handlers"
	"ms-storage/internal/infrastructure/middleware"
	"ms-storage/internal/infrastructure/router"
	"ms-storage/pkg/logging"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("No .env file found, using defaults\n")
	}

	cfg := config.Load()

	log := logging.New(cfg.App.Name, cfg.Log.Level, cfg.Log.Format)

	log.Info().
		Str("event", "startup").
		Str("version", cfg.App.Version).
		Str("env", cfg.App.Env).
		Msg("Starting ms-storage")

	healthUseCase := health.NewHealthUseCase(cfg.App.Version, cfg.App.Name)
	healthHandler := handlers.NewHealthHandler(healthUseCase, log)

	r := router.NewRouter(healthHandler)
	r.Use(middleware.RequestLogger(log))
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().
			Str("event", "server_started").
			Str("addr", srv.Addr).
			Msg("HTTP server listening")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("Server failed")
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().
		Str("event", "shutdown_started").
		Msg("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
		os.Exit(1)
	}

	log.Info().
		Str("event", "shutdown_completed").
		Msg("Server exited gracefully")
}
