package app

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/report/internal/domain"
	"github.com/drukleague/platform/services/report/internal/ports"
)

const maxFixturePages = 100

type Caller struct{ ID, Role string }

type Service struct {
	repo         ports.Repository
	standings    ports.Standings
	fixtures     ports.Fixtures
	downloadBase string
}

func New(repo ports.Repository, standings ports.Standings, fixtures ports.Fixtures, downloadBase string) *Service {
	base := strings.TrimRight(strings.TrimSpace(downloadBase), "/")
	if base == "" {
		base = "/api/reports/files"
	}
	return &Service{repo: repo, standings: standings, fixtures: fixtures, downloadBase: base}
}

func (s *Service) Create(ctx context.Context, caller Caller, request domain.CreateRequest) (domain.Report, error) {
	if !core.IsUUID(caller.ID) || !validRole(caller.Role) {
		return domain.Report{}, core.Forbidden("an authenticated user is required to create reports")
	}
	tournamentID := strings.TrimSpace(request.TournamentID)
	if !core.IsUUID(tournamentID) {
		return domain.Report{}, core.Invalid("tournament id must be a UUID")
	}
	reportType := strings.ToLower(strings.TrimSpace(request.ReportType))
	format := strings.ToLower(strings.TrimSpace(request.Format))
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "json" {
		return domain.Report{}, core.Invalid("format must be csv or json")
	}

	var content []byte
	var contentType string
	var err error
	switch reportType {
	case "standings":
		var rows []domain.Standing
		rows, err = s.standings.Get(ctx, tournamentID)
		if err == nil {
			content, err = buildStandings(rows, format)
		}
	case "fixtures":
		var rows []domain.Fixture
		rows, err = s.allFixtures(ctx, tournamentID)
		if err == nil {
			content, err = buildFixtures(rows, format)
		}
	default:
		return domain.Report{}, core.Invalid("report type must be standings or fixtures")
	}
	if err != nil {
		return domain.Report{}, err
	}
	if format == "csv" {
		contentType = "text/csv; charset=utf-8"
	} else {
		contentType = "application/json"
	}
	fileName := fmt.Sprintf("%s-%s.%s", reportType, tournamentID, format)
	item, err := s.repo.Create(ctx, domain.Report{
		ReportType: reportType, Status: "complete", RequesterID: caller.ID,
		TournamentID: tournamentID, Format: format, FileName: fileName,
		ContentType: contentType, Content: content,
	})
	if err != nil {
		return domain.Report{}, err
	}
	item.DownloadURL = s.downloadBase + "/" + item.ID
	return item, nil
}

func (s *Service) Get(ctx context.Context, caller Caller, id string) (domain.Report, error) {
	id = strings.TrimSpace(id)
	if !core.IsUUID(id) {
		return domain.Report{}, core.Invalid("report id must be a UUID")
	}
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Report{}, err
	}
	if !validRole(caller.Role) || (caller.Role != core.RoleSuperAdmin && caller.ID != item.RequesterID) {
		return domain.Report{}, core.Forbidden("cannot access another user's report")
	}
	item.DownloadURL = s.downloadBase + "/" + item.ID
	return item, nil
}

func (s *Service) Download(ctx context.Context, caller Caller, id string) (domain.Report, error) {
	return s.Get(ctx, caller, id)
}

func (s *Service) allFixtures(ctx context.Context, tournamentID string) ([]domain.Fixture, error) {
	const pageSize = int32(100)
	fixtures := make([]domain.Fixture, 0)
	token := ""
	for page := 0; page < maxFixturePages; page++ {
		items, next, err := s.fixtures.List(ctx, tournamentID, pageSize, token)
		if err != nil {
			return nil, err
		}
		fixtures = append(fixtures, items...)
		if next == "" {
			return fixtures, nil
		}
		if next == token {
			return nil, core.Precondition("fixture service returned a repeating page token")
		}
		token = next
	}
	return nil, core.Precondition("fixture report exceeds the supported 10000 rows")
}

func buildStandings(rows []domain.Standing, format string) ([]byte, error) {
	if format == "json" {
		type standingExport struct {
			TeamID         string `json:"team_id"`
			TeamName       string `json:"team_name"`
			Dzongkhag      string `json:"dzongkhag"`
			Played         int32  `json:"played"`
			Won            int32  `json:"won"`
			Drawn          int32  `json:"drawn"`
			Lost           int32  `json:"lost"`
			GoalsFor       int32  `json:"goals_for"`
			GoalsAgainst   int32  `json:"goals_against"`
			GoalDifference int32  `json:"goal_difference"`
			Points         int32  `json:"points"`
		}
		out := make([]standingExport, 0, len(rows))
		for _, row := range rows {
			out = append(out, standingExport{TeamID: row.TeamID, TeamName: row.TeamName, Dzongkhag: row.Dzongkhag,
				Played: row.Played, Won: row.Won, Drawn: row.Drawn, Lost: row.Lost, GoalsFor: row.GoalsFor,
				GoalsAgainst: row.GoalsAgainst, GoalDifference: row.GoalDifference, Points: row.Points})
		}
		return json.MarshalIndent(out, "", "  ")
	}
	return buildCSV([]string{"team_id", "team_name", "dzongkhag", "played", "won", "drawn", "lost", "goals_for", "goals_against", "goal_difference", "points"}, len(rows), func(i int) []string {
		row := rows[i]
		return []string{safeCell(row.TeamID), safeCell(row.TeamName), safeCell(row.Dzongkhag), number(row.Played), number(row.Won), number(row.Drawn), number(row.Lost), number(row.GoalsFor), number(row.GoalsAgainst), number(row.GoalDifference), number(row.Points)}
	})
}

func buildFixtures(rows []domain.Fixture, format string) ([]byte, error) {
	if format == "json" {
		type fixtureExport struct {
			ID           string `json:"id"`
			TournamentID string `json:"tournament_id"`
			HomeTeamID   string `json:"home_team_id"`
			AwayTeamID   string `json:"away_team_id"`
			VenueID      string `json:"venue_id"`
			MatchID      string `json:"match_id"`
			Kickoff      string `json:"kickoff_utc"`
			Round        int32  `json:"round"`
		}
		out := make([]fixtureExport, 0, len(rows))
		for _, row := range rows {
			out = append(out, fixtureExport{ID: row.ID, TournamentID: row.TournamentID, HomeTeamID: row.HomeTeamID, AwayTeamID: row.AwayTeamID, VenueID: row.VenueID, MatchID: row.MatchID, Kickoff: row.Kickoff.UTC().Format(time.RFC3339), Round: row.Round})
		}
		return json.MarshalIndent(out, "", "  ")
	}
	return buildCSV([]string{"fixture_id", "tournament_id", "round", "kickoff_utc", "home_team_id", "away_team_id", "venue_id", "match_id"}, len(rows), func(i int) []string {
		row := rows[i]
		return []string{safeCell(row.ID), safeCell(row.TournamentID), number(row.Round), row.Kickoff.UTC().Format(time.RFC3339), safeCell(row.HomeTeamID), safeCell(row.AwayTeamID), safeCell(row.VenueID), safeCell(row.MatchID)}
	})
}

func buildCSV(header []string, count int, row func(int) []string) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	for i := 0; i < count; i++ {
		if err := writer.Write(row(i)); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func safeCell(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}

func number(value int32) string { return fmt.Sprintf("%d", value) }

func validRole(role string) bool {
	switch role {
	case core.RoleSuperAdmin, core.RoleTeamManager, core.RoleMatchOfficial, core.RoleViewer, core.RoleService:
		return true
	default:
		return false
	}
}

func SafeFileName(item domain.Report) string {
	name := path.Base(item.FileName)
	if name == "." || name == ".." || name == "/" || name == "" {
		return "tournament-report." + item.Format
	}
	return name
}
