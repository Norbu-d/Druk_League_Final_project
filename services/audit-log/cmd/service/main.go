package main

import (
	"context"
	"database/sql"
	"log"

	auditlogv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/audit_log/v1"
	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/audit-log/internal/adapters/grpc"
	natsadapter "github.com/drukleague/platform/services/audit-log/internal/adapters/nats"
	"github.com/drukleague/platform/services/audit-log/internal/adapters/postgres"
	"github.com/drukleague/platform/services/audit-log/internal/app"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
)

func main() {
	consumerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var natsConn *nats.Conn
	err := servicekit.Run("audit-log-service", func(server *grpc.Server, db *sql.DB) error {
		repository := postgres.New(db)
		service := app.New(repository)
		var js nats.JetStreamContext
		var err error
		natsConn, js, err = eventkit.Connect(servicekit.Env("NATS_URL", "nats://nats:4222"), "drukleague-audit-log")
		if err != nil {
			return err
		}
		if err := eventkit.EnsureStreams(js); err != nil {
			return err
		}
		for _, config := range []eventkit.ConsumerConfig{natsadapter.MatchFinalizedConsumer(service), natsadapter.AdminActionsConsumer(service)} {
			if err := eventkit.StartConsumer(consumerCtx, js, config); err != nil {
				return err
			}
		}
		auditlogv1.RegisterAuditLogServiceServer(server, grpcadapter.New(service))
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
