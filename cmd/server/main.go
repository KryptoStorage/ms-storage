package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KryptoStorage/ms-storage/internal/application/health"
	"github.com/KryptoStorage/ms-storage/internal/application/ports"
	"github.com/KryptoStorage/ms-storage/internal/config"
	"github.com/KryptoStorage/ms-storage/internal/infrastructure/handlers"
	"github.com/KryptoStorage/ms-storage/internal/infrastructure/middleware"
	"github.com/KryptoStorage/ms-storage/internal/infrastructure/router"
	"github.com/KryptoStorage/ms-storage/pkg/logging"
	"github.com/KryptoStorage/ms-storage/pkg/shutdown"

	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	// .env is loaded for local development; in production env vars come from
	// the orchestrator. A missing file is not fatal.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logging.New(cfg.App.Name, cfg.Log.Level, cfg.Log.Format)
	log.Info().
		Str("event", "startup").
		Str("version", cfg.App.Version).
		Str("env", cfg.App.Env).
		Msg("Starting ms-storage")

	probes := []ports.ReadinessProbe{}
	healthUC := health.New(health.Options{
		Version:     cfg.App.Version,
		ServiceName: cfg.App.Name,
		Probes:      probes,
	})
	healthHandler := handlers.NewHealthHandler(healthUC, log)

	r := router.New(router.Deps{Health: healthHandler})

	// Order matters: logging is outermost so every request is observed; CORS
	// handles preflights before security headers; rate limiting comes last
	// so denied requests skip downstream work.
	r.Use(middleware.RequestLogger(log))
	r.Use(middleware.SecurityHeaders)
	if len(cfg.CORS.AllowedOrigins) > 0 {
		r.Use(middleware.CORS(middleware.CORSConfig{
			AllowedOrigins: cfg.CORS.AllowedOrigins,
			AllowedMethods: cfg.CORS.AllowedMethods,
			AllowedHeaders: cfg.CORS.AllowedHeaders,
			MaxAgeSeconds:  cfg.CORS.MaxAgeSeconds,
		}))
	}
	if cfg.Rate.Enabled {
		r.Use(middleware.RateLimit(cfg.Rate.RPS, cfg.Rate.Burst))
	}

	handler := http.Handler(r)
	if cfg.HTTP.HandlerTimeout > 0 {
		handler = http.TimeoutHandler(handler, cfg.HTTP.HandlerTimeout,
			`{"error":{"code":"timeout","message":"request timed out"}}`)
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%s", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:           handler,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	registry := shutdown.NewRegistry()
	registry.Register("http_server", srv.Shutdown)

	serverErr := make(chan error, 1)
	go func() {
		log.Info().Str("event", "server_started").Str("addr", srv.Addr).Msg("HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info().Str("event", "shutdown_started").Str("signal", sig.String()).Msg("Shutting down")
	case err := <-serverErr:
		if err != nil {
			log.Error().Err(err).Msg("server failed")
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := registry.Run(ctx); err != nil {
		log.Error().Err(err).Msg("shutdown encountered errors")
		return err
	}

	log.Info().Str("event", "shutdown_completed").Msg("Server exited gracefully")
	return nil
}
