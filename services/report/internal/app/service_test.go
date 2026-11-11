package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/report/internal/domain"
)

const (
	requesterID  = "11111111-1111-4111-8111-111111111111"
	tournamentID = "22222222-2222-4222-8222-222222222222"
	reportID     = "33333333-3333-4333-8333-333333333333"
)

type fakeRepository struct {
	item      domain.Report
	createErr error
	getErr    error
	created   domain.Report
}

func (f *fakeRepository) Create(_ context.Context, item domain.Report) (domain.Report, error) {
	if f.createErr != nil {
		return domain.Report{}, f.createErr
	}
	item.ID, item.CreatedAt = reportID, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	f.created = item
	return item, nil
}

func (f *fakeRepository) Get(_ context.Context, _ string) (domain.Report, error) {
	if f.getErr != nil {
		return domain.Report{}, f.getErr
	}
	return f.item, nil
}

type fakeStandings struct {
	rows []domain.Standing
	err  error
}

func (f fakeStandings) Get(context.Context, string) ([]domain.Standing, error) { return f.rows, f.err }

type fixturePage struct {
	rows []domain.Fixture
	next string
	err  error
}

type fakeFixtures struct {
	pages map[string]fixturePage
	seen  []string
}

func (f *fakeFixtures) List(_ context.Context, _ string, _ int32, token string) ([]domain.Fixture, string, error) {
	f.seen = append(f.seen, token)
	page, ok := f.pages[token]
	if !ok {
		return nil, "", errors.New("unexpected fixture page token")
	}
	return page.rows, page.next, page.err
}

func testCaller() Caller { return Caller{ID: requesterID, Role: core.RoleSuperAdmin} }

func TestCreateStandingsCSVAndJSON(t *testing.T) {
	rows := []domain.Standing{{TeamID: requesterID, TeamName: "=CMD()", Dzongkhag: "Thimphu", Played: 2, Won: 1, GoalsFor: 3, GoalsAgainst: 1, GoalDifference: 2, Points: 3}}
	repo := &fakeRepository{}
	service := New(repo, fakeStandings{rows: rows}, &fakeFixtures{pages: map[string]fixturePage{}}, "")
	created, err := service.Create(context.Background(), testCaller(), domain.CreateRequest{ReportType: " STANDINGS ", TournamentID: tournamentID})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != "complete" || created.Format != "csv" || created.DownloadURL != "/api/reports/files/"+reportID || created.ContentType != "text/csv; charset=utf-8" {
		t.Fatalf("unexpected report metadata: %+v", created)
	}
	if !strings.Contains(string(created.Content), "team_name") || !strings.Contains(string(created.Content), "'=CMD()") || !strings.Contains(string(created.Content), "Thimphu") {
		t.Fatalf("CSV missing headers/values or formula protection: %s", created.Content)
	}
	if repo.created.RequesterID != requesterID || repo.created.TournamentID != tournamentID {
		t.Fatalf("wrong report ownership or tournament: %+v", repo.created)
	}
	jsonReport, err := service.Create(context.Background(), testCaller(), domain.CreateRequest{ReportType: "standings", TournamentID: tournamentID, Format: "json"})
	if err != nil || jsonReport.ContentType != "application/json" || !strings.Contains(string(jsonReport.Content), `"team_name": "=CMD()"`) {
		t.Fatalf("JSON standings report: report=%+v err=%v", jsonReport, err)
	}
}

func TestCreateFixtureJSONAcrossPages(t *testing.T) {
	kickoff := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	fixtures := &fakeFixtures{pages: map[string]fixturePage{
		"":    {rows: []domain.Fixture{{ID: reportID, TournamentID: tournamentID, HomeTeamID: requesterID, AwayTeamID: "44444444-4444-4444-8444-444444444444", Kickoff: kickoff, Round: 1}}, next: "100"},
		"100": {rows: []domain.Fixture{{ID: "55555555-5555-4555-8555-555555555555", TournamentID: tournamentID, Kickoff: kickoff, Round: 2}}},
	}}
	service := New(&fakeRepository{}, fakeStandings{}, fixtures, "https://league.test/api/reports/files/")
	report, err := service.Create(context.Background(), testCaller(), domain.CreateRequest{ReportType: "fixtures", TournamentID: tournamentID, Format: "json"})
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures.seen) != 2 || report.DownloadURL != "https://league.test/api/reports/files/"+reportID {
		t.Fatalf("pagination/download URL not applied: seen=%v url=%q", fixtures.seen, report.DownloadURL)
	}
	if strings.Count(string(report.Content), `"tournament_id"`) != 2 || !strings.Contains(string(report.Content), "2026-03-04T05:06:07Z") {
		t.Fatalf("invalid fixture JSON: %s", report.Content)
	}
	csvReport, err := service.Create(context.Background(), testCaller(), domain.CreateRequest{ReportType: "fixtures", TournamentID: tournamentID})
	if err != nil || !strings.Contains(string(csvReport.Content), "kickoff_utc") {
		t.Fatalf("fixture CSV missing columns: %s", csvReport.Content)
	}
}

func TestCreateRejectsInvalidInputsAndRole(t *testing.T) {
	service := New(&fakeRepository{}, fakeStandings{}, &fakeFixtures{pages: map[string]fixturePage{}}, "")
	cases := []struct {
		name   string
		caller Caller
		input  domain.CreateRequest
		want   error
	}{
		{name: "missing user", caller: Caller{Role: core.RoleSuperAdmin}, input: domain.CreateRequest{ReportType: "standings", TournamentID: tournamentID}, want: core.ErrForbidden},
		{name: "unsupported role", caller: Caller{ID: requesterID, Role: "guest"}, input: domain.CreateRequest{ReportType: "standings", TournamentID: tournamentID}, want: core.ErrForbidden},
		{name: "invalid tournament", caller: testCaller(), input: domain.CreateRequest{ReportType: "standings", TournamentID: "invalid"}, want: core.ErrInvalid},
		{name: "bad format", caller: testCaller(), input: domain.CreateRequest{ReportType: "standings", TournamentID: tournamentID, Format: "pdf"}, want: core.ErrInvalid},
		{name: "bad report type", caller: testCaller(), input: domain.CreateRequest{ReportType: "analytics", TournamentID: tournamentID}, want: core.ErrInvalid},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), item.caller, item.input)
			if !errors.Is(err, item.want) {
				t.Fatalf("error = %v, want %v", err, item.want)
			}
		})
	}
}

func TestCreateReturnsDependencyAndRepositoryErrors(t *testing.T) {
	dependencyErr := errors.New("standings unavailable")
	service := New(&fakeRepository{}, fakeStandings{err: dependencyErr}, &fakeFixtures{pages: map[string]fixturePage{}}, "")
	if _, err := service.Create(context.Background(), testCaller(), domain.CreateRequest{ReportType: "standings", TournamentID: tournamentID}); !errors.Is(err, dependencyErr) {
		t.Fatalf("standings error = %v", err)
	}
	repoErr := errors.New("database unavailable")
	service = New(&fakeRepository{createErr: repoErr}, fakeStandings{}, &fakeFixtures{pages: map[string]fixturePage{}}, "")
	if _, err := service.Create(context.Background(), testCaller(), domain.CreateRequest{ReportType: "standings", TournamentID: tournamentID}); !errors.Is(err, repoErr) {
		t.Fatalf("repository error = %v", err)
	}
}

func TestFixturePaginationErrors(t *testing.T) {
	service := New(&fakeRepository{}, fakeStandings{}, &fakeFixtures{pages: map[string]fixturePage{"": {next: "same"}, "same": {next: "same"}}}, "")
	if _, err := service.Create(context.Background(), testCaller(), domain.CreateRequest{ReportType: "fixtures", TournamentID: tournamentID}); !errors.Is(err, core.ErrPrecondition) {
		t.Fatalf("repeating token error = %v", err)
	}
	pageMap := make(map[string]fixturePage, maxFixturePages+1)
	pageMap[""] = fixturePage{next: "0"}
	for i := 0; i < maxFixturePages; i++ {
		pageMap[fmt.Sprint(i)] = fixturePage{next: fmt.Sprint(i + 1)}
	}
	service = New(&fakeRepository{}, fakeStandings{}, &fakeFixtures{pages: pageMap}, "")
	if _, err := service.Create(context.Background(), testCaller(), domain.CreateRequest{ReportType: "fixtures", TournamentID: tournamentID}); !errors.Is(err, core.ErrPrecondition) {
		t.Fatalf("row limit error = %v", err)
	}
}

func TestGetReportAuthorizationAndFilename(t *testing.T) {
	item := domain.Report{ID: reportID, RequesterID: requesterID, FileName: "standings-" + tournamentID + ".csv", Format: "csv", Content: []byte("data")}
	service := New(&fakeRepository{item: item}, fakeStandings{}, &fakeFixtures{pages: map[string]fixturePage{}}, "")
	owned, err := service.Get(context.Background(), Caller{ID: requesterID, Role: core.RoleViewer}, reportID)
	if err != nil || owned.DownloadURL != "/api/reports/files/"+reportID {
		t.Fatalf("owner could not read report: item=%+v err=%v", owned, err)
	}
	admin, err := service.Get(context.Background(), testCaller(), reportID)
	if err != nil || SafeFileName(admin) != item.FileName {
		t.Fatalf("admin could not read report: item=%+v err=%v", admin, err)
	}
	if _, err := service.Download(context.Background(), Caller{ID: requesterID, Role: core.RoleViewer}, reportID); err != nil {
		t.Fatalf("owner could not download report: %v", err)
	}
	if _, err := service.Get(context.Background(), Caller{ID: "55555555-5555-4555-8555-555555555555", Role: core.RoleViewer}, reportID); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("foreign user error = %v", err)
	}
	if _, err := service.Get(context.Background(), testCaller(), "invalid"); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("invalid report id error = %v", err)
	}
	if got := SafeFileName(domain.Report{FileName: "../", Format: "json"}); got != "tournament-report.json" {
		t.Fatalf("fallback filename = %q", got)
	}
}
