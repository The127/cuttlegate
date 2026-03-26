package httpadapter_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/The127/cuttlegate/internal/adapters/http"
	"github.com/The127/cuttlegate/internal/app"
	"github.com/The127/cuttlegate/internal/domain"
)

// authAs returns middleware that injects an AuthContext for the given userID.
func authAs(userID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := domain.NewAuthContext(r.Context(), domain.AuthContext{
				UserID: userID,
				Role:   domain.RoleViewer,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// fireEvalRequest fires one POST to the evaluation endpoint and returns the status code.
func fireEvalRequest(mux *http.ServeMux) int {
	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluate",
		strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w.Code
}

// newRateLimitedEvalMux wires the evaluation handler behind auth + limiter.
func newRateLimitedEvalMux(svc *fakeEvaluationService, userID string, limit int, window time.Duration) *http.ServeMux {
	rl := httpadapter.NewRateLimiter(limit, window)
	auth := func(h http.Handler) http.Handler {
		return authAs(userID)(rl.Limit(h))
	}
	return newEvalMux(svc, auth)
}

func enabledView() *app.EvalView {
	return &app.EvalView{Key: "my-flag", Enabled: true, Reason: domain.ReasonDefault, Type: domain.FlagTypeBool}
}

// Scenario D: N requests within window all succeed.
func TestRateLimiter_WithinLimit_Returns200(t *testing.T) {
	mux := newRateLimitedEvalMux(&fakeEvaluationService{view: enabledView()}, "user-a", 3, time.Hour)
	for i := 0; i < 3; i++ {
		if code := fireEvalRequest(mux); code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, code)
		}
	}
}

// Scenario E: N+1 request within window returns 429.
func TestRateLimiter_ExceedsLimit_Returns429(t *testing.T) {
	mux := newRateLimitedEvalMux(&fakeEvaluationService{view: enabledView()}, "user-a", 2, time.Hour)
	fireEvalRequest(mux) // 1
	fireEvalRequest(mux) // 2
	if code := fireEvalRequest(mux); code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", code)
	}
}

// Scenario F: window expiry resets the counter.
func TestRateLimiter_WindowExpiry_ResetsCounter(t *testing.T) {
	mux := newRateLimitedEvalMux(&fakeEvaluationService{view: enabledView()}, "user-a", 1, 50*time.Millisecond)

	if code := fireEvalRequest(mux); code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", code)
	}
	if code := fireEvalRequest(mux); code != http.StatusTooManyRequests {
		t.Fatalf("second request (over limit): expected 429, got %d", code)
	}
	time.Sleep(60 * time.Millisecond) // let the window expire
	if code := fireEvalRequest(mux); code != http.StatusOK {
		t.Fatalf("after window expiry: expected 200, got %d", code)
	}
}

// Scenario G: two users share no quota — a shared RateLimiter keys independently per user.
func TestRateLimiter_DifferentUsers_IndependentQuotas(t *testing.T) {
	rl := httpadapter.NewRateLimiter(1, time.Hour)
	svc := &fakeEvaluationService{view: enabledView()}

	proj := &domain.Project{ID: "proj-1", Slug: "my-project", Name: "My Project"}
	env := &domain.Environment{ID: "env-1", ProjectID: "proj-1", Slug: "production"}

	fire := func(userID string) int {
		mux := http.NewServeMux()
		httpadapter.NewEvaluationHandler(
			svc,
			&fakeProjResolver{projects: map[string]*domain.Project{proj.Slug: proj}},
			&fakeEnvResolver{envs: map[string]*domain.Environment{"proj-1/production": env}},
		).RegisterRoutes(mux, func(h http.Handler) http.Handler {
			return authAs(userID)(rl.Limit(h))
		})
		return fireEvalRequest(mux)
	}

	if code := fire("user-a"); code != http.StatusOK {
		t.Fatalf("user-a first: expected 200, got %d", code)
	}
	if code := fire("user-a"); code != http.StatusTooManyRequests {
		t.Fatalf("user-a second: expected 429, got %d", code)
	}
	if code := fire("user-b"); code != http.StatusOK {
		t.Fatalf("user-b first: expected 200 (independent quota), got %d", code)
	}
}

// No AuthContext (limiter applied without auth middleware) → 403.
func TestRateLimiter_NoAuthContext_Returns403(t *testing.T) {
	rl := httpadapter.NewRateLimiter(100, time.Hour)
	mux := newEvalMux(&fakeEvaluationService{}, func(h http.Handler) http.Handler { return rl.Limit(h) })

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluate",
		strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}
