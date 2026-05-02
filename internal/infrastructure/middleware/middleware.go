package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KryptoStorage/ms-storage/pkg/logging"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

// CORSConfig drives the CORS middleware.
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAgeSeconds  int
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	length     int
	wroteHead  bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHead {
		return
	}
	rw.statusCode = code
	rw.wroteHead = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHead {
		rw.statusCode = http.StatusOK
		rw.wroteHead = true
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.length += n
	return n, err
}

// RequestLogger injects a request_id into the context, logs the request, and
// records status/bytes/duration once it is served.
func RequestLogger(logger *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.NewString()
			}
			w.Header().Set("X-Request-ID", requestID)

			ctx := logging.WithRequestID(r.Context(), requestID)
			r = r.WithContext(ctx)

			log := logger.With().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Logger()

			log.Info().Str("event", "request_started").Msg("Incoming request")

			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			log.Info().
				Str("event", "request_completed").
				Int("status", wrapped.statusCode).
				Int("bytes", wrapped.length).
				Dur("duration", time.Since(start)).
				Msg("Request completed")
		})
	}
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// CORS returns a middleware that honours the supplied configuration. An empty
// AllowedOrigins slice disables CORS.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	allowAll := len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*"
	originSet := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		originSet[o] = struct{}{}
	}

	methods := strings.Join(defaultIfEmpty(cfg.AllowedMethods,
		[]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}), ", ")
	headers := strings.Join(defaultIfEmpty(cfg.AllowedHeaders,
		[]string{"Content-Type", "Authorization", "X-Request-ID"}), ", ")
	maxAge := cfg.MaxAgeSeconds
	if maxAge <= 0 {
		maxAge = 600
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if allowAll {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else if _, ok := originSet[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", "Origin")
				}
				w.Header().Set("Access-Control-Allow-Methods", methods)
				w.Header().Set("Access-Control-Allow-Headers", headers)
				w.Header().Set("Access-Control-Max-Age", itoa(maxAge))
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit applies a simple in-process token bucket per remote IP. Suitable
// for a single replica or behind a sticky load balancer; for fleet-wide
// throttling, use a centralised limiter.
func RateLimit(rps float64, burst int) func(http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	var (
		clients = map[string]*client{}
		mu      sync.Mutex
	)

	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for now := range t.C {
			mu.Lock()
			for ip, c := range clients {
				if now.Sub(c.lastSeen) > 5*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)

			mu.Lock()
			c, ok := clients[ip]
			if !ok {
				c = &client{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
				clients[ip] = c
			}
			c.lastSeen = time.Now()
			allowed := c.limiter.Allow()
			mu.Unlock()

			if !allowed {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Timeout wraps the handler with a request-scoped timeout. The handler must
// honour ctx.Done(); use http.TimeoutHandler at the server level for hard
// caps on response writing.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, `{"error":"timeout","message":"request timed out"}`)
	}
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if i := strings.IndexByte(fwd, ','); i >= 0 {
			return strings.TrimSpace(fwd[:i])
		}
		return strings.TrimSpace(fwd)
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if i := strings.LastIndexByte(r.RemoteAddr, ':'); i >= 0 {
		return r.RemoteAddr[:i]
	}
	return r.RemoteAddr
}

func defaultIfEmpty(v, fallback []string) []string {
	if len(v) == 0 {
		return fallback
	}
	return v
}

func itoa(i int) string { return strconv.Itoa(i) }
