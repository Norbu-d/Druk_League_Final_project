package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
)

// TournamentOpen is the only tournament status a team manager may enter a team into.
const TournamentOpen = "registration_open"

var shortCodePattern = regexp.MustCompile(`^[A-Z0-9]{2,8}$`)

type Team struct {
	ID, Name, ShortCode, Dzongkhag string
	TournamentID, ManagerID        string
	CreatedAt                      time.Time
}

// TournamentRef is what the Tournament service tells us about a tournament.
type TournamentRef struct{ ID, Status string }

// Normalize trims input and checks the team rules shared by every use case.
func (t Team) Normalize() (Team, error) {
	t.Name = strings.Join(strings.Fields(t.Name), " ")
	t.ShortCode = strings.ToUpper(strings.TrimSpace(t.ShortCode))
	t.Dzongkhag, t.TournamentID = strings.TrimSpace(t.Dzongkhag), strings.TrimSpace(t.TournamentID)
	switch {
	case len(t.Name) < 2 || len(t.Name) > 80:
		return t, core.Invalid("team name must be 2 to 80 characters")
	case !shortCodePattern.MatchString(t.ShortCode):
		return t, core.Invalid("short code must be 2 to 8 letters or digits")
	case t.Dzongkhag == "":
		return t, core.Invalid("dzongkhag is required")
	case t.TournamentID != "" && !core.IsUUID(t.TournamentID):
		return t, core.Invalid("tournament id must be a UUID")
	}
	return t, nil
}

// Filter narrows ListTeams; empty fields match everything.
type Filter struct{ TournamentID, Dzongkhag, ManagerID string }
