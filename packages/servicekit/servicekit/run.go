package servicekit

import (
	"context"
	"database/sql"
	"fmt"
	"net"
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
	if err := applyMigrations(ctx, db, migrationsDir); err != nil {
		return fmt.Errorf("apply service migrations: %w", err)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "local-development-secret-change-before-deployment"
	}
	server := grpc.NewServer(grpc.UnaryInterceptor(AuthUnaryInterceptor([]byte(secret))))
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
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	select {
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("serve gRPC: %w", err)
		}
		return nil
	case <-ctx.Done():
		server.GracefulStop()
		return nil
	}
}

func applyMigrations(ctx context.Context, db *sql.DB, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, name := range files {
		contents, err := os.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(contents)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute %s: %w", name, err)
		}
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("commit %s: %w", name, err)
		}
	}
	return nil
}
