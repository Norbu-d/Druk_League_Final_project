package domain

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
)

func teams(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("team-%02d", i)
	}
	return ids
}

var start = time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC)

// Every pair of teams meets exactly once per leg and no team plays twice on one day.
func TestScheduleInvariants(t *testing.T) {
	for n := 2; n <= 9; n++ {
		for _, format := range []string{FormatRoundRobin, FormatDoubleRoundRobin} {
			fixtures, err := Schedule("t", teams(n), nil, start, format)
			if err != nil {
				t.Fatal(err)
			}
			legs := 1
			if format == FormatDoubleRoundRobin {
				legs = 2
			}
			if want := legs * n * (n - 1) / 2; len(fixtures) != want {
				t.Fatalf("n=%d %s: %d fixtures, want %d", n, format, len(fixtures), want)
			}
			pairs := map[string]int{}
			homeAway := map[string]int{}
			busy := map[string]bool{}
			for _, f := range fixtures {
				if f.HomeTeamID == f.AwayTeamID || f.HomeTeamID == "" || f.AwayTeamID == "" {
					t.Fatalf("invalid pairing %+v", f)
				}
				a, b := f.HomeTeamID, f.AwayTeamID
				if a > b {
					a, b = b, a
				}
				pairs[a+b]++
				homeAway[f.HomeTeamID+">"+f.AwayTeamID]++
				for _, team := range []string{f.HomeTeamID, f.AwayTeamID} {
					key := team + f.Kickoff.Format("2006-01-02")
					if busy[key] {
						t.Fatalf("n=%d: %s plays twice on %s", n, team, f.Kickoff.Format("2006-01-02"))
					}
					busy[key] = true
				}
			}
			for pair, count := range pairs {
				if count != legs {
					t.Fatalf("n=%d %s: pair %s met %d times", n, format, pair, count)
				}
			}
			if format == FormatDoubleRoundRobin {
				for pairing, count := range homeAway {
					if count != 1 {
						t.Fatalf("double round robin repeats home fixture %s", pairing)
					}
				}
			}
		}
	}
}

func TestScheduleRoundsAndVenues(t *testing.T) {
	venues := []Venue{{ID: "v1", Name: "Changlimithang"}, {ID: "v2", Name: "Paro Ground"}}
	fixtures, err := Schedule("t", teams(4), venues, start, FormatRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	if fixtures[0].Round != 1 || fixtures[len(fixtures)-1].Round != 3 {
		t.Fatalf("rounds: first %d last %d", fixtures[0].Round, fixtures[len(fixtures)-1].Round)
	}
	if fixtures[0].VenueID != "v1" || fixtures[1].VenueID != "v2" || fixtures[2].VenueName != "Changlimithang" {
		t.Fatal("venues not rotated")
	}
	if !fixtures[0].Kickoff.Equal(start) || !fixtures[2].Kickoff.Equal(start.AddDate(0, 0, 7)) {
		t.Fatalf("kickoffs: %v %v", fixtures[0].Kickoff, fixtures[2].Kickoff)
	}
	again, _ := Schedule("t", []string{"team-03", "team-01", "team-00", "team-02"}, venues, start, FormatRoundRobin)
	for i := range fixtures {
		if fixtures[i].HomeTeamID != again[i].HomeTeamID || fixtures[i].AwayTeamID != again[i].AwayTeamID {
			t.Fatal("schedule must not depend on input order")
		}
	}
}

func TestLargeLeagueSpacing(t *testing.T) {
	fixtures, err := Schedule("t", teams(20), nil, start, FormatRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	// 10 fixtures a round need 10 days, so rounds are spaced 10 days apart, not 7.
	if !fixtures[10].Kickoff.Equal(start.AddDate(0, 0, 10)) {
		t.Fatalf("round 2 starts %v", fixtures[10].Kickoff)
	}
}

func TestAlternatives(t *testing.T) {
	kickoff := time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC)
	planned := Fixture{ID: "f1", VenueID: "v2", VenueName: "Lungtenphu", Kickoff: kickoff}
	slots := Alternatives(planned, []Venue{{ID: "v1", Name: "Changlimithang"}, {ID: "v2", Name: "Lungtenphu"}})
	want := []struct {
		venue string
		shift time.Duration
	}{{"v2", 0}, {"v1", 0}, {"v2", 3 * time.Hour}, {"v1", 3 * time.Hour}, {"v2", 6 * time.Hour}, {"v1", 6 * time.Hour}}
	if len(slots) != len(want) {
		t.Fatalf("got %d slots", len(slots))
	}
	for i, w := range want {
		if slots[i].VenueID != w.venue || !slots[i].Kickoff.Equal(kickoff.Add(w.shift)) || slots[i].ID != "f1" {
			t.Fatalf("slot %d: %+v", i, slots[i])
		}
	}
	if none := Alternatives(Fixture{Kickoff: kickoff}, nil); len(none) != len(SameDaySlots) || none[0].VenueID != "" {
		t.Fatalf("without venues only the kickoff moves: %+v", none)
	}
}

func TestScheduleErrors(t *testing.T) {
	if _, err := Schedule("t", teams(1), nil, start, FormatRoundRobin); !errors.Is(err, core.ErrPrecondition) {
		t.Fatalf("one team: %v", err)
	}
	if _, err := Schedule("t", []string{"a", "a"}, nil, start, FormatRoundRobin); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("duplicate team: %v", err)
	}
}

func TestFormatsAndDefaultStart(t *testing.T) {
	for input, want := range map[string]string{"": FormatRoundRobin, "league": FormatRoundRobin, "round robin": FormatRoundRobin, "home_and_away": FormatDoubleRoundRobin, FormatDoubleRoundRobin: FormatDoubleRoundRobin} {
		if got, err := NormalizeFormat(input); err != nil || got != want {
			t.Fatalf("%q: %q %v", input, got, err)
		}
	}
	if _, err := NormalizeFormat("knockout"); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("unsupported format accepted")
	}
	got := DefaultStart(time.Date(2026, 10, 19, 15, 30, 0, 0, time.UTC))
	if !got.Equal(time.Date(2026, 10, 26, 4, 0, 0, 0, time.UTC)) {
		t.Fatalf("default start %v", got)
	}
}
