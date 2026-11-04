package domain

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
)

const (
	FormatRoundRobin       = "round_robin"
	FormatDoubleRoundRobin = "double_round_robin"
	FormatLeagueKnockout   = "league_knockout"
	FormatKnockout         = "knockout"
)

type Fixture struct {
	ID, TournamentID, HomeTeamID, AwayTeamID string
	VenueID, VenueName, MatchID              string
	Round                                    int32
	Kickoff                                  time.Time
	Format                                   string
	BracketMatch                             int32
	HomeSeed, AwaySeed                       int32
}

type BracketEntry struct {
	Round, Match, Slot int32
	TeamID             string
	Seed               int32
}

type KnockoutPlan struct {
	Fixtures    []Fixture
	Byes        []BracketEntry
	BracketSize int32
	Rounds      int32
}

type MatchFinalized struct {
	EventID, MatchID, TournamentID string
	HomeTeamID, AwayTeamID         string
	HomeScore, AwayScore           int32
	FinalizedAt                    time.Time
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
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "", FormatRoundRobin, "round robin", "league":
		return FormatRoundRobin, nil
	case FormatDoubleRoundRobin, "home_and_away":
		return FormatDoubleRoundRobin, nil
	case FormatKnockout:
		return FormatKnockout, nil
	case FormatLeagueKnockout, "league+knockout", "league + knockout":
		return FormatLeagueKnockout, nil
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
	if format == FormatKnockout {
		return nil, core.Precondition("knockout schedules must be seeded from standings")
	}
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
				f := Fixture{TournamentID: tournamentID, HomeTeamID: home, AwayTeamID: away, Round: int32(round), Format: format, // #nosec G115 -- round is checked against math.MaxInt32 immediately above.
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

// KnockoutSchedule builds a seeded single-elimination bracket. The standings order is
// best to worst; byes are awarded to the highest seeds when the field is not a power of two.
func KnockoutSchedule(tournamentID string, rankedTeamIDs []string, venues []Venue, start time.Time) (KnockoutPlan, error) {
	if len(rankedTeamIDs) < 2 {
		return KnockoutPlan{}, core.Precondition("standings must contain at least two teams for a knockout bracket")
	}
	seen := make(map[string]bool, len(rankedTeamIDs))
	for _, id := range rankedTeamIDs {
		if !core.IsUUID(id) || seen[id] {
			return KnockoutPlan{}, core.Invalid("standings contain an invalid or duplicate team")
		}
		seen[id] = true
	}

	bracketSize := 1
	for bracketSize < len(rankedTeamIDs) {
		// Keep all bracket indexes representable in the int32 fields persisted by
		// the fixture service. The check also prevents bracketSize from wrapping.
		if bracketSize > math.MaxInt32/2 {
			return KnockoutPlan{}, core.Invalid("standings contain too many teams for a knockout bracket")
		}
		bracketSize *= 2
	}
	seedOrder := bracketSeedOrder(bracketSize)
	bracketSize32, ok := int32Index(bracketSize)
	if !ok {
		return KnockoutPlan{}, core.Invalid("knockout bracket exceeds supported range")
	}
	rounds32, ok := int32Index(log2(bracketSize))
	if !ok {
		return KnockoutPlan{}, core.Invalid("knockout rounds exceed supported range")
	}
	plan := KnockoutPlan{BracketSize: bracketSize32, Rounds: rounds32}
	matchCount := bracketSize / 2
	for matchIndex := 0; matchIndex < matchCount; matchIndex++ {
		seedA, seedB := seedOrder[matchIndex*2], seedOrder[matchIndex*2+1]
		teamA, teamB := seededTeam(rankedTeamIDs, seedA), seededTeam(rankedTeamIDs, seedB)
		if teamA == "" || teamB == "" {
			team, seed := teamA, seedA
			if team == "" {
				team, seed = teamB, seedB
			}
			if team != "" && plan.Rounds > 1 {
				parentMatch, ok := int32Index(matchIndex/2 + 1)
				if !ok {
					return KnockoutPlan{}, core.Invalid("knockout bracket index exceeds supported range")
				}
				slot, ok := int32Index(matchIndex%2 + 1)
				if !ok {
					return KnockoutPlan{}, core.Invalid("knockout bracket index exceeds supported range")
				}
				teamSeed, ok := int32Index(seed)
				if !ok {
					return KnockoutPlan{}, core.Invalid("knockout seed exceeds supported range")
				}
				plan.Byes = append(plan.Byes, BracketEntry{Round: 2, Match: parentMatch, Slot: slot, TeamID: team, Seed: teamSeed})
			}
			continue
		}
		kickoff := start.AddDate(0, 0, matchIndex)
		bracketMatch, ok := int32Index(matchIndex + 1)
		if !ok {
			return KnockoutPlan{}, core.Invalid("knockout bracket index exceeds supported range")
		}
		homeSeed, ok := int32Index(seedA)
		if !ok {
			return KnockoutPlan{}, core.Invalid("knockout seed exceeds supported range")
		}
		awaySeed, ok := int32Index(seedB)
		if !ok {
			return KnockoutPlan{}, core.Invalid("knockout seed exceeds supported range")
		}
		fixture := Fixture{TournamentID: tournamentID, HomeTeamID: teamA, AwayTeamID: teamB,
			Round: 1, Kickoff: kickoff, Format: FormatKnockout, BracketMatch: bracketMatch,
			HomeSeed: homeSeed, AwaySeed: awaySeed}
		if len(venues) > 0 {
			venue := venues[matchIndex%len(venues)]
			fixture.VenueID, fixture.VenueName = venue.ID, venue.Name
		}
		plan.Fixtures = append(plan.Fixtures, fixture)
	}
	return plan, nil
}

func int32Index(value int) (int32, bool) {
	if value < 0 || value > math.MaxInt32 {
		return 0, false
	}
	return int32(value), true
}

func bracketSeedOrder(size int) []int {
	order := []int{1, 2}
	for current := 2; current < size; current *= 2 {
		next := make([]int, 0, current*2)
		for _, seed := range order {
			next = append(next, seed, 2*current+1-seed)
		}
		order = next
	}
	return order
}

func seededTeam(ranked []string, seed int) string {
	if seed < 1 || seed > len(ranked) {
		return ""
	}
	return ranked[seed-1]
}

func log2(value int) int {
	result := 0
	for value > 1 {
		value /= 2
		result++
	}
	return result
}
