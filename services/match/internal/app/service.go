package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/match/internal/domain"
	"github.com/drukleague/platform/services/match/internal/ports"
)

type Service struct {
	repo  ports.Repository
	teams ports.TeamDirectory
	users ports.UserDirectory
}

func New(repo ports.Repository, teams ports.TeamDirectory, users ports.UserDirectory) *Service {
	return &Service{repo: repo, teams: teams, users: users}
}

func (s *Service) Create(ctx context.Context, role string, m domain.Match) (domain.Match, error) {
	if role != core.RoleSuperAdmin {
		return domain.Match{}, domain.ErrForbidden
	}
	m.TournamentID = strings.TrimSpace(m.TournamentID)
	m.HomeTeamID = strings.TrimSpace(m.HomeTeamID)
	m.AwayTeamID = strings.TrimSpace(m.AwayTeamID)
	m.Venue = strings.TrimSpace(m.Venue)
	if !core.IsUUID(m.TournamentID) || !core.IsUUID(m.HomeTeamID) || !core.IsUUID(m.AwayTeamID) || m.Kickoff.IsZero() || m.HomeTeamID == m.AwayTeamID {
		return domain.Match{}, domain.ErrInvalid
	}
	for _, teamID := range []string{m.HomeTeamID, m.AwayTeamID} {
		tournamentID, err := s.teams.TournamentForTeam(ctx, teamID)
		if errors.Is(err, domain.ErrInvalid) {
			return domain.Match{}, domain.ErrInvalid
		}
		if err != nil {
			return domain.Match{}, domain.ErrDependency
		}
		if tournamentID != m.TournamentID {
			return domain.Match{}, domain.ErrInvalid
		}
	}
	m.Status = domain.StatusScheduled
	return s.repo.Create(ctx, m)
}

func (s *Service) Get(ctx context.Context, id string) (domain.Match, error) {
	id = strings.TrimSpace(id)
	if !core.IsUUID(id) {
		return domain.Match{}, domain.ErrInvalid
	}
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, tournamentID string, after *time.Time, limit, offset int) ([]domain.Match, bool, error) {
	tournamentID = strings.TrimSpace(tournamentID)
	if tournamentID != "" && !core.IsUUID(tournamentID) {
		return nil, false, domain.ErrInvalid
	}
	return s.repo.List(ctx, tournamentID, after, limit, offset)
}

func (s *Service) AssignOfficial(ctx context.Context, role, matchID, officialID string) (domain.Match, error) {
	if role != core.RoleSuperAdmin {
		return domain.Match{}, domain.ErrForbidden
	}
	matchID, officialID = strings.TrimSpace(matchID), strings.TrimSpace(officialID)
	if !core.IsUUID(matchID) || !core.IsUUID(officialID) {
		return domain.Match{}, domain.ErrInvalid
	}
	current, err := s.repo.Get(ctx, matchID)
	if err != nil {
		return domain.Match{}, err
	}
	if current.Status == domain.StatusCompleted {
		return domain.Match{}, domain.ErrConflict
	}
	roleName, err := s.users.RoleForUser(ctx, officialID)
	if errors.Is(err, domain.ErrInvalid) {
		return domain.Match{}, domain.ErrInvalid
	}
	if err != nil {
		return domain.Match{}, domain.ErrDependency
	}
	if roleName != core.RoleMatchOfficial {
		return domain.Match{}, domain.ErrInvalid
	}
	return s.repo.AssignOfficial(ctx, matchID, officialID)
}

// MarkCompleted handles a MatchFinalized event from Score; duplicates are ignored.
func (s *Service) MarkCompleted(ctx context.Context, eventID, matchID string) (bool, error) {
	eventID, matchID = strings.TrimSpace(eventID), strings.TrimSpace(matchID)
	if !core.IsUUID(eventID) || !core.IsUUID(matchID) {
		return false, domain.ErrInvalid
	}
	return s.repo.MarkCompleted(ctx, eventID, matchID)
}
