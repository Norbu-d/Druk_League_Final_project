package main

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	fixturev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/fixture/v1"
	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/fixture/internal/adapters/grpc"
	"github.com/drukleague/platform/services/fixture/internal/adapters/grpcclients"
	natsadapter "github.com/drukleague/platform/services/fixture/internal/adapters/nats"
	"github.com/drukleague/platform/services/fixture/internal/adapters/postgres"
	"github.com/drukleague/platform/services/fixture/internal/app"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
)

func dial(name, fallback string) *grpc.ClientConn {
	// Generation is an admin action; the admin's token is forwarded so callees apply their own rules.
	conn, err := servicekit.Dial(servicekit.Env(name, fallback), servicekit.ClientOptions{Auth: servicekit.ForwardCaller, Timeout: 5 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	return conn
}

func main() {
	tournamentConn := dial("TOURNAMENT_SERVICE_ADDR", "tournament:50051")
	teamConn := dial("TEAM_SERVICE_ADDR", "team:50051")
	venueConn := dial("VENUE_SERVICE_ADDR", "venue:50051")
	matchConn := dial("MATCH_SERVICE_ADDR", "match:50051")
	for _, conn := range []*grpc.ClientConn{tournamentConn, teamConn, venueConn, matchConn} {
		defer conn.Close()
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var once sync.Once
	var natsConn *nats.Conn
	var natsErr error
	register := func(server *grpc.Server, db *sql.DB) error {
		repository := postgres.New(db)
		once.Do(func() {
			var js nats.JetStreamContext
			natsConn, js, natsErr = eventkit.Connect(servicekit.Env("NATS_URL", "nats://nats:4222"), "drukleague-fixture")
			if natsErr == nil {
				natsErr = eventkit.EnsureStreams(js)
			}
			if natsErr == nil {
				go natsadapter.RunRelay(ctx, repository, js, time.Second)
			}
		})
		if natsErr != nil {
			return natsErr
		}
		service := app.New(repository, grpcclients.NewTournaments(tournamentConn), grpcclients.NewTeams(teamConn), grpcclients.NewVenues(venueConn), grpcclients.NewMatches(matchConn))
		fixturev1.RegisterFixtureServiceServer(server, grpcadapter.New(service))
		return nil
	}
	if err := servicekit.Run("fixture-service", register); err != nil {
		log.Fatal(err)
	}
	if natsConn != nil {
		natsConn.Close()
	}
}
