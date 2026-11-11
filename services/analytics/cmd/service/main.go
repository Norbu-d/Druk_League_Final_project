package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	analyticsv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/analytics/v1"
	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/analytics/internal/adapters/grpc"
	"github.com/drukleague/platform/services/analytics/internal/adapters/grpcclients"
	natsadapter "github.com/drukleague/platform/services/analytics/internal/adapters/nats"
	"github.com/drukleague/platform/services/analytics/internal/adapters/postgres"
	"github.com/drukleague/platform/services/analytics/internal/app"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
)

func main() {
	dial := func(key, fallback string) (*grpc.ClientConn, error) {
		return servicekit.Dial(servicekit.Env(key, fallback), servicekit.ClientOptions{
			Auth: servicekit.ServiceIdentity, Tokens: servicekit.NewTokenSource("analytics-service", servicekit.Secret(), 0), Timeout: 5 * time.Second,
		})
	}
	tournamentConn, err := dial("TOURNAMENT_SERVICE_ADDR", "tournament:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer tournamentConn.Close()
	teamConn, err := dial("TEAM_SERVICE_ADDR", "team:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer teamConn.Close()
	matchConn, err := dial("MATCH_SERVICE_ADDR", "match:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer matchConn.Close()
	playerConn, err := dial("PLAYER_SERVICE_ADDR", "player:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer playerConn.Close()
	registrationConn, err := dial("REGISTRATION_SERVICE_ADDR", "registration:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer registrationConn.Close()
	tournaments := grpcclients.NewTournaments(tournamentConn)
	dashboard := grpcclients.NewDashboard(tournamentConn, teamConn, matchConn, playerConn, registrationConn)

	consumerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var natsConn *nats.Conn
	err = servicekit.Run("analytics-service", func(server *grpc.Server, db *sql.DB) error {
		service := app.New(postgres.New(db), tournaments, dashboard)
		analyticsv1.RegisterAnalyticsServiceServer(server, grpcadapter.New(service))
		var js nats.JetStreamContext
		natsConn, js, err = eventkit.Connect(servicekit.Env("NATS_URL", "nats://nats:4222"), "drukleague-analytics")
		if err != nil {
			return err
		}
		if err := eventkit.EnsureStreams(js); err != nil {
			return err
		}
		return eventkit.StartConsumer(consumerCtx, js, natsadapter.Consumer(service))
	})
	cancel()
	if natsConn != nil {
		natsConn.Close()
	}
	if err != nil {
		log.Fatal(err)
	}
}
