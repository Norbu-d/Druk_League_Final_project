package grpcclients

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/player/internal/domain"
	"google.golang.org/grpc"
)

// Teams reads teams from the Team service.
type Teams struct{ client v1.TeamServiceClient }

func NewTeams(conn grpc.ClientConnInterface) *Teams {
	return &Teams{client: v1.NewTeamServiceClient(conn)}
}

func (t *Teams) Get(ctx context.Context, id string) (domain.TeamRef, error) {
	return servicekit.FetchDependency(ctx, func(ctx context.Context) (*v1.GetTeamResponse, error) {
		return t.client.GetTeam(ctx, &v1.GetTeamRequest{Id: id})
	}, "team", func(response *v1.GetTeamResponse) (domain.TeamRef, bool) {
		team := response.GetTeam()
		if team == nil {
			return domain.TeamRef{}, false
		}
		return domain.TeamRef{ID: team.GetId(), ManagerID: team.GetManagerId()}, true
	})
}
