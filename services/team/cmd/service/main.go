package main

import (
	"database/sql"
	"log"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/dzongkhags"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/team/internal/adapters/grpc"
	"github.com/drukleague/platform/services/team/internal/adapters/grpcclients"
	"github.com/drukleague/platform/services/team/internal/adapters/postgres"
	"github.com/drukleague/platform/services/team/internal/app"
	"google.golang.org/grpc"
)

func main() {
	// Reference checks are lookups, not actions on behalf of the caller, so they use this
	// service's own self-renewing token. The caller's role is checked in the app layer.
	options := func() servicekit.ClientOptions {
		return servicekit.ClientOptions{Auth: servicekit.ServiceIdentity, Tokens: servicekit.NewTokenSource("team-service", servicekit.Secret(), 0)}
	}
	tournamentConn, err := servicekit.Dial(servicekit.Env("TOURNAMENT_SERVICE_ADDR", "tournament:50051"), options())
	if err != nil {
		log.Fatal(err)
	}
	defer tournamentConn.Close()
	dzongkhagConn, err := servicekit.Dial(servicekit.Env("DZONGKHAG_SERVICE_ADDR", "dzongkhag:50051"), options())
	if err != nil {
		log.Fatal(err)
	}
	defer dzongkhagConn.Close()
	tournaments, directory := grpcclients.NewTournaments(tournamentConn), dzongkhags.New(dzongkhagConn, 0)
	register := func(server *grpc.Server, db *sql.DB) error {
		v1.RegisterTeamServiceServer(server, grpcadapter.New(app.New(postgres.New(db), tournaments, directory)))
		return nil
	}
	if err := servicekit.Run("team-service", register); err != nil {
		log.Fatal(err)
	}
}
