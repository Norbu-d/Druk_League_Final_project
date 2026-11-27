package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/notification/internal/domain"
)

const (
	eventID      = "11111111-1111-4111-8111-111111111111"
	matchID      = "22222222-2222-4222-8222-222222222222"
	tournamentID = "33333333-3333-4333-8333-333333333333"
	homeID       = "44444444-4444-4444-8444-444444444444"
	awayID       = "55555555-5555-4555-8555-555555555555"
	managerA     = "66666666-6666-4666-8666-666666666666"
	managerB     = "77777777-7777-4777-8777-777777777777"
)

type fakeRepository struct {
	items []domain.Notification
	err   error
}

func (f *fakeRepository) Create(_ context.Context, item domain.Notification) (domain.Notification, bool, error) {
	if f.err != nil {
		return domain.Notification{}, false, f.err
	}
	if item.EventID != "" {
		for _, existing := range f.items {
			if existing.EventID == item.EventID && existing.UserID == item.UserID && existing.Channel == item.Channel {
				return existing, false, nil
			}
		}
	}
	item.ID = "88888888-8888-4888-8888-888888888888"
	item.CreatedAt = time.Date(2026, 11, 30, 10, 0, 0, 0, time.UTC)
	f.items = append(f.items, item)
	return item, true, nil
}

func (f *fakeRepository) List(_ context.Context, userID string, limit, offset int) ([]domain.Notification, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	filtered := make([]domain.Notification, 0)
	for _, item := range f.items {
		if item.UserID == userID {
			filtered = append(filtered, item)
		}
	}
	if offset >= len(filtered) {
		return []domain.Notification{}, false, nil
	}
	end := min(offset+limit, len(filtered))
	return filtered[offset:end], end < len(filtered), nil
}

type fakeTeams struct {
	items map[string]domain.Team
	err   error
}

func (f fakeTeams) Get(_ context.Context, id string) (domain.Team, error) {
	if f.err != nil {
		return domain.Team{}, f.err
	}
	item, ok := f.items[id]
	if !ok {
		return domain.Team{}, core.NotFound("team not found")
	}
	return item, nil
}

func TestSendAndListAuthorization(t *testing.T) {
	repo := &fakeRepository{}
	service := New(repo, fakeTeams{})
	admin := Caller{ID: managerA, Role: core.RoleSuperAdmin}
	item, err := service.Send(context.Background(), admin, managerB, "in_app", "Result ready", "A final result is ready.")
	if err != nil || item.UserID != managerB || item.Channel != "in_app" || item.Status != "pending" {
		t.Fatalf("send notification: item=%+v err=%v", item, err)
	}
	listed, more, err := service.List(context.Background(), Caller{ID: managerB, Role: core.RoleTeamManager}, managerB, 10, 0)
	if err != nil || more || len(listed) != 1 || listed[0].ID != item.ID {
		t.Fatalf("list own notifications: items=%+v more=%v err=%v", listed, more, err)
	}
	if _, _, err := service.List(context.Background(), Caller{ID: managerA, Role: core.RoleViewer}, managerB, 10, 0); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("reading another user's notifications = %v", err)
	}
	if _, err := service.Send(context.Background(), Caller{Role: core.RoleViewer}, managerB, "in_app", "Result", "Body"); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("viewer send = %v", err)
	}
	for _, tc := range []struct{ userID, channel, subject, body string }{
		{"bad-id", "in_app", "Result", "Body"},
		{managerB, "email", "Result", "Body"},
		{managerB, "in_app", "", "Body"},
		{managerB, "in_app", "Result", ""},
		{managerB, "in_app", strings.Repeat("s", 201), "Body"},
		{managerB, "in_app", "Result", strings.Repeat("b", 4001)},
	} {
		if _, err := service.Send(context.Background(), admin, tc.userID, tc.channel, tc.subject, tc.body); !errors.Is(err, core.ErrInvalid) {
			t.Errorf("invalid send %+v = %v", tc, err)
		}
	}
	if _, _, err := service.List(context.Background(), Caller{ID: managerB}, "bad-id", 10, 0); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("invalid notification owner = %v", err)
	}
}

func TestNotifyMatchFinalizedIsIdempotent(t *testing.T) {
	repo := &fakeRepository{}
	teams := fakeTeams{items: map[string]domain.Team{
		homeID: {ID: homeID, Name: "Thimphu FC", Dzongkhag: "Thimphu", TournamentID: tournamentID, ManagerID: managerA},
		awayID: {ID: awayID, Name: "Paro FC", Dzongkhag: "Paro", TournamentID: tournamentID, ManagerID: managerB},
	}}
	service := New(repo, teams)
	event := finalizedEvent()
	if err := service.NotifyMatchFinalized(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := service.NotifyMatchFinalized(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if len(repo.items) != 2 {
		t.Fatalf("duplicate event created %d notifications, want two unique recipients", len(repo.items))
	}
	for _, item := range repo.items {
		if item.EventID != eventID || item.Channel != "in_app" || item.Subject != "Match result finalized" || !strings.Contains(item.Body, "Thimphu FC 2–1 Paro FC") || !strings.Contains(item.Body, "Thimphu and Paro") {
			t.Errorf("unexpected event notification: %+v", item)
		}
	}
}

func TestNotifyMatchFinalizedSharedManagerAndErrors(t *testing.T) {
	repo := &fakeRepository{}
	teams := fakeTeams{items: map[string]domain.Team{
		homeID: {ID: homeID, Name: "Thimphu FC", Dzongkhag: "Thimphu", TournamentID: tournamentID, ManagerID: managerA},
		awayID: {ID: awayID, Name: "Paro FC", Dzongkhag: "Paro", TournamentID: tournamentID, ManagerID: managerA},
	}}
	service := New(repo, teams)
	if err := service.NotifyMatchFinalized(context.Background(), finalizedEvent()); err != nil || len(repo.items) != 1 {
		t.Fatalf("shared manager notification: items=%+v err=%v", repo.items, err)
	}

	if err := New(&fakeRepository{}, fakeTeams{err: errors.New("team service unavailable")}).NotifyMatchFinalized(context.Background(), finalizedEvent()); err == nil {
		t.Fatal("team service error was ignored")
	}
	missingManager := fakeTeams{items: map[string]domain.Team{
		homeID: {ID: homeID, Name: "Thimphu FC", Dzongkhag: "Thimphu", TournamentID: tournamentID, ManagerID: managerA},
		awayID: {ID: awayID, Name: "Paro FC", Dzongkhag: "Paro", TournamentID: tournamentID},
	}}
	if err := New(&fakeRepository{}, missingManager).NotifyMatchFinalized(context.Background(), finalizedEvent()); !errors.Is(err, core.ErrPrecondition) {
		t.Fatalf("unassigned team error = %v", err)
	}
	wrongTournament := fakeTeams{items: map[string]domain.Team{}}
	for id, item := range teams.items {
		wrongTournament.items[id] = item
	}
	wrong := wrongTournament.items[awayID]
	wrong.TournamentID = homeID
	wrongTournament.items[awayID] = wrong
	if err := New(&fakeRepository{}, wrongTournament).NotifyMatchFinalized(context.Background(), finalizedEvent()); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("mismatched tournament error = %v", err)
	}
	repoError := errors.New("database unavailable")
	if err := New(&fakeRepository{err: repoError}, teams).NotifyMatchFinalized(context.Background(), finalizedEvent()); !errors.Is(err, repoError) {
		t.Fatalf("repository error = %v", err)
	}
	invalid := finalizedEvent()
	invalid.HomeScore = 100
	if err := service.NotifyMatchFinalized(context.Background(), invalid); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("invalid finalized event = %v", err)
	}
}

func finalizedEvent() domain.MatchFinalized {
	return domain.MatchFinalized{EventID: eventID, MatchID: matchID, TournamentID: tournamentID, HomeTeamID: homeID, AwayTeamID: awayID, HomeScore: 2, AwayScore: 1, FinalizedAt: time.Date(2026, 11, 30, 10, 0, 0, 0, time.UTC)}
}
