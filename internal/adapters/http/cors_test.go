package httpadapter_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httpadapter "github.com/The127/cuttlegate/internal/adapters/http"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestCORS_AllowedOrigin_SetsHeaders(t *testing.T) {
	mw := httpadapter.CORS([]string{"https://app.example.com"})
	h := mw(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("ACAO: got %q, want %q", got, "https://app.example.com")
	}
	if got := w.Header().Get("Vary"); got == "" {
		t.Error("Vary header must be set when ACAO is added")
	}
}

func TestCORS_DisallowedOrigin_NoHeaders(t *testing.T) {
	mw := httpadapter.CORS([]string{"https://app.example.com"})
	h := mw(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO for disallowed origin, got %q", got)
	}
}

func TestCORS_NoOrigin_NoHeaders(t *testing.T) {
	mw := httpadapter.CORS([]string{"https://app.example.com"})
	h := mw(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO when no Origin, got %q", got)
	}
}

func TestCORS_Preflight_AllowedOrigin_SetsMethodsAndHeaders(t *testing.T) {
	mw := httpadapter.CORS([]string{"https://app.example.com"})
	h := mw(okHandler())

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Access-Control-Allow-Methods must be set on OPTIONS preflight")
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("Access-Control-Allow-Headers must be set on OPTIONS preflight")
	}
}

func TestCORS_Preflight_DisallowedOrigin_NoMethodsOrHeaders(t *testing.T) {
	mw := httpadapter.CORS([]string{"https://app.example.com"})
	h := mw(okHandler())

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Errorf("expected no ACAM for disallowed origin, got %q", got)
	}
}

func TestCORS_EmptyOriginList_IsNoop(t *testing.T) {
	mw := httpadapter.CORS(nil)
	h := mw(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO when origins list is nil, got %q", got)
	}
}

func TestCORS_TrimsWhitespace(t *testing.T) {
	mw := httpadapter.CORS([]string{" https://app.example.com "})
	h := mw(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("expected origin to match after trimming whitespace, got %q", got)
	}
}

func TestCORS_NoCredentialsHeader(t *testing.T) {
	mw := httpadapter.CORS([]string{"https://app.example.com"})
	h := mw(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Access-Control-Allow-Credentials must not be set, got %q", got)
	}
}

func TestCORS_MultipleAllowedOrigins(t *testing.T) {
	mw := httpadapter.CORS([]string{"https://app.example.com", "http://localhost:3000"})
	h := mw(okHandler())

	for _, origin := range []string{"https://app.example.com", "http://localhost:3000"} {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("origin %q: expected ACAO=%q, got %q", origin, origin, got)
		}
	}
}
