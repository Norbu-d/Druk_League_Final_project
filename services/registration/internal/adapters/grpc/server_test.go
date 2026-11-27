package grpcadapter

import (
	"context"
	"testing"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/registration/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/registration/internal/app"
	"github.com/drukleague/platform/services/registration/internal/domain"
	"github.com/drukleague/platform/services/registration/internal/ports"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	tournamentID = "11111111-1111-4111-8111-111111111111"
	teamID       = "22222222-2222-4222-8222-222222222222"
	regID        = "33333333-3333-4333-8333-333333333333"
)

type repo struct{}

func (repo) Create(_ context.Context, r domain.Registration) (domain.Registration, error) {
	r.ID, r.Status, r.SubmittedAt = regID, domain.StatusPending, time.Now()
	return r, nil
}
func (repo) Review(_ context.Context, id, decision, _, _ string) (domain.Registration, error) {
	return domain.Registration{ID: id, Status: decision, SubmittedAt: time.Now()}, nil
}
func (repo) List(context.Context, ports.Filter, int, int) ([]domain.Registration, bool, error) {
	return []domain.Registration{{ID: regID, SubmittedAt: time.Now()}}, true, nil
}

type teams struct{}

func (teams) Get(context.Context, string) (domain.TeamRef, error) {
	return domain.TeamRef{ID: teamID}, nil
}

type tournaments struct{}

func (tournaments) Get(context.Context, string) (domain.TournamentRef, error) {
	return domain.TournamentRef{ID: tournamentID, Status: domain.TournamentOpen}, nil
}

func TestRegistrationRPCs(t *testing.T) {
	s := New(app.New(repo{}, teams{}, tournaments{}))
	admin := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "a", Role: core.RoleSuperAdmin})

	registered, err := s.RegisterTeam(admin, &v1.RegisterTeamRequest{TournamentId: tournamentID, TeamId: teamID})
	if err != nil || registered.GetRegistration().GetId() != regID || registered.GetRegistration().GetSubmittedAt() == nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := s.RegisterTeam(context.Background(), &v1.RegisterTeamRequest{TournamentId: tournamentID, TeamId: teamID}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("anonymous register: %v", err)
	}
	if _, err := s.RegisterTeam(admin, &v1.RegisterTeamRequest{TournamentId: "x", TeamId: teamID}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad id: %v", err)
	}

	reviewed, err := s.ReviewRegistration(admin, &v1.ReviewRegistrationRequest{RegistrationId: regID, Approved: true})
	if err != nil || reviewed.GetRegistration().GetStatus() != domain.StatusApproved {
		t.Fatalf("review: %v", err)
	}
	if _, err := s.ReviewRegistration(admin, &v1.ReviewRegistrationRequest{RegistrationId: regID}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("rejection without a reason: %v", err)
	}
	if _, err := s.ReviewRegistration(context.Background(), &v1.ReviewRegistrationRequest{RegistrationId: regID, Approved: true}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("anonymous review: %v", err)
	}

	list, err := s.ListRegistrations(admin, &v1.ListRegistrationsRequest{PageSize: 1})
	if err != nil || len(list.GetRegistrations()) != 1 || list.GetNextPageToken() != "1" {
		t.Fatalf("list: %v %v", err, list)
	}
	if _, err := s.ListRegistrations(admin, &v1.ListRegistrationsRequest{PageToken: "nope"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad page token: %v", err)
	}
	if _, err := s.ListRegistrations(context.Background(), &v1.ListRegistrationsRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("anonymous list: %v", err)
	}
}
