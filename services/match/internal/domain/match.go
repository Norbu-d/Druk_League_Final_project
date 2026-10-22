package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalid    = errors.New("invalid match request")
	ErrNotFound   = errors.New("match not found")
	ErrConflict   = errors.New("match scheduling conflict")
	ErrForbidden  = errors.New("operation is not permitted")
	ErrDependency = errors.New("dependent service request failed")
)

// SlotLength is how long a match occupies its venue, teams and official: about 90 minutes
// of play plus warm-up and changeover. Two matches clash when their kickoffs are closer.
const SlotLength = 2 * time.Hour

type Match struct {
	ID, TournamentID, HomeTeamID, AwayTeamID string
	Venue, Status, OfficialID                string
	Kickoff                                  time.Time
}

const (
	StatusScheduled = "scheduled"
	StatusCompleted = "completed"
)

// Clash reports what a new booking collides with, so the caller can pick another slot.
func Clash(what string, at time.Time) error {
	return fmt.Errorf("%w: %s already has a match at %s UTC", ErrConflict, what, at.UTC().Format("2006-01-02 15:04"))
}
