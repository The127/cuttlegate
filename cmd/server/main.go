package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"

	dbmigrations "github.com/The127/cuttlegate/db"
	dbadapter "github.com/The127/cuttlegate/internal/adapters/db"
	httpadapter "github.com/The127/cuttlegate/internal/adapters/http"
	mcpadapter "github.com/The127/cuttlegate/internal/adapters/mcp"
	"github.com/The127/cuttlegate/internal/app"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	// ── Auto-migrate (dev/test only) ──────────────────────────────────────────

	if cfg.AutoMigrate {
		log.Println("WARNING: AUTO_MIGRATE is enabled — this is not safe for production deployments")
		if cfg.DSN == "" {
			return errors.New("AUTO_MIGRATE=true requires DATABASE_URL")
		}
		if err := runMigrations(cfg.DSN); err != nil {
			return fmt.Errorf("auto-migrate: %w", err)
		}
	}

	// ── OIDC verifier ─────────────────────────────────────────────────────────

	verifier, err := httpadapter.NewOIDCVerifier(
		context.Background(),
		cfg.OIDCIssuer,
		cfg.OIDCAudience,
		cfg.OIDCRoleClaim,
		cfg.OIDCMissingRolePolicy,
		cfg.OIDCRoleMapper,
		slog.Default(),
	)
	if err != nil {
		return fmt.Errorf("oidc: %w", err)
	}

	// ── Event broker ─────────────────────────────────────────────────────────

	broker := httpadapter.NewBroker(64)

	// ── HTTP mux ──────────────────────────────────────────────────────────────

	mux := http.NewServeMux()

	// Public: SPA OIDC config — no auth required.
	spaAuthority := cfg.OIDCSPAAuthority
	if spaAuthority == "" {
		spaAuthority = cfg.OIDCIssuer
	}
	var logoURL *string
	if cfg.UILogoURL != "" {
		logoURL = &cfg.UILogoURL
	}
	clientCfg := spaClientConfig{
		Authority:    spaAuthority,
		ClientID:     cfg.OIDCClientID,
		RedirectURI:  cfg.OIDCRedirectURI,
		AppName:      cfg.UIAppName,
		LogoURL:      logoURL,
		AccentColour: cfg.UIAccentColour,
	}
	mux.HandleFunc("GET /api/v1/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clientCfg) //nolint:errcheck
	})

	// ── Signal context (needed by retention worker before server start) ───────

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── Database connection ──────────────────────────────────────────────────

	var conn *sql.DB
	if cfg.DSN != "" {
		conn, err = sql.Open("postgres", cfg.DSN)
		if err != nil {
			return fmt.Errorf("db: %w", err)
		}
		defer conn.Close() //nolint:errcheck
	}

	// Authenticated API routes — wired when DATABASE_URL is available.
	if conn != nil {
		projRepo := dbadapter.NewPostgresProjectRepository(conn)
		envRepo := dbadapter.NewPostgresEnvironmentRepository(conn)
		memberRepo := dbadapter.NewPostgresProjectMemberRepository(conn)
		userRepo := dbadapter.NewPostgresUserRepository(conn)
		flagRepo := dbadapter.NewPostgresFlagRepository(conn)
		stateRepo := dbadapter.NewPostgresFlagEnvironmentStateRepository(conn)
		ruleRepo := dbadapter.NewPostgresRuleRepository(conn)
		segmentRepo := dbadapter.NewPostgresSegmentRepository(conn)
		apiKeyRepo := dbadapter.NewPostgresAPIKeyRepository(conn)
		auditRepo := dbadapter.NewPostgresAuditRepository(conn)
		evalEventRepo := dbadapter.NewPostgresEvaluationEventRepository(conn)
		evalStatsRepo := dbadapter.NewPostgresFlagEvaluationStatsRepository(conn)

		requireBearer := httpadapter.RequireBearer(verifier, userRepo)
		tenantRLS := httpadapter.TenantRLS(conn, projRepo)
		// requireBearerWithRLS chains auth → tenant RLS for project-scoped routes.
		requireBearerWithRLS := func(h http.Handler) http.Handler {
			return requireBearer(tenantRLS(h))
		}

		projSvc := app.NewProjectService(projRepo)
		envSvc := app.NewEnvironmentService(envRepo, flagRepo, stateRepo)
		memberSvc := app.NewProjectMemberService(memberRepo, projRepo, userRepo)
		flagSvc := app.NewFlagService(flagRepo, envRepo, stateRepo, ruleRepo, broker, auditRepo)
		ruleSvc := app.NewRuleService(ruleRepo)
		segmentSvc := app.NewSegmentService(segmentRepo)
		evalSvc := app.NewEvaluationService(flagRepo, stateRepo, ruleRepo, segmentRepo, evalEventRepo).WithStatsRepo(evalStatsRepo).WithAuditRepo(auditRepo)
		evalAuditSvc := app.NewEvaluationAuditService(evalEventRepo, flagRepo)
		evalStatsSvc := app.NewEvaluationStatsService(evalStatsRepo, flagRepo)
		apiKeySvc := app.NewAPIKeyService(apiKeyRepo)
		auditSvc := app.NewAuditService(auditRepo)
		// Start retention worker — runs in background, exits when ctx is cancelled.
		dbadapter.StartEvaluationRetentionWorker(ctx, evalEventRepo, cfg.EvalEventRetentionDays, cfg.EvalEventRetentionInterval)

		httpadapter.NewProjectHandler(projSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewEnvironmentHandler(envSvc, projSvc).RegisterRoutes(mux, requireBearerWithRLS)
		httpadapter.NewProjectMemberHandler(memberSvc).RegisterRoutes(mux, requireBearerWithRLS)
		httpadapter.NewFlagHandler(flagSvc, projSvc).RegisterRoutes(mux, requireBearerWithRLS)
		httpadapter.NewFlagVariantHandler(flagSvc, projSvc).RegisterRoutes(mux, requireBearerWithRLS)
		httpadapter.NewFlagEnvironmentHandler(flagSvc, projSvc, envSvc).RegisterRoutes(mux, requireBearerWithRLS)
		httpadapter.NewRuleHandler(ruleSvc, projSvc, flagSvc, envSvc).RegisterRoutes(mux, requireBearerWithRLS)
		httpadapter.NewSegmentHandler(segmentSvc, projSvc).RegisterRoutes(mux, requireBearerWithRLS)
		httpadapter.NewAPIKeyHandler(apiKeySvc, projSvc, envSvc).RegisterRoutes(mux, requireBearerWithRLS)
		evalRateLimiter := httpadapter.NewRateLimiter(cfg.EvalRateLimit, cfg.EvalRateLimitWindow)
		requireBearerOrAPIKey := httpadapter.RequireBearerOrAPIKey(verifier, apiKeySvc)
		evalAuth := func(h http.Handler) http.Handler { return requireBearerOrAPIKey(tenantRLS(evalRateLimiter.Limit(h))) }
		httpadapter.NewEvaluationHandler(evalSvc, projSvc, envSvc).RegisterRoutes(mux, evalAuth)
		httpadapter.NewEvaluationAuditHandler(evalAuditSvc, projSvc, envSvc).RegisterRoutes(mux, requireBearerWithRLS)
		httpadapter.NewEvaluationStatsHandler(evalStatsSvc, projSvc, envSvc).RegisterRoutes(mux, requireBearerWithRLS)

		sseAuth := func(h http.Handler) http.Handler { return requireBearerOrAPIKey(tenantRLS(h)) }
		httpadapter.NewSSEHandler(broker, projSvc, envSvc).RegisterRoutes(mux, sseAuth)
		httpadapter.NewAuditHandler(auditSvc, projSvc).RegisterRoutes(mux, requireBearerWithRLS)
		// ── MCP server ────────────────────────────────────────────────────────
		mcpMux := http.NewServeMux()
		mcpSrv := mcpadapter.NewServer(apiKeySvc, apiKeyRepo, flagSvc, evalSvc, projSvc, envSvc)
		mcpSrv.RegisterRoutes(mcpMux)
		mcpServer := &http.Server{
			Addr:    cfg.MCPAddr,
			Handler: mcpMux,
		}
		go func() {
			log.Printf("mcp: listening on %s", cfg.MCPAddr)
			if err := mcpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("mcp: server error: %v", err)
			}
		}()
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := mcpServer.Shutdown(shutdownCtx); err != nil {
				log.Printf("mcp: shutdown error: %v", err)
			}
		}()
	}

	// Health checks — public, no auth required.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if conn == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not_ready","reason":"no database configured"}`)) //nolint:errcheck
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := conn.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not_ready","reason":"database unreachable"}`)) //nolint:errcheck
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	// /health — operator-facing deployment readiness probe. 500ms timeout; structured JSON response.
	// Distinct from /healthz (liveness, always 200) and /readyz (Kubernetes-style readiness).
	mux.HandleFunc("GET /health", healthHandler(conn))

	// SPA static files — registered last so /api/v1/* routes take precedence.
	serveSPA(mux)

	// ── Server ────────────────────────────────────────────────────────────────

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
	}

	go func() {
		log.Printf("listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = srv.Shutdown(shutdownCtx)
	broker.Shutdown()
	return err
}

// spaClientConfig is the public config returned to the SPA at GET /api/v1/config.
// Contains OIDC discovery fields and operator-controlled branding.
// Only values safe to expose to the browser are included.
type spaClientConfig struct {
	Authority    string  `json:"authority"`
	ClientID     string  `json:"client_id"`
	RedirectURI  string  `json:"redirect_uri"`
	AppName      string  `json:"app_name"`
	LogoURL      *string `json:"logo_url"`
	AccentColour string  `json:"accent_colour"`
}

// healthHandler returns an http.HandlerFunc for GET /health.
// Returns 200 {"status":"ok"} when the DB is reachable within 500ms.
// Returns 503 {"status":"degraded","reason":"database"} when conn is nil or the ping fails.
func healthHandler(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if conn == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"degraded","reason":"database"}`)) //nolint:errcheck
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		if err := conn.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"degraded","reason":"database"}`)) //nolint:errcheck
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	}
}

func runMigrations(dsn string) error {
	src, err := iofs.New(dbmigrations.FS, "migrations")
	if err != nil {
		return fmt.Errorf("migrations source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer m.Close() //nolint:errcheck
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	log.Println("migrations applied")
	return nil
}
