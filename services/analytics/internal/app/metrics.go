package app

import (
	"context"
	"strings"

	"github.com/drukleague/platform/services/analytics/internal/domain"
)

func (s *Service) DashboardMetrics(ctx context.Context, season, dzongkhag string) (domain.DashboardMetrics, error) {
	if s.dashboard != nil {
		return s.dashboard.DashboardMetrics(ctx, strings.TrimSpace(season), strings.TrimSpace(dzongkhag))
	}
	return s.repo.DashboardMetrics(ctx, strings.TrimSpace(season), strings.TrimSpace(dzongkhag))
}
