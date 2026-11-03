package domain

import (
	"math"
	"sort"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
)

const (
	FormatRoundRobin       = "round_robin"
	FormatDoubleRoundRobin = "double_round_robin"
)

type Fixture struct {
	ID, TournamentID, HomeTeamID, AwayTeamID string
	VenueID, VenueName, MatchID              string
	Round                                    int32
	Kickoff                                  time.Time
}

type Tournament struct{ ID, Dzongkhag string }

type Venue struct{ ID, Name string }

// FixturesGenerated is published once per tournament after its schedule is stored.
type FixturesGenerated struct {
	EventID      string    `json:"event_id"`
	TournamentID string    `json:"tournament_id"`
	Fixtures     int       `json:"fixtures"`
	GeneratedBy  string    `json:"generated_by"`
	GeneratedAt  time.Time `json:"generated_at"`
}

// OutboxEvent is one unpublished row of the fixture outbox.
type OutboxEvent struct {
	EventID string
	Payload []byte
}

// NormalizeFormat maps the request's format to a supported one; empty means round robin.
func NormalizeFormat(format string) (string, error) {
	switch format {
	case "", FormatRoundRobin, "round robin", "league":
		return FormatRoundRobin, nil
	case FormatDoubleRoundRobin, "home_and_away":
		return FormatDoubleRoundRobin, nil
	}
	return "", core.Invalid("format must be round_robin or double_round_robin")
}

// DefaultStart is the default first kickoff: one week ahead at 10:00 Bhutan time (04:00 UTC).
func DefaultStart(now time.Time) time.Time {
	day := now.UTC().AddDate(0, 0, 7)
	return time.Date(day.Year(), day.Month(), day.Day(), 4, 0, 0, 0, time.UTC)
}

// Schedule builds a round-robin schedule with the circle method. Teams are sorted so the
// result is deterministic; an odd team count gets a bye each round. Each round spans at
// least a week and one fixture per day, so no team plays twice on the same day, and
// venues are rotated so consecutive fixtures use different grounds when possible.
func Schedule(tournamentID string, teamIDs []string, venues []Venue, start time.Time, format string) ([]Fixture, error) {
	if len(teamIDs) < 2 {
		return nil, core.Precondition("at least two teams must belong to the tournament")
	}
	seen := map[string]bool{}
	ids := make([]string, 0, len(teamIDs)+1)
	for _, id := range teamIDs {
		if seen[id] {
			return nil, core.Invalid("duplicate team in tournament")
		}
		seen[id] = true
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids)%2 == 1 {
		ids = append(ids, "") // bye
	}
	n := len(ids)
	rounds, half := n-1, n/2
	spacing := max(7, half)
	fixtures := make([]Fixture, 0, rounds*half*2)
	legs := 1
	if format == FormatDoubleRoundRobin {
		legs = 2
	}
	for leg := 0; leg < legs; leg++ {
		order := append([]string(nil), ids...)
		for round := 0; round < rounds; round++ {
			slot := 0
			for i := 0; i < half; i++ {
				home, away := order[i], order[n-1-i]
				if home == "" || away == "" {
					continue
				}
				if (round+i)%2 == 1 {
					home, away = away, home
				}
				if leg == 1 {
					home, away = away, home
				}
				absolute := leg*rounds + round
				round := absolute + 1
				if round > math.MaxInt32 {
					return nil, core.Invalid("schedule exceeds supported round range")
				}
				f := Fixture{TournamentID: tournamentID, HomeTeamID: home, AwayTeamID: away, Round: int32(round), // #nosec G115 -- round is checked against math.MaxInt32 immediately above.
					Kickoff: start.AddDate(0, 0, absolute*spacing+slot)}
				if len(venues) > 0 {
					venue := venues[len(fixtures)%len(venues)]
					f.VenueID, f.VenueName = venue.ID, venue.Name
				}
				fixtures = append(fixtures, f)
				slot++
			}
			last := order[n-1]
			copy(order[2:], order[1:n-1])
			order[1] = last
		}
	}
	return fixtures, nil
}
