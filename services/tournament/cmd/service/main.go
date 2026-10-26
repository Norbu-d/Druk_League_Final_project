package main

import (
	"database/sql"
	"log"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/dzongkhags"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/tournament/internal/adapters/grpc"
	"github.com/drukleague/platform/services/tournament/internal/adapters/postgres"
	"github.com/drukleague/platform/services/tournament/internal/app"
	"google.golang.org/grpc"
)

func main() {
	dzongkhagConn, err := servicekit.Dial(servicekit.Env("DZONGKHAG_SERVICE_ADDR", "dzongkhag:50051"), servicekit.ClientOptions{Auth: servicekit.ServiceIdentity, Tokens: servicekit.NewTokenSource("tournament-service", servicekit.Secret(), 0)})
	if err != nil {
		log.Fatal(err)
	}
	defer dzongkhagConn.Close()
	directory := dzongkhags.New(dzongkhagConn, 0)
	register := func(server *grpc.Server, db *sql.DB) error {
		v1.RegisterTournamentServiceServer(server, grpcadapter.New(app.New(postgres.New(db), directory)))
		return nil
	}
	if err := servicekit.Run("tournament-service", register); err != nil {
		log.Fatal(err)
	}
}
