package core

import (
	"fmt"
	"strings"
)

// SportRules describes how one sport is scored and ranked. Score validates results with it;
// Standings turns final results into table points with it. Unknown sports use football rules,
// so results recorded before sports were distinguished keep their meaning.
type SportRules struct {
	Sport      string // canonical lower-case name
	Unit       string // what a score counts: goals, points, sets or set points
	MaxScore   int32  // highest score one side can have
	AllowDraw  bool   // whether a final result may be level
	WinPoints  int32
	DrawPoints int32
	LossPoints int32
	// BestOf, when set, is the number of sets in a match (volleyball: best of five), so the
	// winner must take exactly (BestOf/2)+1 sets.
	BestOf int32
}

var sportRules = map[string]SportRules{
	"football":   {Sport: "football", Unit: "goals", MaxScore: 99, AllowDraw: true, WinPoints: 3, DrawPoints: 1},
	"futsal":     {Sport: "futsal", Unit: "goals", MaxScore: 99, AllowDraw: true, WinPoints: 3, DrawPoints: 1},
	"basketball": {Sport: "basketball", Unit: "points", MaxScore: 250, WinPoints: 2, LossPoints: 1},
	"volleyball": {Sport: "volleyball", Unit: "sets", MaxScore: 3, WinPoints: 3, BestOf: 5},
	"archery":    {Sport: "archery", Unit: "points", MaxScore: 99, WinPoints: 2},
}

// RulesFor returns the rules for a sport name (case-insensitive); unknown sports get football's.
func RulesFor(sport string) SportRules {
	if rules, ok := sportRules[strings.ToLower(strings.TrimSpace(sport))]; ok {
		return rules
	}
	rules := sportRules["football"]
	rules.Sport = ""
	return rules
}

// ValidateLive checks a score that is still being played.
func (r SportRules) ValidateLive(home, away int32) error {
	if home < 0 || away < 0 || home > r.MaxScore || away > r.MaxScore {
		return fmt.Errorf("%s must be between 0 and %d", r.Unit, r.MaxScore)
	}
	return nil
}

// ValidateFinal checks a score that is about to be locked as the result.
func (r SportRules) ValidateFinal(home, away int32) error {
	if err := r.ValidateLive(home, away); err != nil {
		return err
	}
	if !r.AllowDraw && home == away {
		return fmt.Errorf("a %s match cannot end in a draw", r.name())
	}
	if r.BestOf > 0 {
		need := r.BestOf/2 + 1
		winner, loser := max(home, away), min(home, away)
		if winner != need || loser >= need {
			return fmt.Errorf("a best-of-%d %s match ends when one side wins %d sets", r.BestOf, r.name(), need)
		}
	}
	return nil
}

// Outcome converts one side's final score into a table entry: won, drawn, lost and points.
// Volleyball gives the loser of a deciding fifth set one point and the winner two.
func (r SportRules) Outcome(scored, conceded int32) (won, drawn, lost, points int32) {
	switch {
	case scored == conceded:
		return 0, 1, 0, r.DrawPoints
	case scored > conceded:
		if r.BestOf > 0 && conceded == r.BestOf/2 {
			return 1, 0, 0, r.WinPoints - 1
		}
		return 1, 0, 0, r.WinPoints
	default:
		if r.BestOf > 0 && scored == r.BestOf/2 {
			return 0, 0, 1, r.LossPoints + 1
		}
		return 0, 0, 1, r.LossPoints
	}
}

func (r SportRules) name() string {
	if r.Sport == "" {
		return "football"
	}
	return r.Sport
}
