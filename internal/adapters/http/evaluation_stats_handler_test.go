package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpadapter "github.com/karo/cuttlegate/internal/adapters/http"
	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// fakeEvaluationStatsService is a test double for the evaluationStatsService interface.
type fakeEvaluationStatsService struct {
	view        *app.FlagStatsView
	err         error
	bucketsView *app.EvaluationBucketsView
	bucketsErr  error
}

func (f *fakeEvaluationStatsService) GetStats(_ context.Context, _, _, _ string) (*app.FlagStatsView, error) {
	return f.view, f.err
}

func (f *fakeEvaluationStatsService) GetBuckets(_ context.Context, _, _, _, _, _, _ string) (*app.EvaluationBucketsView, error) {
	return f.bucketsView, f.bucketsErr
}

func newStatsMux(svc *fakeEvaluationStatsService, auth func(http.Handler) http.Handler) *http.ServeMux {
	proj := &domain.Project{ID: "proj-1", Slug: "my-project", Name: "My Project", CreatedAt: time.Now()}
	env := &domain.Environment{ID: "env-1", ProjectID: "proj-1", Slug: "production", Name: "Production"}

	mux := http.NewServeMux()
	httpadapter.NewEvaluationStatsHandler(
		svc,
		&fakeProjResolver{projects: map[string]*domain.Project{proj.Slug: proj}},
		&fakeEnvResolver{envs: map[string]*domain.Environment{"proj-1/production": env}},
	).RegisterRoutes(mux, auth)
	return mux
}

// @happy: flag has evaluations — returns count and last_evaluated_at.
func TestEvaluationStatsHandler_Happy(t *testing.T) {
	ts := time.Date(2026, 3, 21, 14, 32, 0, 0, time.UTC)
	svc := &fakeEvaluationStatsService{
		view: &app.FlagStatsView{
			LastEvaluatedAt: &ts,
			EvaluationCount: 4821,
		},
	}
	mux := newStatsMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/stats", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["evaluation_count"] != float64(4821) {
		t.Errorf("evaluation_count: want 4821, got %v", resp["evaluation_count"])
	}
	if resp["last_evaluated_at"] == nil {
		t.Errorf("last_evaluated_at: want non-null, got nil")
	}
}

// @edge: flag never evaluated — returns count 0 and last_evaluated_at null.
func TestEvaluationStatsHandler_NeverEvaluated(t *testing.T) {
	svc := &fakeEvaluationStatsService{
		view: &app.FlagStatsView{
			LastEvaluatedAt: nil,
			EvaluationCount: 0,
		},
	}
	mux := newStatsMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/new-flag/stats", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["evaluation_count"] != float64(0) {
		t.Errorf("evaluation_count: want 0, got %v", resp["evaluation_count"])
	}
	if v, present := resp["last_evaluated_at"]; present && v != nil {
		t.Errorf("last_evaluated_at: want null, got %v", v)
	}
}

// @error-path: flag not found returns 404 with {"error": "not_found"}.
func TestEvaluationStatsHandler_FlagNotFound(t *testing.T) {
	svc := &fakeEvaluationStatsService{err: domain.ErrNotFound}
	mux := newStatsMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/no-such-flag/stats", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "not_found" {
		t.Errorf("error: want not_found, got %q", body["error"])
	}
}

// @auth-bypass: unauthenticated request returns 401.
func TestEvaluationStatsHandler_Unauthorized(t *testing.T) {
	svc := &fakeEvaluationStatsService{}
	mux := newStatsMux(svc, rejectAllAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/stats", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

// @happy: buckets endpoint returns 200 with correct shape.
func TestEvaluationStatsHandler_GetBuckets_Happy(t *testing.T) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	buckets := make([]app.BucketView, 7)
	for i := range buckets {
		buckets[i] = app.BucketView{
			Timestamp: now.Add(time.Duration(i-6) * 24 * time.Hour),
			Total:     0,
			Variants:  map[string]int64{},
		}
	}
	buckets[1].Total = 142
	buckets[1].Variants = map[string]int64{"enabled": 98, "disabled": 44}

	svc := &fakeEvaluationStatsService{
		bucketsView: &app.EvaluationBucketsView{
			FlagKey:     "my-flag",
			Environment: "production",
			Window:      "7d",
			BucketSize:  "day",
			Buckets:     buckets,
		},
	}
	mux := newStatsMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/stats/buckets?window=7d&bucket=day", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["flag_key"] != "my-flag" {
		t.Errorf("flag_key: want my-flag, got %v", resp["flag_key"])
	}
	if resp["window"] != "7d" {
		t.Errorf("window: want 7d, got %v", resp["window"])
	}
	if resp["bucket_size"] != "day" {
		t.Errorf("bucket_size: want day, got %v", resp["bucket_size"])
	}
	bkts, ok := resp["buckets"].([]interface{})
	if !ok || len(bkts) != 7 {
		t.Fatalf("buckets: want 7 entries, got %v", resp["buckets"])
	}
}

// @error-path: missing window/bucket params returns 400 {"error": "invalid_parameter"}.
func TestEvaluationStatsHandler_GetBuckets_MissingParams(t *testing.T) {
	svc := &fakeEvaluationStatsService{}
	mux := newStatsMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/stats/buckets", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "invalid_parameter" {
		t.Errorf("error: want invalid_parameter, got %q", body["error"])
	}
}

// @error-path: invalid window returns 400 {"error": "invalid_parameter"}.
func TestEvaluationStatsHandler_GetBuckets_InvalidWindow(t *testing.T) {
	svc := &fakeEvaluationStatsService{bucketsErr: app.ErrInvalidParameter}
	mux := newStatsMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/stats/buckets?window=45d&bucket=day", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "invalid_parameter" {
		t.Errorf("error: want invalid_parameter, got %q", body["error"])
	}
}

// @error-path: flag not found on buckets endpoint returns 404.
func TestEvaluationStatsHandler_GetBuckets_FlagNotFound(t *testing.T) {
	svc := &fakeEvaluationStatsService{bucketsErr: domain.ErrNotFound}
	mux := newStatsMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/no-such-flag/stats/buckets?window=7d&bucket=day", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "not_found" {
		t.Errorf("error: want not_found, got %q", body["error"])
	}
}

// @auth-bypass: unauthenticated request to buckets returns 401.
func TestEvaluationStatsHandler_GetBuckets_Unauthorized(t *testing.T) {
	svc := &fakeEvaluationStatsService{}
	mux := newStatsMux(svc, rejectAllAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/stats/buckets?window=7d&bucket=day", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

// @auth-bypass: API key token rejected — returns 401 {"error": "unauthorized"}.
// The buckets route uses RequireBearer (OIDC-only). rejectAllAuth simulates
// the OIDC verifier failing on a cg_ API key token.
func TestEvaluationStatsHandler_GetBuckets_APIKeyRejected(t *testing.T) {
	svc := &fakeEvaluationStatsService{}
	mux := newStatsMux(svc, rejectAllAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/stats/buckets?window=7d&bucket=day", nil)
	req.Header.Set("Authorization", "Bearer cg_fakekeyvalue")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "unauthorized" {
		t.Errorf("error: want unauthorized, got %q", body["error"])
	}
}
