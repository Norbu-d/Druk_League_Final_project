package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/drukleague/platform/packages/servicekit/servicekit"
	mediahttp "github.com/drukleague/platform/services/media/internal/adapters/http"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	for {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = db.PingContext(pingCtx)
		cancel()
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			log.Fatal("database was not ready before shutdown")
		}
		time.Sleep(time.Second)
	}
	if err := servicekit.ApplyMigrations(ctx, db, env("MIGRATIONS_DIR", "migrations")); err != nil {
		log.Fatal(err)
	}
	addr := env("MEDIA_HTTP_ADDR", ":8080")
	server := &http.Server{Addr: addr, Handler: mediahttp.New(db, []byte(servicekit.Secret()), env("MEDIA_DATA_DIR", "/data")).Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	errCh := make(chan error, 1)
	go func() { errCh <- server.ListenAndServe() }()
	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("media HTTP shutdown: %v", err)
		}
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
