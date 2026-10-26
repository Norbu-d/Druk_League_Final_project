package main

import (
	"database/sql"
	"log"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/player/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/player/internal/adapters/grpc"
	"github.com/drukleague/platform/services/player/internal/adapters/grpcclients"
	"github.com/drukleague/platform/services/player/internal/adapters/postgres"
	"github.com/drukleague/platform/services/player/internal/app"
	"google.golang.org/grpc"
)

func main() {
	// Team lookups are not actions on behalf of the caller, so they use this service's own
	// self-renewing token. The caller's role and team ownership are checked in the app layer.
	teamConn, err := servicekit.Dial(servicekit.Env("TEAM_SERVICE_ADDR", "team:50051"), servicekit.ClientOptions{Auth: servicekit.ServiceIdentity, Tokens: servicekit.NewTokenSource("player-service", servicekit.Secret(), 0)})
	if err != nil {
		log.Fatal(err)
	}
	defer teamConn.Close()
	teams := grpcclients.NewTeams(teamConn)
	register := func(server *grpc.Server, db *sql.DB) error {
		v1.RegisterPlayerServiceServer(server, grpcadapter.New(app.New(postgres.New(db), teams)))
		return nil
	}
	if err := servicekit.Run("player-service", register); err != nil {
		log.Fatal(err)
	}
}
