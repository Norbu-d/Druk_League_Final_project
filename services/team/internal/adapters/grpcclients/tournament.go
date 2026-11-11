package grpcclients

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/team/internal/domain"
	"google.golang.org/grpc"
)

// Tournaments reads tournaments from the Tournament service.
type Tournaments struct{ client v1.TournamentServiceClient }

func NewTournaments(conn grpc.ClientConnInterface) *Tournaments {
	return &Tournaments{client: v1.NewTournamentServiceClient(conn)}
}

func (t *Tournaments) Get(ctx context.Context, id string) (domain.TournamentRef, error) {
	return servicekit.FetchDependency(ctx, func(ctx context.Context) (*v1.GetTournamentResponse, error) {
		return t.client.GetTournament(ctx, &v1.GetTournamentRequest{Id: id})
	}, "tournament", func(response *v1.GetTournamentResponse) (domain.TournamentRef, bool) {
		tournament := response.GetTournament()
		if tournament == nil {
			return domain.TournamentRef{}, false
		}
		return domain.TournamentRef{ID: tournament.GetId(), Status: tournament.GetStatus()}, true
	})
}
