package grpcclients

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/services/match/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// The connections passed in come from servicekit.Dial, which adds deadlines, retries,
// the circuit breaker and the bearer token.
type TeamDirectory struct{ client v1.TeamServiceClient }
type UserDirectory struct{ client v1.UserServiceClient }

func NewTeamDirectory(conn grpc.ClientConnInterface) *TeamDirectory {
	return &TeamDirectory{client: v1.NewTeamServiceClient(conn)}
}
func NewUserDirectory(conn grpc.ClientConnInterface) *UserDirectory {
	return &UserDirectory{client: v1.NewUserServiceClient(conn)}
}

func (d *TeamDirectory) TournamentForTeam(ctx context.Context, id string) (string, error) {
	response, err := d.client.GetTeam(ctx, &v1.GetTeamRequest{Id: id})
	if err != nil {
		return "", lookupError(err)
	}
	if response.GetTeam() == nil {
		return "", domain.ErrDependency
	}
	return response.GetTeam().GetTournamentId(), nil
}

func (d *UserDirectory) RoleForUser(ctx context.Context, id string) (string, error) {
	response, err := d.client.GetUser(ctx, &v1.GetUserRequest{Id: id})
	if err != nil {
		return "", lookupError(err)
	}
	if response.GetUser() == nil {
		return "", domain.ErrDependency
	}
	return response.GetUser().GetRole(), nil
}

func lookupError(err error) error {
	if code := status.Code(err); code == codes.NotFound || code == codes.InvalidArgument {
		return domain.ErrInvalid
	}
	return domain.ErrDependency
}
