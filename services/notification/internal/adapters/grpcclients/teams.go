package grpcclients

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/notification/internal/domain"
	"google.golang.org/grpc"
)

type TeamDirectory struct{ client v1.TeamServiceClient }

func NewTeamDirectory(conn grpc.ClientConnInterface) *TeamDirectory {
	return &TeamDirectory{client: v1.NewTeamServiceClient(conn)}
}

func (t *TeamDirectory) Get(ctx context.Context, id string) (domain.Team, error) {
	response, err := t.client.GetTeam(ctx, &v1.GetTeamRequest{Id: id})
	if err != nil {
		return domain.Team{}, servicekit.FromStatus(err, "team")
	}
	team := response.GetTeam()
	if team == nil {
		return domain.Team{}, core.NotFound("team not found")
	}
	return domain.Team{ID: team.GetId(), Name: team.GetName(), ShortCode: team.GetShortCode(), Dzongkhag: team.GetDzongkhag(), TournamentID: team.GetTournamentId(), ManagerID: team.GetManagerId()}, nil
}
