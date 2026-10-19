package grpcclients

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/official/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Matches calls the Match service with the admin's forwarded token.
type Matches struct{ client v1.MatchServiceClient }

func NewMatches(conn grpc.ClientConnInterface) *Matches {
	return &Matches{client: v1.NewMatchServiceClient(conn)}
}

func (m *Matches) AssignOfficial(ctx context.Context, matchID, officialUserID string) (domain.MatchRef, error) {
	response, err := m.client.AssignOfficial(ctx, &v1.AssignOfficialRequest{MatchId: matchID, OfficialId: officialUserID})
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return domain.MatchRef{}, core.Invalid("the user is not a match official or the ids are invalid")
		case codes.AlreadyExists:
			return domain.MatchRef{}, core.Conflict("the match is already completed")
		}
		return domain.MatchRef{}, servicekit.FromStatus(err, "match")
	}
	match := response.GetMatch()
	if match == nil {
		return domain.MatchRef{}, core.Dependency("match service returned no match")
	}
	return domain.MatchRef{ID: match.GetId(), TournamentID: match.GetTournamentId(), OfficialID: match.GetOfficialId()}, nil
}
