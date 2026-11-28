package core

import "testing"

func TestRulesForKnownAndUnknownSports(t *testing.T) {
	if r := RulesFor(" Basketball "); r.Sport != "basketball" || r.MaxScore != 250 || r.AllowDraw {
		t.Fatalf("basketball rules: %+v", r)
	}
	if r := RulesFor("kabaddi"); r.Sport != "" || r.MaxScore != 99 || !r.AllowDraw || r.WinPoints != 3 {
		t.Fatalf("unknown sports should use football rules: %+v", r)
	}
	if r := RulesFor(""); r.name() != "football" {
		t.Fatalf("empty sport should read as football, got %q", r.name())
	}
}

func TestValidateLive(t *testing.T) {
	cases := []struct {
		sport      string
		home, away int32
		ok         bool
	}{
		{"football", 3, 2, true}, {"football", 100, 0, false}, {"football", -1, 0, false},
		{"basketball", 102, 98, true}, {"basketball", 251, 0, false},
		{"volleyball", 2, 1, true}, {"volleyball", 4, 0, false},
	}
	for _, c := range cases {
		if err := RulesFor(c.sport).ValidateLive(c.home, c.away); (err == nil) != c.ok {
			t.Fatalf("%s %d-%d live: %v", c.sport, c.home, c.away, err)
		}
	}
}

func TestValidateFinal(t *testing.T) {
	cases := []struct {
		sport      string
		home, away int32
		ok         bool
	}{
		{"football", 1, 1, true}, {"futsal", 4, 4, true},
		{"basketball", 88, 88, false}, {"basketball", 90, 88, true},
		{"archery", 26, 26, false}, {"archery", 27, 21, true},
		{"volleyball", 3, 0, true}, {"volleyball", 2, 3, true}, {"volleyball", 2, 1, false}, {"volleyball", 3, 3, false},
		{"football", 120, 0, false},
	}
	for _, c := range cases {
		if err := RulesFor(c.sport).ValidateFinal(c.home, c.away); (err == nil) != c.ok {
			t.Fatalf("%s %d-%d final: %v", c.sport, c.home, c.away, err)
		}
	}
	if err := RulesFor("basketball").ValidateFinal(70, 70); err == nil || err.Error() != "a basketball match cannot end in a draw" {
		t.Fatalf("draw message: %v", err)
	}
}

func TestOutcomePoints(t *testing.T) {
	type out struct{ won, drawn, lost, points int32 }
	check := func(sport string, scored, conceded int32, want out) {
		t.Helper()
		w, d, l, p := RulesFor(sport).Outcome(scored, conceded)
		if (out{w, d, l, p}) != want {
			t.Fatalf("%s %d-%d: got %v, want %v", sport, scored, conceded, out{w, d, l, p}, want)
		}
	}
	check("football", 2, 1, out{1, 0, 0, 3})
	check("football", 1, 1, out{0, 1, 0, 1})
	check("football", 0, 2, out{0, 0, 1, 0})
	check("basketball", 90, 80, out{1, 0, 0, 2})
	check("basketball", 80, 90, out{0, 0, 1, 1})
	check("volleyball", 3, 1, out{1, 0, 0, 3})
	check("volleyball", 3, 2, out{1, 0, 0, 2})
	check("volleyball", 2, 3, out{0, 0, 1, 1})
	check("volleyball", 0, 3, out{0, 0, 1, 0})
	check("archery", 27, 21, out{1, 0, 0, 2})
	check("archery", 21, 27, out{0, 0, 1, 0})
}
