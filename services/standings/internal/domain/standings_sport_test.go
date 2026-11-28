package domain

import "testing"

func TestOutcomeForUsesTheSportsPointsSystem(t *testing.T) {
	cases := []struct {
		sport                    string
		scored, conceded         int32
		won, drawn, lost, points int32
	}{
		{"", 2, 1, 1, 0, 0, 3}, {"football", 1, 1, 0, 1, 0, 1},
		{"basketball", 80, 90, 0, 0, 1, 1}, {"basketball", 90, 80, 1, 0, 0, 2},
		{"volleyball", 3, 2, 1, 0, 0, 2}, {"volleyball", 2, 3, 0, 0, 1, 1}, {"volleyball", 3, 0, 1, 0, 0, 3},
		{"archery", 25, 20, 1, 0, 0, 2},
	}
	for _, c := range cases {
		w, d, l, p := OutcomeFor(c.sport, c.scored, c.conceded)
		if w != c.won || d != c.drawn || l != c.lost || p != c.points {
			t.Fatalf("%s %d-%d: got %d/%d/%d %dpts", c.sport, c.scored, c.conceded, w, d, l, p)
		}
	}
}
