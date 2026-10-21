package grpcclients

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/registration/internal/domain"
	"google.golang.org/grpc"
)

// Tournaments reads tournaments from the Tournament service.
type Tournaments struct{ client v1.TournamentServiceClient }

func NewTournaments(conn grpc.ClientConnInterface) *Tournaments {
	return &Tournaments{client: v1.NewTournamentServiceClient(conn)}
}

func (t *Tournaments) Get(ctx context.Context, id string) (domain.TournamentRef, error) {
	response, err := t.client.GetTournament(ctx, &v1.GetTournamentRequest{Id: id})
	if err != nil {
		return domain.TournamentRef{}, servicekit.FromStatus(err, "tournament")
	}
	tournament := response.GetTournament()
	if tournament == nil {
		return domain.TournamentRef{}, core.Dependency("tournament service returned no tournament")
	}
	return domain.TournamentRef{ID: tournament.GetId(), Status: tournament.GetStatus()}, nil
}
