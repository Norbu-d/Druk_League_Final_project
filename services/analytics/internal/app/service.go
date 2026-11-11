package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/analytics/internal/domain"
	"github.com/drukleague/platform/services/analytics/internal/ports"
)

type Service struct {
	repo        ports.Repository
	tournaments ports.TournamentDirectory
	dashboard   ports.DashboardMetricsSource
}

func New(repo ports.Repository, tournaments ports.TournamentDirectory, dashboard ...ports.DashboardMetricsSource) *Service {
	service := &Service{repo: repo, tournaments: tournaments}
	if len(dashboard) > 0 {
		service.dashboard = dashboard[0]
	}
	return service
}

func (s *Service) RecordFinalized(ctx context.Context, event domain.MatchFinalized) error {
	if err := event.NormalizeAndValidate(); err != nil {
		return err
	}
	tournament, err := s.tournaments.GetTournament(ctx, event.TournamentID)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrDependency, err)
	}
	tournament.ID = strings.TrimSpace(tournament.ID)
	tournament.Season = strings.TrimSpace(tournament.Season)
	tournament.Dzongkhag = strings.TrimSpace(tournament.Dzongkhag)
	if tournament.ID != event.TournamentID || !core.IsUUID(tournament.ID) || tournament.Season == "" || tournament.Dzongkhag == "" {
		return domain.ErrInvalid
	}
	return s.repo.RecordFinalized(ctx, event, tournament)
}
