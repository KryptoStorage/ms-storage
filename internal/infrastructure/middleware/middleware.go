package middleware

import (
	"context"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KryptoStorage/ms-storage/pkg/logging"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

// Recover catches panics in downstream handlers/middleware, logs them with the
// active request_id and stack trace, and replies with a normalised 500. It
// re-panics on http.ErrAbortHandler — that sentinel is intentional and must
// reach the server so the connection is closed.
func Recover(logger *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				logger.Error().
					Str("event", "panic_recovered").
					Str("request_id", logging.GetRequestID(r.Context())).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Interface("panic", rec).
					Bytes("stack", debug.Stack()).
					Msg("recovered from panic")

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"internal error"}}`))
			}()
			next.ServeHTTP(w, r)
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
// AllowedOrigins slice disables CORS. Vary: Origin is always added when the
// request carries an Origin so caches do not collapse responses across
// different origins.
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
				w.Header().Add("Vary", "Origin")
				switch {
				case allowAll:
					w.Header().Set("Access-Control-Allow-Origin", "*")
				default:
					if _, ok := originSet[origin]; ok {
						w.Header().Set("Access-Control-Allow-Origin", origin)
					}
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

// RateLimiter is an in-process per-IP token bucket. Suitable for a single
// replica or behind a sticky load balancer; for fleet-wide throttling, use a
// centralised limiter. Call Shutdown on process termination to stop the
// background eviction goroutine.
type RateLimiter struct {
	rps     float64
	burst   int
	mu      sync.Mutex
	clients map[string]*rateClient
	stop    chan struct{}
	stopped chan struct{}
	once    sync.Once
}

type rateClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter starts the eviction goroutine immediately. The caller is
// responsible for invoking Shutdown when the limiter is no longer needed.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		rps:     rps,
		burst:   burst,
		clients: map[string]*rateClient{},
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go rl.evict()
	return rl
}

func (rl *RateLimiter) evict() {
	defer close(rl.stopped)
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-rl.stop:
			return
		case now := <-t.C:
			rl.mu.Lock()
			for ip, c := range rl.clients {
				if now.Sub(c.lastSeen) > 5*time.Minute {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Shutdown stops the eviction goroutine. It is safe to call multiple times and
// returns ctx.Err() if the goroutine does not exit before the deadline.
func (rl *RateLimiter) Shutdown(ctx context.Context) error {
	rl.once.Do(func() { close(rl.stop) })
	select {
	case <-rl.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Middleware enforces the limit on each incoming request, keyed by client IP.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		rl.mu.Lock()
		c, ok := rl.clients[ip]
		if !ok {
			c = &rateClient{limiter: rate.NewLimiter(rate.Limit(rl.rps), rl.burst)}
			rl.clients[ip] = c
		}
		c.lastSeen = time.Now()
		allowed := c.limiter.Allow()
		rl.mu.Unlock()

		if !allowed {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Timeout wraps the handler with a request-scoped timeout. The handler must
// honour ctx.Done(); use http.TimeoutHandler at the server level for hard
// caps on response writing.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, `{"error":{"code":"timeout","message":"request timed out"}}`)
	}
}

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests processed, labelled by method, route template and status.",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)
)

// Metrics records request count and latency labelled by route template (not
// raw URL) so high-cardinality paths do not blow up Prometheus storage.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		path := routeTemplate(r)
		status := strconv.Itoa(wrapped.statusCode)
		httpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path, status).Observe(time.Since(start).Seconds())
	})
}

func routeTemplate(r *http.Request) string {
	if route := mux.CurrentRoute(r); route != nil {
		if tpl, err := route.GetPathTemplate(); err == nil {
			return tpl
		}
	}
	return "unmatched"
}

// clientIP extracts the best-effort caller IP from common forwarding headers
// and falls back to RemoteAddr. Uses net.SplitHostPort so IPv6 addresses such
// as "[::1]:8080" are parsed correctly.
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
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
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
