package logging

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	loggerKey    contextKey = "logger"
)

func New(service string, level string, format string) *zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	var output io.Writer = os.Stdout

	if format == "console" {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}

	l := zerolog.New(output).
		With().
		Timestamp().
		Str("service", service).
		Logger()

	switch level {
	case "debug":
		l = l.Level(zerolog.DebugLevel)
	case "info":
		l = l.Level(zerolog.InfoLevel)
	case "warn":
		l = l.Level(zerolog.WarnLevel)
	case "error":
		l = l.Level(zerolog.ErrorLevel)
	default:
		l = l.Level(zerolog.InfoLevel)
	}

	return &l
}

func NewWithOutput(w io.Writer, service string) *zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	l := zerolog.New(w).
		With().
		Timestamp().
		Str("service", service).
		Logger()

	return &l
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

func WithLogger(ctx context.Context, logger *zerolog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *zerolog.Logger {
	if l, ok := ctx.Value(loggerKey).(*zerolog.Logger); ok {
		return l
	}
	l := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return &l
}

type ResponseWriter struct {
	statusCode int
	length     int
	w          io.Writer
}

func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
}

func (rw *ResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.w.Write(b)
	rw.length += n
	return n, err
}
