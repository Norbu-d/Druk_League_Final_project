package main

import (
	"database/sql"
	"log"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/official/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/official/internal/adapters/grpc"
	"github.com/drukleague/platform/services/official/internal/adapters/grpcclients"
	"github.com/drukleague/platform/services/official/internal/adapters/postgres"
	"github.com/drukleague/platform/services/official/internal/app"
	"google.golang.org/grpc"
)

func main() {
	// Assignment is an admin action, so the admin's token is forwarded to the Match service.
	matchConn, err := servicekit.Dial(servicekit.Env("MATCH_SERVICE_ADDR", "match:50051"), servicekit.ClientOptions{Auth: servicekit.ForwardCaller})
	if err != nil {
		log.Fatal(err)
	}
	defer matchConn.Close()
	matches := grpcclients.NewMatches(matchConn)
	register := func(server *grpc.Server, db *sql.DB) error {
		v1.RegisterOfficialServiceServer(server, grpcadapter.New(app.New(postgres.New(db), matches)))
		return nil
	}
	if err := servicekit.Run("official-service", register); err != nil {
		log.Fatal(err)
	}
}
