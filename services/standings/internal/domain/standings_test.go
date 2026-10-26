package domain

import "testing"

func TestOutcome(t *testing.T) {
	cases := []struct{ gf, ga, w, d, l, p int32 }{{2, 1, 1, 0, 0, 3}, {1, 1, 0, 1, 0, 1}, {0, 3, 0, 0, 1, 0}}
	for _, c := range cases {
		w, d, l, p := Outcome(c.gf, c.ga)
		if w != c.w || d != c.d || l != c.l || p != c.p {
			t.Fatalf("%d-%d: got %d %d %d %d", c.gf, c.ga, w, d, l, p)
		}
	}
}
