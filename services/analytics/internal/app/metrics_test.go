package app

import (
	"context"
	"testing"

	"github.com/drukleague/platform/services/analytics/internal/domain"
)

type metricsRepository struct {
	season, dzongkhag string
	metrics           domain.DashboardMetrics
}

func (r *metricsRepository) RecordFinalized(context.Context, domain.MatchFinalized, domain.Tournament) error {
	return nil
}

func (r *metricsRepository) DashboardMetrics(_ context.Context, season, dzongkhag string) (domain.DashboardMetrics, error) {
	r.season, r.dzongkhag = season, dzongkhag
	return r.metrics, nil
}

type metricsTournamentDirectory struct{}

func (metricsTournamentDirectory) GetTournament(context.Context, string) (domain.Tournament, error) {
	return domain.Tournament{}, nil
}

type dashboardMetricsSource struct {
	season, dzongkhag string
	metrics           domain.DashboardMetrics
}

func (s *dashboardMetricsSource) DashboardMetrics(_ context.Context, season, dzongkhag string) (domain.DashboardMetrics, error) {
	s.season, s.dzongkhag = season, dzongkhag
	return s.metrics, nil
}

func TestDashboardMetricsTrimsFiltersAndReturnsProjection(t *testing.T) {
	repository := &metricsRepository{metrics: domain.DashboardMetrics{Tournaments: 2, Teams: 7, Matches: 12}}
	service := New(repository, metricsTournamentDirectory{})
	got, err := service.DashboardMetrics(context.Background(), " 2026 ", " Thimphu ")
	if err != nil {
		t.Fatal(err)
	}
	if repository.season != "2026" || repository.dzongkhag != "Thimphu" {
		t.Fatalf("filters were not normalized: season=%q dzongkhag=%q", repository.season, repository.dzongkhag)
	}
	if got != repository.metrics {
		t.Fatalf("metrics=%+v want %+v", got, repository.metrics)
	}
}

func TestDashboardMetricsUsesCurrentServiceSource(t *testing.T) {
	source := &dashboardMetricsSource{metrics: domain.DashboardMetrics{
		Tournaments: 1, Teams: 2, Players: 3, Matches: 4, CompletedMatches: 2,
		Venues: 1, Officials: 2, PendingRegistrations: 1,
	}}
	service := New(&metricsRepository{}, metricsTournamentDirectory{}, source)
	got, err := service.DashboardMetrics(context.Background(), " 2026 ", " Thimphu ")
	if err != nil {
		t.Fatal(err)
	}
	if source.season != "2026" || source.dzongkhag != "Thimphu" {
		t.Fatalf("filters were not normalized: season=%q dzongkhag=%q", source.season, source.dzongkhag)
	}
	if got != source.metrics {
		t.Fatalf("metrics=%+v want %+v", got, source.metrics)
	}
}
