package main

import (
	"database/sql"
	"log"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/search/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/search/internal/adapters/grpc"
	"github.com/drukleague/platform/services/search/internal/adapters/postgres"
	"google.golang.org/grpc"
)

func main() {
	servicekit.LoadConfiguration("search-service",
		"GRPC_ADDR",
		"METRICS_ADDR",
	)
	register := func(server *grpc.Server, db *sql.DB) error {
		v1.RegisterSearchServiceServer(server, grpcadapter.New(postgres.New(db)))
		return nil
	}
	if err := servicekit.Run("search-service", register); err != nil {
		log.Fatal(err)
	}
}
