package httpadapter

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// ── HTTP metrics ─────────────────────────────────────────────────────────────

var (
	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cuttlegate_http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path_pattern", "status_code"})

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "cuttlegate_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path_pattern"})
)

// ── Application metrics ──────────────────────────────────────────────────────

var (
	// FlagEvaluationsTotal counts flag evaluations by project, environment, and reason.
	FlagEvaluationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cuttlegate_flag_evaluations_total",
		Help: "Total number of flag evaluations.",
	}, []string{"project", "environment", "reason"})

	// SSEConnectionsActive tracks the number of active SSE connections.
	SSEConnectionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cuttlegate_sse_connections_active",
		Help: "Number of active SSE streaming connections.",
	})
)

// RegisterMetrics registers all Prometheus metrics. Call once from main.
func RegisterMetrics() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		FlagEvaluationsTotal,
		SSEConnectionsActive,
	)
}

// ── Middleware ────────────────────────────────────────────────────────────────

// MetricsMiddleware wraps an http.Handler and records request count and duration
// for every HTTP request. The path_pattern label uses the ServeMux route pattern
// (Go 1.22+) to avoid cardinality explosion from path parameters.
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		pattern := r.Pattern
		if pattern == "" {
			pattern = "unknown"
		}

		elapsed := time.Since(start).Seconds()
		status := strconv.Itoa(rw.statusCode)

		httpRequestsTotal.WithLabelValues(r.Method, pattern, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, pattern).Observe(elapsed)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.wroteHeader = true
	}
	return rw.ResponseWriter.Write(b)
}

// Unwrap returns the underlying ResponseWriter, allowing http.Flusher
// and other interface assertions to work through the wrapper.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}
