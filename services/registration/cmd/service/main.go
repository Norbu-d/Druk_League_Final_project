package main

import (
	"database/sql"
	"log"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/registration/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/registration/internal/adapters/grpc"
	"github.com/drukleague/platform/services/registration/internal/adapters/grpcclients"
	"github.com/drukleague/platform/services/registration/internal/adapters/postgres"
	"github.com/drukleague/platform/services/registration/internal/app"
	"google.golang.org/grpc"
)

func main() {
	// Existence checks are lookups, not actions on behalf of the caller, so they use this
	// service's own self-renewing token. The caller's role is checked in the app layer.
	options := func() servicekit.ClientOptions {
		return servicekit.ClientOptions{Auth: servicekit.ServiceIdentity, Tokens: servicekit.NewTokenSource("registration-service", servicekit.Secret(), 0)}
	}
	teamConn, err := servicekit.Dial(servicekit.Env("TEAM_SERVICE_ADDR", "team:50051"), options())
	if err != nil {
		log.Fatal(err)
	}
	defer teamConn.Close()
	tournamentConn, err := servicekit.Dial(servicekit.Env("TOURNAMENT_SERVICE_ADDR", "tournament:50051"), options())
	if err != nil {
		log.Fatal(err)
	}
	defer tournamentConn.Close()
	teams, tournaments := grpcclients.NewTeams(teamConn), grpcclients.NewTournaments(tournamentConn)

	register := func(server *grpc.Server, db *sql.DB) error {
		v1.RegisterRegistrationServiceServer(server, grpcadapter.New(app.New(postgres.New(db), teams, tournaments)))
		return nil
	}
	if err := servicekit.Run("registration-service", register); err != nil {
		log.Fatal(err)
	}
}
