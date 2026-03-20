package main

import (
	"context"
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

	dbmigrations "github.com/karo/cuttlegate/db"
	httpadapter "github.com/karo/cuttlegate/internal/adapters/http"
	"github.com/karo/cuttlegate/internal/domain/ports"
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

	// ── Adapters ──────────────────────────────────────────────────────────────

	verifier, err := httpadapter.NewOIDCVerifier(
		context.Background(),
		cfg.OIDCIssuer,
		cfg.OIDCAudience,
		cfg.OIDCRoleClaim,
	)
	if err != nil {
		return fmt.Errorf("oidc: %w", err)
	}

	// DB connection and repository adapters are wired here once dbadapter
	// implementations land (Sprint 2). Example shape:
	//
	//   db, err := sql.Open("pgx", cfg.DSN)
	//   if err != nil { return fmt.Errorf("db: %w", err) }
	//   var flags    ports.FlagRepository        = dbadapter.NewFlagRepository(db)
	//   var projects ports.ProjectRepository     = dbadapter.NewProjectRepository(db)
	//   var envs     ports.EnvironmentRepository = dbadapter.NewPostgresEnvironmentRepository(db)
	//   var events   ports.EventPublisher        = pubsub.NewEventPublisher()

	// ── Use cases ─────────────────────────────────────────────────────────────

	// Use-case constructors receive repository and publisher ports and are
	// registered here before being passed to HTTP handlers (Sprint 2). Example:
	//
	//   flagUC := app.NewFlagUseCase(flags, events)

	// ── HTTP ──────────────────────────────────────────────────────────────────

	mux := buildMux(verifier)

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
	}

	// ── Startup and graceful shutdown ─────────────────────────────────────────

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

func buildMux(verifier ports.TokenVerifier) *http.ServeMux {
	mux := http.NewServeMux()
	requireBearer := httpadapter.RequireBearer(verifier)

	// ── Application routes (Sprint 2) ─────────────────────────────────────────
	//
	// All API routes are Bearer-authenticated. Example shape:
	//
	//   mux.Handle("GET /api/projects", requireBearer(flagHandler.ListProjects))
	//   mux.Handle("POST /api/projects", requireBearer(flagHandler.CreateProject))

	// Placeholder — replaced by real routes in Sprint 2.
	mux.Handle("/", requireBearer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	return mux
}
