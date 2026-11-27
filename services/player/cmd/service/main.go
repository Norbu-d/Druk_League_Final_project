package main

import (
	"context"
	"database/sql"
	"log"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/player/v1"
	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/player/internal/adapters/grpc"
	"github.com/drukleague/platform/services/player/internal/adapters/grpcclients"
	"github.com/drukleague/platform/services/player/internal/adapters/postgres"
	"github.com/drukleague/platform/services/player/internal/app"
	"google.golang.org/grpc"
)

func main() {
	servicekit.LoadConfiguration("player-service",
		"GRPC_ADDR",
		"METRICS_ADDR",
		"TEAM_SERVICE_ADDR",
		"NATS_URL",
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var natsConn interface{ Close() }
	// Team lookups are not actions on behalf of the caller, so they use this service's own
	// self-renewing token. The caller's role and team ownership are checked in the app layer.
	teamConn, err := servicekit.Dial(servicekit.Env("TEAM_SERVICE_ADDR", "team:50051"), servicekit.ClientOptions{Auth: servicekit.ServiceIdentity, Tokens: servicekit.NewTokenSource("player-service", servicekit.Secret(), 0)})
	if err != nil {
		log.Fatal(err)
	}
	defer teamConn.Close()
	teams := grpcclients.NewTeams(teamConn)
	register := func(server *grpc.Server, db *sql.DB) error {
		v1.RegisterPlayerServiceServer(server, grpcadapter.New(app.New(postgres.New(db), teams)))
		conn, err := eventkit.StartSQLOutboxRelay(ctx, db, servicekit.Env("NATS_URL", "nats://nats:4222"), "drukleague-player")
		if err != nil {
			return err
		}
		natsConn = conn
		return nil
	}
	if err := servicekit.Run("player-service", register); err != nil {
		log.Fatal(err)
	}
	if natsConn != nil {
		natsConn.Close()
	}
}
