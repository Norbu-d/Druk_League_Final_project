package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/services/analytics/internal/domain"
)

type recordingRepository struct {
	event      domain.MatchFinalized
	tournament domain.Tournament
	recordErr  error
}

func (r *recordingRepository) RecordFinalized(_ context.Context, event domain.MatchFinalized, tournament domain.Tournament) error {
	r.event, r.tournament = event, tournament
	return r.recordErr
}
func (r *recordingRepository) DashboardMetrics(context.Context, string, string) (domain.DashboardMetrics, error) {
	return domain.DashboardMetrics{}, nil
}

type tournamentDirectory struct {
	tournament domain.Tournament
	err        error
}

func (d tournamentDirectory) GetTournament(context.Context, string) (domain.Tournament, error) {
	return d.tournament, d.err
}

func validFinalizedEvent() domain.MatchFinalized {
	return domain.MatchFinalized{
		EventID: "11111111-1111-4111-8111-111111111111", MatchID: "22222222-2222-4222-8222-222222222222",
		TournamentID: "33333333-3333-4333-8333-333333333333", HomeTeamID: "44444444-4444-4444-8444-444444444444",
		AwayTeamID: "55555555-5555-4555-8555-555555555555", HomeScore: 2, AwayScore: 1,
		FinalizedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("test", 3600)),
	}
}

func TestRecordFinalizedValidatesResolvesAndPersists(t *testing.T) {
	repo := &recordingRepository{}
	directory := tournamentDirectory{tournament: domain.Tournament{
		ID: " 33333333-3333-4333-8333-333333333333 ", Season: " 2026 ", Dzongkhag: " Thimphu ",
	}}
	err := New(repo, directory).RecordFinalized(context.Background(), validFinalizedEvent())
	if err != nil {
		t.Fatal(err)
	}
	if repo.event.EventID != "11111111-1111-4111-8111-111111111111" || repo.event.FinalizedAt.Location() != time.UTC {
		t.Fatalf("event not normalized/persisted: %+v", repo.event)
	}
	if repo.tournament.Season != "2026" || repo.tournament.Dzongkhag != "Thimphu" {
		t.Fatalf("tournament not normalized: %+v", repo.tournament)
	}
}

func TestRecordFinalizedRejectsInvalidEventAndTournament(t *testing.T) {
	validTournament := domain.Tournament{ID: "33333333-3333-4333-8333-333333333333", Season: "2026", Dzongkhag: "Thimphu"}
	tests := []struct {
		name       string
		event      domain.MatchFinalized
		tournament domain.Tournament
	}{
		{"invalid event", domain.MatchFinalized{}, validTournament},
		{"wrong tournament id", validFinalizedEvent(), domain.Tournament{ID: "66666666-6666-4666-8666-666666666666", Season: "2026", Dzongkhag: "Thimphu"}},
		{"missing season", validFinalizedEvent(), domain.Tournament{ID: validTournament.ID, Dzongkhag: "Thimphu"}},
		{"missing dzongkhag", validFinalizedEvent(), domain.Tournament{ID: validTournament.ID, Season: "2026"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(&recordingRepository{}, tournamentDirectory{tournament: tt.tournament}).RecordFinalized(context.Background(), tt.event)
			if !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestRecordFinalizedWrapsDependencyAndReturnsRepositoryErrors(t *testing.T) {
	dependencyErr := errors.New("tournament unavailable")
	service := New(&recordingRepository{}, tournamentDirectory{err: dependencyErr})
	if err := service.RecordFinalized(context.Background(), validFinalizedEvent()); !errors.Is(err, domain.ErrDependency) {
		t.Fatalf("dependency error=%v", err)
	}
	repositoryErr := errors.New("database unavailable")
	repo := &recordingRepository{recordErr: repositoryErr}
	service = New(repo, tournamentDirectory{tournament: domain.Tournament{ID: "33333333-3333-4333-8333-333333333333", Season: "2026", Dzongkhag: "Thimphu"}})
	if err := service.RecordFinalized(context.Background(), validFinalizedEvent()); !errors.Is(err, repositoryErr) {
		t.Fatalf("repository error=%v", err)
	}
}
