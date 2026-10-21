package grpcclients

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/registration/internal/domain"
	"google.golang.org/grpc"
)

// Teams reads teams from the Team service.
type Teams struct{ client v1.TeamServiceClient }

func NewTeams(conn grpc.ClientConnInterface) *Teams {
	return &Teams{client: v1.NewTeamServiceClient(conn)}
}

func (t *Teams) Get(ctx context.Context, id string) (domain.TeamRef, error) {
	response, err := t.client.GetTeam(ctx, &v1.GetTeamRequest{Id: id})
	if err != nil {
		return domain.TeamRef{}, servicekit.FromStatus(err, "team")
	}
	team := response.GetTeam()
	if team == nil {
		return domain.TeamRef{}, core.Dependency("team service returned no team")
	}
	return domain.TeamRef{ID: team.GetId(), ManagerID: team.GetManagerId()}, nil
}
