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
	entries     ports.TournamentEntries
	venues      ports.Venues
	matches     ports.Matches
	standings   ports.Standings
	now         func() time.Time
	newID       func() (string, error)
}

func New(repo ports.Repository, tournaments ports.Tournaments, entries ports.TournamentEntries, venues ports.Venues, matches ports.Matches, standings ...ports.Standings) *Service {
	service := &Service{repo: repo, tournaments: tournaments, entries: entries, venues: venues, matches: matches, now: time.Now, newID: core.NewUUID}
	if len(standings) > 0 {
		service.standings = standings[0]
	}
	return service
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
	teamIDs, err := s.entries.ApprovedTeamIDs(ctx, tournamentID)
	if err != nil {
		return nil, 0, err
	}
	venues, err := s.venues.InDzongkhag(ctx, tournament.Dzongkhag)
	if err != nil {
		return nil, 0, err
	}
	var schedule []domain.Fixture
	var knockout domain.KnockoutPlan
	if format == domain.FormatKnockout {
		if s.standings == nil {
			return nil, 0, core.Dependency("standings service is not configured")
		}
		ranked, err := s.standings.TeamIDsByRank(ctx, tournamentID)
		if err != nil {
			return nil, 0, err
		}
		teamSet := make(map[string]bool, len(teamIDs))
		for _, id := range teamIDs {
			teamSet[id] = true
		}
		qualified := make([]string, 0, len(ranked))
		for _, id := range ranked {
			if teamSet[id] {
				qualified = append(qualified, id)
			}
		}
		if len(qualified) != len(teamIDs) {
			return nil, 0, core.Precondition("every tournament team needs a standings row before knockout fixtures can be generated")
		}
		knockout, err = domain.KnockoutSchedule(tournamentID, qualified, venues, start.UTC())
		if err != nil {
			return nil, 0, err
		}
		schedule = knockout.Fixtures
	} else {
		schedule, err = domain.Schedule(tournamentID, teamIDs, venues, start.UTC(), format)
		if err != nil {
			return nil, 0, err
		}
	}
	eventID, err := s.newID()
	if err != nil {
		return nil, 0, err
	}
	payload, err := json.Marshal(domain.FixturesGenerated{EventID: eventID, TournamentID: tournamentID, Fixtures: len(schedule), GeneratedBy: caller.ID, GeneratedAt: s.now().UTC()})
	if err != nil {
		return nil, 0, err
	}
	var stored []domain.Fixture
	var created bool
	if format == domain.FormatKnockout {
		stored, created, err = s.repo.SaveKnockoutSchedule(ctx, tournamentID, knockout, domain.OutboxEvent{EventID: eventID, Payload: payload})
	} else {
		stored, created, err = s.repo.SaveSchedule(ctx, tournamentID, schedule, domain.OutboxEvent{EventID: eventID, Payload: payload})
	}
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

// HandleMatchFinalized advances a knockout bracket once both feeder matches for a slot
// have finalized. Persisting the result and creating the next fixture are idempotent.
func (s *Service) HandleMatchFinalized(ctx context.Context, event domain.MatchFinalized) error {
	fixture, created, err := s.repo.AdvanceKnockout(ctx, event)
	if err != nil {
		return err
	}
	if created && fixture.Format == domain.FormatLeagueKnockout {
		if err := s.startKnockout(ctx, event.TournamentID, event.FinalizedAt.AddDate(0, 0, 7)); err != nil {
			return err
		}
	}
	pending, err := s.repo.PendingFixtures(ctx, event.TournamentID)
	if err != nil {
		return err
	}
	for _, next := range pending {
		matchID, err := s.matches.Create(ctx, next)
		if err != nil {
			return err
		}
		next.MatchID = matchID
		if err := s.repo.LinkMatch(ctx, next); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) startKnockout(ctx context.Context, tournamentID string, start time.Time) error {
	if s.standings == nil {
		return core.Dependency("standings service is not configured")
	}
	tournament, err := s.tournaments.Get(ctx, tournamentID)
	if err != nil {
		return err
	}
	teamIDs, err := s.entries.ApprovedTeamIDs(ctx, tournamentID)
	if err != nil {
		return err
	}
	ranked, err := s.standings.TeamIDsByRank(ctx, tournamentID)
	if err != nil {
		return err
	}
	teamSet := make(map[string]bool, len(teamIDs))
	for _, id := range teamIDs {
		teamSet[id] = true
	}
	qualified := make([]string, 0, len(ranked))
	for _, id := range ranked {
		if teamSet[id] {
			qualified = append(qualified, id)
		}
	}
	if len(qualified) != len(teamIDs) {
		return core.Precondition("every tournament team needs a standings row before knockout fixtures can be generated")
	}
	venues, err := s.venues.InDzongkhag(ctx, tournament.Dzongkhag)
	if err != nil {
		return err
	}
	plan, err := domain.KnockoutSchedule(tournamentID, qualified, venues, start.UTC())
	if err != nil {
		return err
	}
	eventID, err := s.newID()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(domain.FixturesGenerated{EventID: eventID, TournamentID: tournamentID, Fixtures: len(plan.Fixtures), GeneratedBy: "fixture-service", GeneratedAt: s.now().UTC()})
	if err != nil {
		return err
	}
	_, _, err = s.repo.SaveKnockoutSchedule(ctx, tournamentID, plan, domain.OutboxEvent{EventID: eventID, Payload: payload})
	return err
}

func (s *Service) List(ctx context.Context, tournamentID string, limit, offset int) ([]domain.Fixture, bool, error) {
	tournamentID = strings.TrimSpace(tournamentID)
	if !core.IsUUID(tournamentID) {
		return nil, false, core.Invalid("tournament id must be a UUID")
	}
	return s.repo.List(ctx, tournamentID, limit, offset)
}
