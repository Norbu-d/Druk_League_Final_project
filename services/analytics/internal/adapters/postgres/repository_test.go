package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/analytics/internal/domain"
)

func TestDashboardMetricsApplySeasonAndDzongkhagFilters(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	repo := New(db)
	rows := []struct {
		event, match, tournament, home, away, season, dzongkhag string
	}{
		{"11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "2026", "Thimphu"},
		{"11111111-1111-4111-8111-111111111112", "22222222-2222-4222-8222-222222222223", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "dddddddd-dddd-4ddd-8ddd-dddddddddddd", "2026", "Thimphu"},
		{"11111111-1111-4111-8111-111111111113", "22222222-2222-4222-8222-222222222224", "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee", "ffffffff-ffff-4fff-8fff-ffffffffffff", "99999999-9999-4999-8999-999999999999", "2025", "Paro"},
	}
	for i, row := range rows {
		_, err := db.ExecContext(context.Background(), `INSERT INTO analytics_match_facts
			(event_id,match_id,tournament_id,home_team_id,away_team_id,home_score,away_score,season,dzongkhag,finalized_at)
			VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,1,0,$6,$7,$8)`,
			row.event, row.match, row.tournament, row.home, row.away, row.season, row.dzongkhag, time.Date(2026, time.January, 1+i, 12, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("insert fixture fact %d: %v", i, err)
		}
	}

	got, err := repo.DashboardMetrics(context.Background(), "2026", "Thimphu")
	if err != nil {
		t.Fatal(err)
	}
	want := domain.DashboardMetrics{Tournaments: 1, Teams: 3, Matches: 2}
	if got != want {
		t.Fatalf("metrics=%+v want %+v", got, want)
	}
	all, err := repo.DashboardMetrics(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if all != (domain.DashboardMetrics{Tournaments: 2, Teams: 5, Matches: 3}) {
		t.Fatalf("unfiltered metrics=%+v", all)
	}
}

func TestRecordFinalizedIsIdempotent(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	repo := New(db)
	event := domain.MatchFinalized{
		EventID: "11111111-1111-4111-8111-111111111121", MatchID: "22222222-2222-4222-8222-222222222221",
		TournamentID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaa21", HomeTeamID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbb121",
		AwayTeamID: "cccccccc-cccc-4ccc-8ccc-cccccccccc21", HomeScore: 3, AwayScore: 2,
		FinalizedAt: time.Date(2026, time.February, 1, 12, 0, 0, 0, time.UTC),
	}
	tournament := domain.Tournament{Season: "2026", Dzongkhag: "Thimphu"}
	if err := repo.RecordFinalized(context.Background(), event, tournament); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordFinalized(context.Background(), event, tournament); err != nil {
		t.Fatal(err)
	}
	metrics, err := repo.DashboardMetrics(context.Background(), "2026", "Thimphu")
	if err != nil {
		t.Fatal(err)
	}
	if metrics != (domain.DashboardMetrics{Tournaments: 1, Teams: 2, Matches: 1}) {
		t.Fatalf("metrics=%+v", metrics)
	}
}
