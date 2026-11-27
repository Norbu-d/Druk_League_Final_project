package domain

import (
	"strconv"
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
)

type Player struct {
	ID, TeamID, GivenName, FamilyName, ShirtNumber string
	CreatedAt                                      time.Time
}

// TeamRef is what the Team service tells us about a team.
type TeamRef struct{ ID, ManagerID string }

// Normalize trims input and checks the player rules shared by every use case. The shirt
// number is optional; when given it is 1 to 99 and stored without leading zeros.
func (p Player) Normalize() (Player, error) {
	p.TeamID = strings.TrimSpace(p.TeamID)
	p.GivenName, p.FamilyName = strings.Join(strings.Fields(p.GivenName), " "), strings.Join(strings.Fields(p.FamilyName), " ")
	p.ShirtNumber = strings.TrimSpace(p.ShirtNumber)
	switch {
	case !core.IsUUID(p.TeamID):
		return p, core.Invalid("team id must be a UUID")
	case p.GivenName == "" || len(p.GivenName) > 60:
		return p, core.Invalid("given name must be 1 to 60 characters")
	case len(p.FamilyName) > 60:
		return p, core.Invalid("family name must be at most 60 characters")
	}
	if p.ShirtNumber != "" {
		n, err := strconv.Atoi(p.ShirtNumber)
		if err != nil || n < 1 || n > 99 {
			return p, core.Invalid("shirt number must be between 1 and 99")
		}
		p.ShirtNumber = strconv.Itoa(n)
	}
	return p, nil
}
