package grpcadapter

import (
	"context"
	"testing"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/user/internal/app"
	"github.com/drukleague/platform/services/user/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const userID = "11111111-1111-4111-8111-111111111111"

type repo struct{}

func (repo) Create(_ context.Context, u domain.User, _ string) (domain.User, error) {
	u.ID, u.CreatedAt = userID, time.Now()
	return u, nil
}
func (repo) Get(_ context.Context, id string) (domain.User, error) {
	return domain.User{ID: id, Role: core.RoleViewer}, nil
}
func (repo) Credentials(_ context.Context, username string) (domain.User, string, error) {
	return domain.User{ID: userID, Username: username}, "ok", nil
}
func (repo) UpdatePasswordHash(context.Context, string, string) error { return nil }
func (repo) List(context.Context, string, int, int) ([]domain.User, bool, error) {
	return []domain.User{{ID: userID}, {ID: userID}}, true, nil
}
func (repo) CreateFirstSuperAdmin(context.Context, string, string) (bool, error) { return true, nil }

type hasher struct{}

func (hasher) Hash(p string) (string, error) { return p, nil }
func (hasher) Verify(hash, password string) (bool, bool) {
	return hash == "ok" && password == "secret-password", false
}

func as(role string) context.Context {
	return servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "caller", Role: role})
}

func TestUserRPCs(t *testing.T) {
	s := New(app.New(repo{}, hasher{}))
	created, err := s.CreateUser(as(core.RoleSuperAdmin), &v1.CreateUserRequest{Username: "karma", Password: "secret-password", Role: core.RoleMatchOfficial})
	if err != nil || created.GetUser().GetId() != userID || created.GetUser().GetCreatedAt() == nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.CreateUser(as(core.RoleViewer), &v1.CreateUserRequest{Username: "karma", Password: "secret-password", Role: core.RoleViewer}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("viewer create: %v", err)
	}
	if got, err := s.GetUser(as(core.RoleService), &v1.GetUserRequest{Id: userID}); err != nil || got.GetUser().GetRole() != core.RoleViewer {
		t.Fatalf("get: %v", err)
	}
	list, err := s.ListUsers(as(core.RoleSuperAdmin), &v1.ListUsersRequest{PageSize: 2})
	if err != nil || len(list.GetUsers()) != 2 || list.GetNextPageToken() != "2" {
		t.Fatalf("list: %v", err)
	}
	if _, err := s.ListUsers(as(core.RoleSuperAdmin), &v1.ListUsersRequest{PageToken: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatal("bad token accepted")
	}
	if _, err := s.ListUsers(as(core.RoleViewer), &v1.ListUsersRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatal("viewer listed users")
	}
	verified, err := s.VerifyCredentials(as(core.RoleService), &v1.VerifyCredentialsRequest{Username: "admin", Password: "secret-password"})
	if err != nil || verified.GetUser().GetUsername() != "admin" {
		t.Fatalf("verify: %v", err)
	}
	if _, err := s.VerifyCredentials(as(core.RoleService), &v1.VerifyCredentialsRequest{Username: "admin", Password: "wrong"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("wrong password: %v", err)
	}
}
