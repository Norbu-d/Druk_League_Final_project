package servicekit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Run starts one independently deployable service with its own database and gRPC health endpoint.
func Run(serviceName string, register ...func(*grpc.Server, *sql.DB) error) error {
	if serviceName == "" {
		return fmt.Errorf("service name is required")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	addr := os.Getenv("GRPC_ADDR")
	if addr == "" {
		addr = ":50051"
	}
	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()
	pingCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("connect to service database: %w", err)
	}
	if err := ApplyMigrations(ctx, db, migrationsDir); err != nil {
		return fmt.Errorf("apply service migrations: %w", err)
	}
	metricsAddr := os.Getenv("METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = ":9090"
	}
	metricsListener, err := net.Listen("tcp", metricsAddr)
	if err != nil {
		return fmt.Errorf("listen for Prometheus metrics on %s: %w", metricsAddr, err)
	}
	defer metricsListener.Close()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	metricsInterceptor, metricsHandler := NewPrometheusMetrics(serviceName)
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", metricsHandler)
	metricsServer := &http.Server{Handler: metricsMux, ReadHeaderTimeout: 5 * time.Second}
	// Metrics wraps recovery so panics are recorded with their final gRPC status code.
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		metricsInterceptor,
		RecoveryUnaryInterceptor(),
		LoggingUnaryInterceptor(serviceName),
		AuthUnaryInterceptor(Secret()),
	))
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(serviceName, healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	for _, configure := range register {
		if err := configure(server, db); err != nil {
			_ = listener.Close()
			return fmt.Errorf("register gRPC service: %w", err)
		}
	}
	serveErr := make(chan error, 2)
	go func() {
		if err := metricsServer.Serve(metricsListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- fmt.Errorf("serve Prometheus metrics: %w", err)
		}
	}()
	go func() {
		if err := server.Serve(listener); err != nil {
			serveErr <- fmt.Errorf("serve gRPC: %w", err)
		}
	}()
	select {
	case err := <-serveErr:
		server.Stop()
		_ = metricsServer.Close()
		return err
	case <-ctx.Done():
		healthServer.Shutdown()
		server.GracefulStop()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shut down Prometheus metrics server: %w", err)
		}
		return nil
	}
}

// ApplyMigrations runs every *.sql file in dir in name order, each in its own transaction.
// Migrations must be idempotent (CREATE ... IF NOT EXISTS) because they run on every start.
func ApplyMigrations(ctx context.Context, db *sql.DB, dir string) error {
	root, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(root, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, name := range files {
		// Glob restricts migration reads to SQL files directly inside the configured directory.
		contents, err := os.ReadFile(name) // #nosec G304,G703 -- name is returned by the constrained *.sql glob above.
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		// Migration files contain executable schema SQL by design and are deployment-controlled.
		if _, err = tx.ExecContext(ctx, string(contents)); err != nil { // #nosec G701 -- SQL comes from deployment-controlled migration files.
			_ = tx.Rollback()
			return fmt.Errorf("execute %s: %w", name, err)
		}
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("commit %s: %w", name, err)
		}
	}
	return nil
}
