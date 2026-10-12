package main

import (
	"database/sql"
	"log"

	dzongkhagv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/dzongkhag/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/dzongkhag/internal/adapters/grpc"
	"github.com/drukleague/platform/services/dzongkhag/internal/adapters/postgres"
	"github.com/drukleague/platform/services/dzongkhag/internal/app"
	"google.golang.org/grpc"
)

func main() {
	register := func(server *grpc.Server, db *sql.DB) error {
		repository := postgres.New(db)
		dzongkhagv1.RegisterDzongkhagServiceServer(server, grpcadapter.New(app.New(repository)))
		return nil
	}
	if err := servicekit.Run("dzongkhag-service", register); err != nil {
		log.Fatal(err)
	}
}
