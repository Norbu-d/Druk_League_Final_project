package main

import (
	"context"
	"database/sql"
	"log"
	"sync"

	standingsv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/standings/v1"
	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/standings/internal/adapters/grpc"
	"github.com/drukleague/platform/services/standings/internal/adapters/grpcclients"
	natsadapter "github.com/drukleague/platform/services/standings/internal/adapters/nats"
	"github.com/drukleague/platform/services/standings/internal/adapters/postgres"
	"github.com/drukleague/platform/services/standings/internal/app"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
)

func main() {
	// The consumer has no caller token, so lookups always use a self-renewing service token.
	teamConn, err := servicekit.Dial(servicekit.Env("TEAM_SERVICE_ADDR", "team:50051"), servicekit.ClientOptions{Auth: servicekit.ServiceIdentity, Tokens: servicekit.NewTokenSource("standings-service", servicekit.Secret(), 0)})
	if err != nil {
		log.Fatal(err)
	}
	defer teamConn.Close()
	teams := grpcclients.NewTeamDirectory(teamConn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var once sync.Once
	var natsConn *nats.Conn
	var natsErr error
	register := func(server *grpc.Server, db *sql.DB) error {
		service := app.New(postgres.New(db), teams)
		once.Do(func() {
			var js nats.JetStreamContext
			natsConn, js, natsErr = eventkit.Connect(servicekit.Env("NATS_URL", "nats://nats:4222"), "drukleague-standings")
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
		standingsv1.RegisterStandingsServiceServer(server, grpcadapter.New(service))
		return nil
	}
	if err := servicekit.Run("standings-service", register); err != nil {
		log.Fatal(err)
	}
	if natsConn != nil {
		natsConn.Close()
	}
}
