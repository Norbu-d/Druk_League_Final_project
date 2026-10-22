package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"

	drukleaguev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/match/internal/adapters/grpc"
	"github.com/drukleague/platform/services/match/internal/adapters/grpcclients"
	natsadapter "github.com/drukleague/platform/services/match/internal/adapters/nats"
	"github.com/drukleague/platform/services/match/internal/adapters/postgres"
	"github.com/drukleague/platform/services/match/internal/app"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
)

func main() {
	// Forward the caller's token for user-driven calls; fall back to the service identity.
	options := servicekit.ClientOptions{Auth: servicekit.ForwardOrService, Tokens: servicekit.NewTokenSource("match-service", servicekit.Secret(), 0)}
	teamConn, err := servicekit.Dial(servicekit.Env("TEAM_SERVICE_ADDR", "team:50051"), options)
	if err != nil {
		log.Fatal(fmt.Errorf("configure Team client: %w", err))
	}
	defer teamConn.Close()
	userConn, err := servicekit.Dial(servicekit.Env("USER_SERVICE_ADDR", "user:50051"), options)
	if err != nil {
		log.Fatal(fmt.Errorf("configure User client: %w", err))
	}
	defer userConn.Close()
	teams := grpcclients.NewTeamDirectory(teamConn)
	users := grpcclients.NewUserDirectory(userConn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var once sync.Once
	var natsConn *nats.Conn
	var natsErr error
	register := func(server *grpc.Server, db *sql.DB) error {
		service := app.New(postgres.New(db), teams, users)
		once.Do(func() {
			var js nats.JetStreamContext
			natsConn, js, natsErr = eventkit.Connect(servicekit.Env("NATS_URL", "nats://nats:4222"), "drukleague-match")
			if natsErr == nil {
				natsErr = eventkit.EnsureStreams(js)
			}
			if natsErr == nil {
				natsErr = eventkit.StartConsumer(ctx, js, natsadapter.Consumer(service))
			}
		})
		if natsErr != nil {
			return natsErr
		}
		drukleaguev1.RegisterMatchServiceServer(server, grpcadapter.New(service))
		return nil
	}
	if err := servicekit.Run("match-service", register); err != nil {
		log.Fatal(err)
	}
	if natsConn != nil {
		natsConn.Close()
	}
}
