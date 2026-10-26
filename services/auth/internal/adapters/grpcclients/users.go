package grpcclients

import (
	"context"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/auth/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Users calls the User service with Auth's own service token (login has no caller token).
type Users struct{ client v1.UserServiceClient }

func NewUsers(conn grpc.ClientConnInterface) *Users {
	return &Users{client: v1.NewUserServiceClient(conn)}
}

func (u *Users) Verify(ctx context.Context, username, password string) (domain.User, error) {
	response, err := u.client.VerifyCredentials(ctx, &v1.VerifyCredentialsRequest{Username: username, Password: password})
	if err != nil {
		if status.Code(err) == codes.Unauthenticated {
			return domain.User{}, core.ErrUnauthenticated
		}
		return domain.User{}, core.Dependency("user service unavailable")
	}
	return toUser(response.GetUser()), nil
}

func (u *Users) Get(ctx context.Context, id string) (domain.User, error) {
	response, err := u.client.GetUser(ctx, &v1.GetUserRequest{Id: id})
	if err != nil {
		return domain.User{}, servicekit.FromStatus(err, "user")
	}
	return toUser(response.GetUser()), nil
}

func toUser(user *v1.User) domain.User {
	return domain.User{ID: user.GetId(), Username: user.GetUsername(), Role: user.GetRole()}
}

// Signer issues access tokens in the shared servicekit format.
type Signer struct{ secret []byte }

func NewSigner(secret []byte) *Signer { return &Signer{secret: secret} }

func (s *Signer) Sign(user domain.User, sessionID string, ttl time.Duration) (string, time.Time, error) {
	return servicekit.SignToken(s.secret, servicekit.Identity{ID: user.ID, Username: user.Username, Role: user.Role, SessionID: sessionID}, ttl)
}
