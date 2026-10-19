package main

import (
	"database/sql"
	"log"
	"time"

	authv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/auth/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/auth/internal/adapters/grpc"
	"github.com/drukleague/platform/services/auth/internal/adapters/grpcclients"
	"github.com/drukleague/platform/services/auth/internal/adapters/postgres"
	"github.com/drukleague/platform/services/auth/internal/app"
	"google.golang.org/grpc"
)

func duration(name string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(servicekit.Env(name, ""))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func main() {
	userConn, err := servicekit.Dial(servicekit.Env("USER_SERVICE_ADDR", "user:50051"), servicekit.ClientOptions{Auth: servicekit.ServiceIdentity, Tokens: servicekit.NewTokenSource("auth-service", servicekit.Secret(), 0)})
	if err != nil {
		log.Fatal(err)
	}
	defer userConn.Close()
	users := grpcclients.NewUsers(userConn)
	signer := grpcclients.NewSigner(servicekit.Secret())
	config := app.Config{AccessTTL: duration("ACCESS_TOKEN_TTL", 30*time.Minute), RefreshTTL: duration("REFRESH_TOKEN_TTL", 7*24*time.Hour)}
	register := func(server *grpc.Server, db *sql.DB) error {
		service := app.New(users, postgres.New(db), postgres.NewAttempts(db), signer, config)
		authv1.RegisterAuthServiceServer(server, grpcadapter.New(service))
		return nil
	}
	if err := servicekit.Run("auth-service", register); err != nil {
		log.Fatal(err)
	}
}
