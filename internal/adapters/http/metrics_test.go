package httpadapter

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// newTestRegistry creates an isolated Prometheus registry and re-registers
// the package-level HTTP metrics so parallel tests do not collide with
// the global default registry.
func newTestRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(httpRequestsTotal, httpRequestDuration)
	return reg
}

func TestMetricsMiddleware_RequestCount(t *testing.T) {
	// Reset counters for a clean test.
	httpRequestsTotal.Reset()

	inner := http.NewServeMux()
	inner.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := MetricsMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	count := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "GET /test", "200"))
	if count != 1 {
		t.Errorf("expected request count 1, got %f", count)
	}
}

func TestMetricsMiddleware_Duration(t *testing.T) {
	httpRequestDuration.Reset()

	inner := http.NewServeMux()
	inner.HandleFunc("GET /slow", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := MetricsMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Verify the histogram has at least one observation by collecting it.
	count := testutil.CollectAndCount(httpRequestDuration)
	if count == 0 {
		t.Error("expected at least one duration metric")
	}
}

func TestMetricsMiddleware_StatusCodes(t *testing.T) {
	httpRequestsTotal.Reset()

	inner := http.NewServeMux()
	inner.HandleFunc("GET /ok", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	inner.HandleFunc("GET /notfound", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	inner.HandleFunc("POST /created", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := MetricsMiddleware(inner)

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{"GET", "/ok", 200},
		{"GET", "/notfound", 404},
		{"POST", "/created", 201},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != tc.want {
			t.Errorf("%s %s: expected %d, got %d", tc.method, tc.path, tc.want, rec.Code)
		}
	}

	for _, tc := range tests {
		pattern := tc.method + " " + tc.path
		count := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(tc.method, pattern, strconv.Itoa(tc.want)))
		if count != 1 {
			t.Errorf("%s %s status %d: expected count 1, got %f", tc.method, tc.path, tc.want, count)
		}
	}
}

func TestMetricsEndpoint_ReturnsPrometheusFormat(t *testing.T) {
	reg := newTestRegistry()
	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	// Fire a request through the middleware to generate some metrics.
	httpRequestsTotal.Reset()
	httpRequestDuration.Reset()

	inner := http.NewServeMux()
	inner.HandleFunc("GET /ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := MetricsMiddleware(inner)
	mw.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/ping", nil))

	// Now scrape /metrics.
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") && !strings.Contains(ct, "text/openmetrics") {
		t.Errorf("unexpected Content-Type: %s", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "cuttlegate_http_requests_total") {
		t.Error("/metrics response missing cuttlegate_http_requests_total")
	}
	if !strings.Contains(body, "cuttlegate_http_request_duration_seconds") {
		t.Error("/metrics response missing cuttlegate_http_request_duration_seconds")
	}
}
