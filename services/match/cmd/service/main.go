package main

import (
	"database/sql"
	"log"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/match/internal/adapters/grpc"
	"google.golang.org/grpc"
)

func main() {
	register := func(server *grpc.Server, db *sql.DB) error {
		v1.RegisterMatchServiceServer(server, grpcadapter.New(db))
		return nil
	}
	if err := servicekit.Run("match-service", register); err != nil {
		log.Fatal(err)
	}
}
