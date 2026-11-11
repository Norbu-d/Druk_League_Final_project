package domain

import (
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
)

// Tournament statuses. New tournaments accept registrations; there is no status-change RPC yet.
const (
	StatusRegistrationOpen = "registration_open"
	StatusInProgress       = "in_progress"
	StatusCompleted        = "completed"
)

type Tournament struct {
	ID, Name, Sport, Season, Status, Venue, Dzongkhag string
	StartsAt                                          *time.Time
}

// Normalize trims input and checks the tournament rules shared by every use case.
func (t Tournament) Normalize() (Tournament, error) {
	t.Name, t.Venue = strings.Join(strings.Fields(t.Name), " "), strings.Join(strings.Fields(t.Venue), " ")
	t.Sport, t.Season, t.Dzongkhag = strings.ToLower(strings.TrimSpace(t.Sport)), strings.TrimSpace(t.Season), strings.TrimSpace(t.Dzongkhag)
	switch {
	case len(t.Name) < 3 || len(t.Name) > 120:
		return t, core.Invalid("tournament name must be 3 to 120 characters")
	case t.Sport == "" || len(t.Sport) > 40:
		return t, core.Invalid("sport is required (up to 40 characters)")
	case len(t.Season) < 4 || len(t.Season) > 20:
		return t, core.Invalid("season must be 4 to 20 characters, for example 2026")
	case t.Dzongkhag == "":
		return t, core.Invalid("dzongkhag is required")
	case len(t.Venue) > 120:
		return t, core.Invalid("venue must be at most 120 characters")
	}
	if t.StartsAt != nil {
		utc := t.StartsAt.UTC()
		t.StartsAt = &utc
	}
	return t, nil
}
