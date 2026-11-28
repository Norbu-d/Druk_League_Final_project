package domain

import (
	"errors"

	"github.com/drukleague/platform/packages/servicekit/core"
)

var (
	ErrInvalid    = errors.New("invalid finalized match event")
	ErrDependency = errors.New("team lookup failed")
)

type MatchFinalized struct {
	EventID      string `json:"event_id"`
	MatchID      string `json:"match_id"`
	TournamentID string `json:"tournament_id"`
	HomeTeamID   string `json:"home_team_id"`
	AwayTeamID   string `json:"away_team_id"`
	HomeScore    int32  `json:"home_score"`
	AwayScore    int32  `json:"away_score"`
	// Sport selects the points system; events written before sports were tracked have none
	// and keep football's 3/1/0.
	Sport string `json:"sport,omitempty"`
}

type Team struct{ ID, Name, Dzongkhag, TournamentID string }
type Standing struct {
	TeamID, TeamName, Dzongkhag                    string
	Played, Won, Drawn, Lost                       int32
	GoalsFor, GoalsAgainst, GoalDifference, Points int32
}

// Outcome converts one side's goals into a football standings delta: 3 points for a win, 1 for a draw.
func Outcome(goalsFor, goalsAgainst int32) (won, drawn, lost, points int32) {
	return OutcomeFor("", goalsFor, goalsAgainst)
}

// OutcomeFor converts one side's final score into a standings delta using the sport's
// points system (for example basketball: 2 for a win and 1 for a loss).
func OutcomeFor(sport string, scored, conceded int32) (won, drawn, lost, points int32) {
	return core.RulesFor(sport).Outcome(scored, conceded)
}
