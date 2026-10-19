package main

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	drukleaguev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/score/internal/adapters/grpc"
	"github.com/drukleague/platform/services/score/internal/adapters/grpcclients"
	natsadapter "github.com/drukleague/platform/services/score/internal/adapters/nats"
	"github.com/drukleague/platform/services/score/internal/adapters/postgres"
	"github.com/drukleague/platform/services/score/internal/app"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
)

func main() {
	matchConn, err := servicekit.Dial(servicekit.Env("MATCH_SERVICE_ADDR", "match:50051"), servicekit.ClientOptions{Auth: servicekit.ForwardOrService, Tokens: servicekit.NewTokenSource("score-service", servicekit.Secret(), 0)})
	if err != nil {
		log.Fatal(err)
	}
	defer matchConn.Close()
	matches := grpcclients.NewMatchReader(matchConn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var once sync.Once
	var natsConn *nats.Conn
	var natsErr error
	register := func(server *grpc.Server, db *sql.DB) error {
		repository := postgres.New(db)
		once.Do(func() {
			var js nats.JetStreamContext
			natsConn, js, natsErr = eventkit.Connect(servicekit.Env("NATS_URL", "nats://nats:4222"), "drukleague-score")
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
		drukleaguev1.RegisterScoreServiceServer(server, grpcadapter.New(app.New(repository, matches)))
		return nil
	}
	if err := servicekit.Run("score-service", register); err != nil {
		log.Fatal(err)
	}
	if natsConn != nil {
		natsConn.Close()
	}
}
