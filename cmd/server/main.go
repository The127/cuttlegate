package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	)
	if err != nil {
		return fmt.Errorf("oidc: %w", err)
	}

	// ── HTTP mux ──────────────────────────────────────────────────────────────

	mux := http.NewServeMux()
	requireBearer := httpadapter.RequireBearer(verifier)

	// Public: SPA OIDC config — no auth required.
	clientCfg := spaClientConfig{
		Authority:   cfg.OIDCIssuer,
		ClientID:    cfg.OIDCClientID,
		RedirectURI: cfg.OIDCRedirectURI,
	}
	mux.HandleFunc("GET /api/v1/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clientCfg) //nolint:errcheck
	})

	// Authenticated API routes — wired when DATABASE_URL is available.
	if cfg.DSN != "" {
		conn, err := sql.Open("postgres", cfg.DSN)
		if err != nil {
			return fmt.Errorf("db: %w", err)
		}
		defer conn.Close() //nolint:errcheck

		projRepo := dbadapter.NewPostgresProjectRepository(conn)
		envRepo := dbadapter.NewPostgresEnvironmentRepository(conn)
		memberRepo := dbadapter.NewPostgresProjectMemberRepository(conn)
		flagRepo := dbadapter.NewPostgresFlagRepository(conn)
		stateRepo := dbadapter.NewPostgresFlagEnvironmentStateRepository(conn)
		ruleRepo := &dbadapter.NoOpRuleRepository{} // TODO: replace with postgres adapter once migration exists

		projSvc := app.NewProjectService(projRepo)
		envSvc := app.NewEnvironmentService(envRepo, projRepo)
		memberSvc := app.NewProjectMemberService(memberRepo, projRepo)
		flagSvc := app.NewFlagService(flagRepo, envRepo, stateRepo)
		evalSvc := app.NewEvaluationService(flagRepo, stateRepo, ruleRepo)

		httpadapter.NewProjectHandler(projSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewEnvironmentHandler(envSvc, projSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewProjectMemberHandler(memberSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewFlagHandler(flagSvc, projSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewFlagVariantHandler(flagSvc, projSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewFlagEnvironmentHandler(flagSvc, projSvc, envSvc).RegisterRoutes(mux, requireBearer)
		httpadapter.NewEvaluationHandler(evalSvc, projSvc, envSvc).RegisterRoutes(mux, requireBearer)
	}

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

	return srv.Shutdown(shutdownCtx)
}

// spaClientConfig is the public OIDC config returned to the SPA.
// Only values safe to expose to the browser are included.
type spaClientConfig struct {
	Authority   string `json:"authority"`
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
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
