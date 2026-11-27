package main

import (
	"context"
	"database/sql"
	"log"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/search/v1"
	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/search/internal/adapters/grpc"
	natsadapter "github.com/drukleague/platform/services/search/internal/adapters/nats"
	"github.com/drukleague/platform/services/search/internal/adapters/postgres"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
)

func main() {
	servicekit.LoadConfiguration("search-service",
		"GRPC_ADDR",
		"METRICS_ADDR",
		"NATS_URL",
	)

	consumerCtx, cancel := context.WithCancel(context.Background())
	var natsConn *nats.Conn
	register := func(server *grpc.Server, db *sql.DB) error {
		repository := postgres.New(db)
		v1.RegisterSearchServiceServer(server, grpcadapter.New(repository))
		var js nats.JetStreamContext
		var err error
		natsConn, js, err = eventkit.Connect(servicekit.Env("NATS_URL", "nats://nats:4222"), "drukleague-search")
		if err != nil {
			return err
		}
		if err := eventkit.EnsureStreams(js); err != nil {
			return err
		}
		for _, consumer := range natsadapter.Consumers(repository) {
			if err := eventkit.StartConsumer(consumerCtx, js, consumer); err != nil {
				return err
			}
		}
		return nil
	}
	err := servicekit.Run("search-service", register)
	cancel()
	if natsConn != nil {
		natsConn.Close()
	}
	if err != nil {
		log.Fatal(err)
	}
}
