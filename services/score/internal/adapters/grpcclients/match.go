package grpcclients

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/services/score/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MatchReader reads matches through a servicekit.Dial connection (deadline, retry, breaker, token).
type MatchReader struct{ client v1.MatchServiceClient }

func NewMatchReader(conn grpc.ClientConnInterface) *MatchReader {
	return &MatchReader{client: v1.NewMatchServiceClient(conn)}
}

func (r *MatchReader) GetMatch(ctx context.Context, id string) (domain.Match, error) {
	response, err := r.client.GetMatch(ctx, &v1.GetMatchRequest{Id: id})
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound:
			return domain.Match{}, domain.ErrNotFound
		case codes.InvalidArgument:
			return domain.Match{}, domain.ErrInvalid
		default:
			return domain.Match{}, domain.ErrDependency
		}
	}
	m := response.GetMatch()
	if m == nil {
		return domain.Match{}, domain.ErrDependency
	}
	return domain.Match{ID: m.GetId(), TournamentID: m.GetTournamentId(), HomeTeamID: m.GetHomeTeamId(), AwayTeamID: m.GetAwayTeamId(), OfficialID: m.GetOfficialId(), Status: m.GetStatus()}, nil
}
