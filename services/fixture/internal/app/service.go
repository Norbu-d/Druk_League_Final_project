package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
		booked, err := s.book(ctx, stored[i], venues)
		if err != nil {
			return stored, createdCount, err
		}
		if err := s.repo.LinkMatch(ctx, booked); err != nil {
			return stored, createdCount, err
		}
		stored[i] = booked
	}
	return stored, createdCount, nil
}

// book creates the fixture's match in the first free slot. When Match reports that a venue
// or team is already booked (by another tournament), it tries the other venues and later
// kickoffs on the same day before giving up.
func (s *Service) book(ctx context.Context, f domain.Fixture, venues []domain.Venue) (domain.Fixture, error) {
	var clash error
	for _, slot := range domain.Alternatives(f, venues) {
		matchID, err := s.matches.Create(ctx, slot)
		if err == nil {
			slot.MatchID = matchID
			return slot, nil
		}
		if !errors.Is(err, core.ErrConflict) {
			return f, err
		}
		clash = err
	}
	return f, core.Conflict(fmt.Sprintf("no free venue or kickoff on %s for round %d; last clash: %s",
		f.Kickoff.UTC().Format("2006-01-02"), f.Round, strings.TrimPrefix(clash.Error(), core.ErrConflict.Error()+": ")))
}

func (s *Service) List(ctx context.Context, tournamentID string, limit, offset int) ([]domain.Fixture, bool, error) {
	tournamentID = strings.TrimSpace(tournamentID)
	if !core.IsUUID(tournamentID) {
		return nil, false, core.Invalid("tournament id must be a UUID")
	}
	return s.repo.List(ctx, tournamentID, limit, offset)
}
