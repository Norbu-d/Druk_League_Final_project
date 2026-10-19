package grpcadapter

import (
	"context"
	"testing"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/official/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/official/internal/app"
	"github.com/drukleague/platform/services/official/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	matchID  = "11111111-1111-4111-8111-111111111111"
	official = "22222222-2222-4222-8222-222222222222"
)

type repo struct{}

func (repo) Record(_ context.Context, a domain.Assignment) (domain.Assignment, error) {
	a.ID, a.AssignedAt = "a1", time.Now()
	return a, nil
}
func (repo) List(context.Context, string, string) ([]domain.Assignment, error) {
	return []domain.Assignment{{ID: "a1", MatchID: matchID, OfficialUserID: official}}, nil
}

type matches struct{}

func (matches) AssignOfficial(_ context.Context, m, o string) (domain.MatchRef, error) {
	return domain.MatchRef{ID: m, OfficialID: o}, nil
}

func TestOfficialRPCs(t *testing.T) {
	s := New(app.New(repo{}, matches{}))
	admin := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "a", Role: core.RoleSuperAdmin})
	assigned, err := s.AssignOfficial(admin, &v1.AssignOfficialRequest{MatchId: matchID, OfficialUserId: official})
	if err != nil || assigned.GetAssignment().GetOfficialUserId() != official || assigned.GetAssignment().GetAssignedAt() == nil {
		t.Fatalf("assign: %v", err)
	}
	if _, err := s.AssignOfficial(context.Background(), &v1.AssignOfficialRequest{MatchId: matchID, OfficialUserId: official}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("anonymous: %v", err)
	}
	list, err := s.ListAssignments(admin, &v1.ListAssignmentsRequest{})
	if err != nil || len(list.GetAssignments()) != 1 {
		t.Fatalf("list: %v", err)
	}
	if _, err := s.ListAssignments(context.Background(), &v1.ListAssignmentsRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("anonymous list: %v", err)
	}
}
