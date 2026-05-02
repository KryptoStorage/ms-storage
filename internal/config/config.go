package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App  AppConfig
	HTTP HTTPConfig
	Log  LogConfig
	CORS CORSConfig
	Rate RateConfig
}

type AppConfig struct {
	Name    string
	Version string
	Env     string
}

type HTTPConfig struct {
	Host            string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	HandlerTimeout  time.Duration
	ShutdownTimeout time.Duration
}

type LogConfig struct {
	Level  string
	Format string
}

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAgeSeconds  int
}

type RateConfig struct {
	Enabled bool
	RPS     float64
	Burst   int
}

var (
	validLogLevels  = map[string]struct{}{"debug": {}, "info": {}, "warn": {}, "error": {}}
	validLogFormats = map[string]struct{}{"json": {}, "console": {}}
	validEnvs       = map[string]struct{}{"development": {}, "staging": {}, "production": {}, "test": {}}
)

func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Name:    getEnv("APP_NAME", "ms-storage"),
			Version: getEnv("APP_VERSION", "1.0.0"),
			Env:     strings.ToLower(getEnv("APP_ENV", "development")),
		},
		HTTP: HTTPConfig{
			Host:            getEnv("HTTP_HOST", "0.0.0.0"),
			Port:            getEnv("HTTP_PORT", "8080"),
			ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:     getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			HandlerTimeout:  getDuration("HTTP_HANDLER_TIMEOUT", 15*time.Second),
			ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Log: LogConfig{
			Level:  strings.ToLower(getEnv("LOG_LEVEL", "info")),
			Format: strings.ToLower(getEnv("LOG_FORMAT", "json")),
		},
		CORS: CORSConfig{
			AllowedOrigins: getCSV("CORS_ALLOWED_ORIGINS", nil),
			AllowedMethods: getCSV("CORS_ALLOWED_METHODS", nil),
			AllowedHeaders: getCSV("CORS_ALLOWED_HEADERS", nil),
			MaxAgeSeconds:  getInt("CORS_MAX_AGE_SECONDS", 600),
		},
		Rate: RateConfig{
			Enabled: getBool("RATE_LIMIT_ENABLED", false),
			RPS:     getFloat("RATE_LIMIT_RPS", 50),
			Burst:   getInt("RATE_LIMIT_BURST", 100),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	var errs []string

	if c.App.Name == "" {
		errs = append(errs, "APP_NAME is required")
	}
	if _, ok := validEnvs[c.App.Env]; !ok {
		errs = append(errs, fmt.Sprintf("APP_ENV %q is not one of development|staging|production|test", c.App.Env))
	}
	if c.HTTP.Host == "" {
		errs = append(errs, "HTTP_HOST is required")
	}
	if p, err := strconv.Atoi(c.HTTP.Port); err != nil || p < 1 || p > 65535 {
		errs = append(errs, fmt.Sprintf("HTTP_PORT %q must be a valid TCP port", c.HTTP.Port))
	}
	if _, ok := validLogLevels[c.Log.Level]; !ok {
		errs = append(errs, fmt.Sprintf("LOG_LEVEL %q must be debug|info|warn|error", c.Log.Level))
	}
	if _, ok := validLogFormats[c.Log.Format]; !ok {
		errs = append(errs, fmt.Sprintf("LOG_FORMAT %q must be json|console", c.Log.Format))
	}
	if c.Rate.Enabled && (c.Rate.RPS <= 0 || c.Rate.Burst <= 0) {
		errs = append(errs, "RATE_LIMIT_RPS and RATE_LIMIT_BURST must be > 0 when RATE_LIMIT_ENABLED")
	}

	if len(errs) > 0 {
		return errors.New("invalid configuration: " + strings.Join(errs, "; "))
	}
	return nil
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getFloat(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func getBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func getCSV(key string, def []string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
