package main

import (
	"database/sql"
	"log"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/registration/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/registration/internal/adapters/grpc"
	"google.golang.org/grpc"
)

func main() {
	register := func(server *grpc.Server, db *sql.DB) error {
		v1.RegisterRegistrationServiceServer(server, grpcadapter.New(db))
		return nil
	}
	if err := servicekit.Run("registration-service", register); err != nil {
		log.Fatal(err)
	}
}
