package main

import (
	"database/sql"
	"log"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/venue/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/venue/internal/adapters/grpc"
	"google.golang.org/grpc"
)

func main() {
	register := func(server *grpc.Server, db *sql.DB) error {
		v1.RegisterVenueServiceServer(server, grpcadapter.New(db))
		return nil
	}
	if err := servicekit.Run("venue-service", register); err != nil {
		log.Fatal(err)
	}
}
