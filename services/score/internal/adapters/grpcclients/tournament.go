package grpcclients

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/services/score/internal/domain"
	"google.golang.org/grpc"
)

// TournamentReader reads a tournament's sport through a servicekit.Dial connection.
type TournamentReader struct{ client v1.TournamentServiceClient }

func NewTournamentReader(conn grpc.ClientConnInterface) *TournamentReader {
	return &TournamentReader{client: v1.NewTournamentServiceClient(conn)}
}

func (r *TournamentReader) Sport(ctx context.Context, tournamentID string) (string, error) {
	response, err := r.client.GetTournament(ctx, &v1.GetTournamentRequest{Id: tournamentID})
	if err != nil {
		return "", domain.ErrDependency
	}
	return response.GetTournament().GetSport(), nil
}
