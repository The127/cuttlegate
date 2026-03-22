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

	dbmigrations "github.com/karo/cuttlegate/db"
	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
	httpadapter "github.com/karo/cuttlegate/internal/adapters/http"
	"github.com/karo/cuttlegate/internal/app"
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
		uowFactory := dbadapter.NewPostgresUnitOfWorkFactory(conn)

		requireBearer := httpadapter.RequireBearer(verifier, userRepo)

		projSvc := app.NewProjectService(projRepo)
		envSvc := app.NewEnvironmentService(envRepo, projRepo)
		memberSvc := app.NewProjectMemberService(memberRepo, projRepo, userRepo)
		flagSvc := app.NewFlagService(flagRepo, envRepo, stateRepo, broker, auditRepo)
		ruleSvc := app.NewRuleService(ruleRepo)
		segmentSvc := app.NewSegmentService(segmentRepo)
		evalSvc := app.NewEvaluationService(flagRepo, stateRepo, ruleRepo, segmentRepo)
		apiKeySvc := app.NewAPIKeyService(apiKeyRepo)
		auditSvc := app.NewAuditService(auditRepo)
		promotionSvc := app.NewPromotionService(uowFactory, flagRepo)

		httpadapter.NewProjectHandler(projSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewEnvironmentHandler(envSvc, projSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewProjectMemberHandler(memberSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewFlagHandler(flagSvc, projSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewFlagVariantHandler(flagSvc, projSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewFlagEnvironmentHandler(flagSvc, projSvc, envSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewRuleHandler(ruleSvc, projSvc, flagSvc, envSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewSegmentHandler(segmentSvc, projSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewAPIKeyHandler(apiKeySvc, projSvc, envSvc).RegisterRoutes(mux, requireBearer)
		evalRateLimiter := httpadapter.NewRateLimiter(cfg.EvalRateLimit, cfg.EvalRateLimitWindow)
		requireBearerOrAPIKey := httpadapter.RequireBearerOrAPIKey(verifier, apiKeySvc)
		evalAuth := func(h http.Handler) http.Handler { return requireBearerOrAPIKey(evalRateLimiter.Limit(h)) }
		httpadapter.NewEvaluationHandler(evalSvc, projSvc, envSvc).RegisterRoutes(mux, evalAuth)

		httpadapter.NewSSEHandler(broker, projSvc, envSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewAuditHandler(auditSvc, projSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewPromotionHandler(promotionSvc, projSvc, envSvc).RegisterRoutes(mux, requireBearer)
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

	// SPA static files — registered last so /api/v1/* routes take precedence.
	serveSPA(mux)

	// ── Server ────────────────────────────────────────────────────────────────

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
