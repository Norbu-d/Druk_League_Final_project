package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	notificationv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/notification/v1"
	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/notification/internal/adapters/grpc"
	"github.com/drukleague/platform/services/notification/internal/adapters/grpcclients"
	natsadapter "github.com/drukleague/platform/services/notification/internal/adapters/nats"
	"github.com/drukleague/platform/services/notification/internal/adapters/postgres"
	"github.com/drukleague/platform/services/notification/internal/app"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
)

func main() {
	teamConn, err := servicekit.Dial(servicekit.Env("TEAM_SERVICE_ADDR", "team:50051"), servicekit.ClientOptions{
		Auth: servicekit.ServiceIdentity, Tokens: servicekit.NewTokenSource("notification-service", servicekit.Secret(), 0), Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer teamConn.Close()
	teams := grpcclients.NewTeamDirectory(teamConn)

	consumerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var natsConn *nats.Conn
	err = servicekit.Run("notification-service", func(server *grpc.Server, db *sql.DB) error {
		repository := postgres.New(db)
		service := app.New(repository, teams)
		var js nats.JetStreamContext
		natsConn, js, err = eventkit.Connect(servicekit.Env("NATS_URL", "nats://nats:4222"), "drukleague-notification")
		if err != nil {
			return err
		}
		if err := eventkit.EnsureStreams(js); err != nil {
			return err
		}
		if err := eventkit.StartConsumer(consumerCtx, js, natsadapter.MatchFinalizedConsumer(service)); err != nil {
			return err
		}
		notificationv1.RegisterNotificationServiceServer(server, grpcadapter.New(service))
		return nil
	})
	cancel()
	if natsConn != nil {
		natsConn.Close()
	}
	if err != nil {
		log.Fatal(err)
	}
}
