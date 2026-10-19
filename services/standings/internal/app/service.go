package app

import (
	"context"
	"errors"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/standings/internal/domain"
	"github.com/drukleague/platform/services/standings/internal/ports"
)

type Service struct {
	repo  ports.Repository
	teams ports.TeamDirectory
}

func New(repo ports.Repository, teams ports.TeamDirectory) *Service {
	return &Service{repo: repo, teams: teams}
}

func (s *Service) ApplyFinalized(ctx context.Context, event domain.MatchFinalized) (bool, error) {
	event.EventID = strings.TrimSpace(event.EventID)
	event.MatchID = strings.TrimSpace(event.MatchID)
	event.TournamentID = strings.TrimSpace(event.TournamentID)
	event.HomeTeamID = strings.TrimSpace(event.HomeTeamID)
	event.AwayTeamID = strings.TrimSpace(event.AwayTeamID)
	if !core.IsUUID(event.EventID) || !core.IsUUID(event.MatchID) || !core.IsUUID(event.TournamentID) || !core.IsUUID(event.HomeTeamID) || !core.IsUUID(event.AwayTeamID) || event.HomeTeamID == event.AwayTeamID || event.HomeScore < 0 || event.AwayScore < 0 {
		return false, domain.ErrInvalid
	}
	home, err := s.team(ctx, event.TournamentID, event.HomeTeamID)
	if err != nil {
		return false, err
	}
	away, err := s.team(ctx, event.TournamentID, event.AwayTeamID)
	if err != nil {
		return false, err
	}
	return s.repo.ApplyFinalized(ctx, event, home, away)
}

func (s *Service) team(ctx context.Context, tournamentID, teamID string) (domain.Team, error) {
	known, err := s.repo.TeamKnown(ctx, tournamentID, teamID)
	if err != nil {
		return domain.Team{}, err
	}
	if known {
		return domain.Team{ID: teamID}, nil
	}
	team, err := s.teams.GetTeam(ctx, teamID)
	if errors.Is(err, domain.ErrInvalid) {
		return domain.Team{}, domain.ErrInvalid
	}
	if err != nil {
		return domain.Team{}, domain.ErrDependency
	}
	if team.ID != teamID || team.TournamentID != tournamentID || strings.TrimSpace(team.Name) == "" || strings.TrimSpace(team.Dzongkhag) == "" {
		return domain.Team{}, domain.ErrInvalid
	}
	return team, nil
}

func (s *Service) Get(ctx context.Context, tournamentID string) ([]domain.Standing, error) {
	if !core.IsUUID(tournamentID) {
		return nil, domain.ErrInvalid
	}
	return s.repo.Get(ctx, strings.TrimSpace(tournamentID))
}
