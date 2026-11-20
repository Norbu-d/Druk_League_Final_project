package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/dzongkhags"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/tournament/internal/adapters/grpc"
	"github.com/drukleague/platform/services/tournament/internal/adapters/postgres"
	"github.com/drukleague/platform/services/tournament/internal/app"
	"google.golang.org/grpc"
)

func main() {
	servicekit.LoadConfiguration("tournament-service",
		"GRPC_ADDR",
		"METRICS_ADDR",
		"DZONGKHAG_SERVICE_ADDR",
		"NATS_URL",
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var natsConn interface{ Close() }
	dzongkhagConn, err := servicekit.Dial(servicekit.Env("DZONGKHAG_SERVICE_ADDR", "dzongkhag:50051"), servicekit.ClientOptions{Auth: servicekit.ServiceIdentity, Tokens: servicekit.NewTokenSource("tournament-service", servicekit.Secret(), 0)})
	if err != nil {
		log.Fatal(err)
	}
	defer dzongkhagConn.Close()
	directory := dzongkhags.New(dzongkhagConn, 0)
	register := func(server *grpc.Server, db *sql.DB) error {
		v1.RegisterTournamentServiceServer(server, grpcadapter.New(app.New(postgres.New(db), directory)))
		conn, js, err := eventkit.Connect(servicekit.Env("NATS_URL", "nats://nats:4222"), "drukleague-tournament")
		if err != nil {
			return err
		}
		natsConn = conn
		if err := eventkit.EnsureStreams(js); err != nil {
			return err
		}
		store, err := eventkit.NewSQLOutboxStore(db)
		if err != nil {
			return err
		}
		go eventkit.RunOutboxRelay(ctx, store, js, time.Second)
		return nil
	}
	if err := servicekit.Run("tournament-service", register); err != nil {
		log.Fatal(err)
	}
	if natsConn != nil {
		natsConn.Close()
	}
}
