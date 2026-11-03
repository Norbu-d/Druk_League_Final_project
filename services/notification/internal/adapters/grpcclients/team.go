package grpcclients

import (
	"context"
	"fmt"
	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/services/notification/internal/ports"
	"google.golang.org/grpc"
)

type Teams struct{ client v1.TeamServiceClient }

func NewTeams(conn grpc.ClientConnInterface) *Teams {
	return &Teams{client: v1.NewTeamServiceClient(conn)}
}
func (t *Teams) GetTeam(ctx context.Context, id string) (ports.Team, error) {
	r, e := t.client.GetTeam(ctx, &v1.GetTeamRequest{Id: id})
	if e != nil {
		return ports.Team{}, e
	}
	x := r.GetTeam()
	if x == nil {
		return ports.Team{}, fmt.Errorf("team response is empty")
	}
	return ports.Team{ID: x.GetId(), Name: x.GetName(), Dzongkhag: x.GetDzongkhag(), ManagerID: x.GetManagerId()}, nil
}
