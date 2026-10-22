package grpcclients

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/services/standings/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TeamDirectory reads team metadata. The connection comes from servicekit.Dial with the
// ServiceIdentity mode, whose TokenSource re-signs the service token before it expires.
type TeamDirectory struct{ client v1.TeamServiceClient }

func NewTeamDirectory(conn grpc.ClientConnInterface) *TeamDirectory {
	return &TeamDirectory{client: v1.NewTeamServiceClient(conn)}
}

func (d *TeamDirectory) GetTeam(ctx context.Context, id string) (domain.Team, error) {
	response, err := d.client.GetTeam(ctx, &v1.GetTeamRequest{Id: id})
	if err != nil {
		if code := status.Code(err); code == codes.NotFound || code == codes.InvalidArgument {
			return domain.Team{}, domain.ErrInvalid
		}
		return domain.Team{}, domain.ErrDependency
	}
	team := response.GetTeam()
	if team == nil {
		return domain.Team{}, domain.ErrDependency
	}
	return domain.Team{ID: team.GetId(), Name: team.GetName(), Dzongkhag: team.GetDzongkhag(), TournamentID: team.GetTournamentId()}, nil
}
