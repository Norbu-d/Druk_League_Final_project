package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
)

var (
	ErrInvalid    = errors.New("invalid finalized match event")
	ErrDependency = errors.New("analytics dependency failed")
)

type MatchFinalized struct {
	EventID      string    `json:"event_id"`
	MatchID      string    `json:"match_id"`
	TournamentID string    `json:"tournament_id"`
	HomeTeamID   string    `json:"home_team_id"`
	AwayTeamID   string    `json:"away_team_id"`
	HomeScore    int32     `json:"home_score"`
	AwayScore    int32     `json:"away_score"`
	FinalizedAt  time.Time `json:"finalized_at"`
}

type Tournament struct {
	ID        string
	Season    string
	Dzongkhag string
}

// DashboardMetrics contains dashboard totals from the owning services.
type DashboardMetrics struct {
	Tournaments          int32
	Teams                int32
	Players              int32
	Matches              int32
	CompletedMatches     int32
	Venues               int32
	Officials            int32
	PendingRegistrations int32
}

func (e *MatchFinalized) NormalizeAndValidate() error {
	e.EventID = strings.TrimSpace(e.EventID)
	e.MatchID = strings.TrimSpace(e.MatchID)
	e.TournamentID = strings.TrimSpace(e.TournamentID)
	e.HomeTeamID = strings.TrimSpace(e.HomeTeamID)
	e.AwayTeamID = strings.TrimSpace(e.AwayTeamID)
	if !core.IsUUID(e.EventID) || !core.IsUUID(e.MatchID) || !core.IsUUID(e.TournamentID) ||
		!core.IsUUID(e.HomeTeamID) || !core.IsUUID(e.AwayTeamID) || e.HomeTeamID == e.AwayTeamID ||
		e.HomeScore < 0 || e.HomeScore > 99 || e.AwayScore < 0 || e.AwayScore > 99 || e.FinalizedAt.IsZero() {
		return ErrInvalid
	}
	e.FinalizedAt = e.FinalizedAt.UTC()
	return nil
}
