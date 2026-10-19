package app

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/fixture/internal/domain"
	"github.com/drukleague/platform/services/fixture/internal/ports"
)

type Caller struct{ ID, Role string }

type Service struct {
	repo        ports.Repository
	tournaments ports.Tournaments
	teams       ports.Teams
	venues      ports.Venues
	matches     ports.Matches
	now         func() time.Time
	newID       func() (string, error)
}

func New(repo ports.Repository, tournaments ports.Tournaments, teams ports.Teams, venues ports.Venues, matches ports.Matches) *Service {
	return &Service{repo: repo, tournaments: tournaments, teams: teams, venues: venues, matches: matches, now: time.Now, newID: core.NewUUID}
}

// Generate stores the tournament's schedule once and creates one match per fixture in the
// Match service. Running it again resumes any fixture whose match was not yet created and
// never duplicates fixtures; the returned count is how many fixtures were new.
func (s *Service) Generate(ctx context.Context, caller Caller, tournamentID, format string, start time.Time) ([]domain.Fixture, int, error) {
	if caller.Role != core.RoleSuperAdmin {
		return nil, 0, core.Forbidden("only the tournament admin can generate fixtures")
	}
	tournamentID = strings.TrimSpace(tournamentID)
	if !core.IsUUID(tournamentID) {
		return nil, 0, core.Invalid("tournament id must be a UUID")
	}
	format, err := domain.NormalizeFormat(strings.ToLower(strings.TrimSpace(format)))
	if err != nil {
		return nil, 0, err
	}
	if start.IsZero() {
		start = domain.DefaultStart(s.now())
	}
	tournament, err := s.tournaments.Get(ctx, tournamentID)
	if err != nil {
		return nil, 0, err
	}
	teamIDs, err := s.teams.IDsInTournament(ctx, tournamentID)
	if err != nil {
		return nil, 0, err
	}
	venues, err := s.venues.InDzongkhag(ctx, tournament.Dzongkhag)
	if err != nil {
		return nil, 0, err
	}
	schedule, err := domain.Schedule(tournamentID, teamIDs, venues, start.UTC(), format)
	if err != nil {
		return nil, 0, err
	}
	eventID, err := s.newID()
	if err != nil {
		return nil, 0, err
	}
	payload, err := json.Marshal(domain.FixturesGenerated{EventID: eventID, TournamentID: tournamentID, Fixtures: len(schedule), GeneratedBy: caller.ID, GeneratedAt: s.now().UTC()})
	if err != nil {
		return nil, 0, err
	}
	stored, created, err := s.repo.SaveSchedule(ctx, tournamentID, schedule, domain.OutboxEvent{EventID: eventID, Payload: payload})
	if err != nil {
		return nil, 0, err
	}
	createdCount := 0
	if created {
		createdCount = len(stored)
	}
	for i := range stored {
		if stored[i].MatchID != "" {
			continue
		}
		matchID, err := s.matches.Create(ctx, stored[i])
		if err != nil {
			return stored, createdCount, err
		}
		if err := s.repo.LinkMatch(ctx, stored[i].ID, matchID); err != nil {
			return stored, createdCount, err
		}
		stored[i].MatchID = matchID
	}
	return stored, createdCount, nil
}

func (s *Service) List(ctx context.Context, tournamentID string, limit, offset int) ([]domain.Fixture, bool, error) {
	tournamentID = strings.TrimSpace(tournamentID)
	if !core.IsUUID(tournamentID) {
		return nil, false, core.Invalid("tournament id must be a UUID")
	}
	return s.repo.List(ctx, tournamentID, limit, offset)
}
