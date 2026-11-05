package grpcclients

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/services/analytics/internal/domain"
	"google.golang.org/grpc"
)

type Tournaments struct{ client v1.TournamentServiceClient }

func NewTournaments(conn grpc.ClientConnInterface) *Tournaments {
	return &Tournaments{client: v1.NewTournamentServiceClient(conn)}
}

func (t *Tournaments) GetTournament(ctx context.Context, id string) (domain.Tournament, error) {
	response, err := t.client.GetTournament(ctx, &v1.GetTournamentRequest{Id: id})
	if err != nil {
		return domain.Tournament{}, err
	}
	item := response.GetTournament()
	if item == nil {
		return domain.Tournament{}, domain.ErrInvalid
	}
	return domain.Tournament{ID: item.GetId(), Season: item.GetSeason(), Dzongkhag: item.GetDzongkhag()}, nil
}
