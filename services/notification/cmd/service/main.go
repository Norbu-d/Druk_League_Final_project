package main

import (
	"context"
	"database/sql"
	"log"
	"sync"

	notificationv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/notification/v1"
	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/notification/internal/adapters/grpc"
	"github.com/drukleague/platform/services/notification/internal/adapters/grpcclients"
	natsadapter "github.com/drukleague/platform/services/notification/internal/adapters/nats"
	"github.com/drukleague/platform/services/notification/internal/adapters/postgres"
	"github.com/drukleague/platform/services/notification/internal/app"
	"google.golang.org/grpc"
)

func main() {
	teamConn, err := servicekit.Dial(servicekit.Env("TEAM_SERVICE_ADDR", "team:50051"), servicekit.ClientOptions{Auth: servicekit.ForwardOrService, Tokens: servicekit.NewTokenSource("notification-service", servicekit.Secret(), 0)})
	if err != nil {
		log.Fatal(err)
	}
	defer teamConn.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var once sync.Once
	var closeNATS func()
	var natsErr error

	register := func(server *grpc.Server, db *sql.DB) error {
		service := app.New(postgres.New(db), grpcclients.NewTeams(teamConn))
		once.Do(func() {
			conn, js, err := eventkit.Connect(
				servicekit.Env("NATS_URL", "nats://nats:4222"),
				"drukleague-notification",
			)
			natsErr = err
			if err == nil {
				closeNATS = conn.Close
				natsErr = eventkit.EnsureStreams(js)
			}
			if natsErr == nil {
				natsErr = eventkit.StartConsumer(ctx, js, natsadapter.Consumer(service))
			}
		})
		if natsErr != nil {
			return natsErr
		}

		notificationv1.RegisterNotificationServiceServer(server, grpcadapter.New(service))
		return nil
	}

	if err := servicekit.Run("notification-service", register); err != nil {
		log.Fatal(err)
	}
	if closeNATS != nil {
		closeNATS()
	}
}
