// Command bff serves the web app's REST API by calling the gRPC microservices.
package main

import (
	"log"
	"net/http"
	"time"

	authv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/auth/v1"
	fixturev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/fixture/v1"
	officialv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/official/v1"
	standingsv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/standings/v1"
	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/bff/internal/httpapi"
	"google.golang.org/grpc"
)

func main() {
	// Every call forwards the signed-in user's token, so each service applies that user's role.
	dial := func(name, fallback string) *grpc.ClientConn {
		conn, err := servicekit.Dial(servicekit.Env(name, fallback), servicekit.ClientOptions{Auth: servicekit.ForwardCaller})
		if err != nil {
			log.Fatal(err)
		}
		return conn
	}
	clients := httpapi.Clients{
		Auth:        authv1.NewAuthServiceClient(dial("AUTH_SERVICE_ADDR", "auth:50051")),
		Users:       v1.NewUserServiceClient(dial("USER_SERVICE_ADDR", "user:50051")),
		Tournaments: v1.NewTournamentServiceClient(dial("TOURNAMENT_SERVICE_ADDR", "tournament:50051")),
		Teams:       v1.NewTeamServiceClient(dial("TEAM_SERVICE_ADDR", "team:50051")),
		Matches:     v1.NewMatchServiceClient(dial("MATCH_SERVICE_ADDR", "match:50051")),
		Scores:      v1.NewScoreServiceClient(dial("SCORE_SERVICE_ADDR", "score:50051")),
		Standings:   standingsv1.NewStandingsServiceClient(dial("STANDINGS_SERVICE_ADDR", "standings:50051")),
		Fixtures:    fixturev1.NewFixtureServiceClient(dial("FIXTURE_SERVICE_ADDR", "fixture:50051")),
		Officials:   officialv1.NewOfficialServiceClient(dial("OFFICIAL_SERVICE_ADDR", "official:50051")),
	}
	server := httpapi.New(clients, servicekit.Secret(), servicekit.Env("CORS_ORIGIN", "*"))
	addr := servicekit.Env("BFF_ADDR", ":8080")
	log.Printf("bff listening on %s", addr)
	log.Fatal((&http.Server{Addr: addr, Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}).ListenAndServe())
}
