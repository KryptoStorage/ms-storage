package logging

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// New builds a configured *zerolog.Logger. Format must be "json" or "console".
// Level is debug|info|warn|error and falls back to info on unknown values.
func New(service, level, format string) *zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	var output io.Writer = os.Stdout
	if format == "console" {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}

	l := zerolog.New(output).
		Level(parseLevel(level)).
		With().
		Timestamp().
		Str("service", service).
		Logger()

	return &l
}

func parseLevel(s string) zerolog.Level {
	switch s {
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}
