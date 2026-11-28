package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalid    = errors.New("invalid score request")
	ErrNotFound   = errors.New("score not found")
	ErrConflict   = errors.New("score is finalised")
	ErrForbidden  = errors.New("caller is not the assigned official")
	ErrDependency = errors.New("match lookup failed")
)

// MaxGoals bounds a football score; other sports' limits come from core.RulesFor.
const MaxGoals = 99

// MaxAnyScore is the highest score any sport allows (basketball); larger values are
// rejected before the match is even looked up.
const MaxAnyScore = 250

// RuleError is a score that breaks the sport's rules. Its message is shown to the official,
// and it still matches ErrInvalid for callers that only need the category.
type RuleError struct{ Reason string }

func (e RuleError) Error() string        { return e.Reason }
func (e RuleError) Is(target error) bool { return target == ErrInvalid }

type Score struct {
	MatchID              string
	HomeScore, AwayScore int32
	Finalized            bool
	UpdatedAt            time.Time
}

type Match struct {
	ID, TournamentID, HomeTeamID, AwayTeamID, OfficialID, Status string
	Sport                                                        string // from the tournament; "" means football rules
}

// MatchFinalized is the event payload Score writes to its outbox. FinalizedBy and Reason
// let Audit Log and Notification record who closed the result without calling back.
type MatchFinalized struct {
	EventID      string    `json:"event_id"`
	MatchID      string    `json:"match_id"`
	TournamentID string    `json:"tournament_id"`
	HomeTeamID   string    `json:"home_team_id"`
	AwayTeamID   string    `json:"away_team_id"`
	HomeScore    int32     `json:"home_score"`
	AwayScore    int32     `json:"away_score"`
	FinalizedAt  time.Time `json:"finalized_at"`
	FinalizedBy  string    `json:"finalized_by,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	// Sport lets Standings award points by the sport's rules; older events have none (football).
	Sport string `json:"sport,omitempty"`
}

// OutboxEvent is one unpublished row of the transactional outbox.
type OutboxEvent struct {
	EventID string
	Payload []byte
}
