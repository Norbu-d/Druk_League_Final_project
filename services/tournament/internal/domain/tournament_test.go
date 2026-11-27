package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
)

func TestNormalize(t *testing.T) {
	bhutan := time.FixedZone("BTT", 6*60*60)
	starts := time.Date(2026, 11, 1, 10, 0, 0, 0, bhutan)
	got, err := Tournament{Name: "  National   League ", Sport: " Football ", Season: " 2026 ", Dzongkhag: " Thimphu ", Venue: " Changlimithang  Stadium ", StartsAt: &starts}.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "National League" || got.Sport != "football" || got.Season != "2026" || got.Dzongkhag != "Thimphu" || got.Venue != "Changlimithang Stadium" {
		t.Fatalf("not normalised: %+v", got)
	}
	if got.StartsAt.Location() != time.UTC || !got.StartsAt.Equal(starts) || got.StartsAt.Hour() != 4 {
		t.Fatalf("start time should be stored in UTC: %v", got.StartsAt)
	}
	valid := Tournament{Name: "League", Sport: "football", Season: "2026", Dzongkhag: "Paro"}
	change := func(f func(*Tournament)) Tournament { v := valid; f(&v); return v }
	for name, bad := range map[string]Tournament{
		"short name":   change(func(t *Tournament) { t.Name = "Cu" }),
		"long name":    change(func(t *Tournament) { t.Name = strings.Repeat("x", 121) }),
		"no sport":     change(func(t *Tournament) { t.Sport = " " }),
		"long sport":   change(func(t *Tournament) { t.Sport = strings.Repeat("x", 41) }),
		"short season": change(func(t *Tournament) { t.Season = "26" }),
		"long season":  change(func(t *Tournament) { t.Season = strings.Repeat("2", 21) }),
		"no dzongkhag": change(func(t *Tournament) { t.Dzongkhag = "" }),
		"long venue":   change(func(t *Tournament) { t.Venue = strings.Repeat("x", 121) }),
	} {
		if _, err := bad.Normalize(); !errors.Is(err, core.ErrInvalid) {
			t.Fatalf("%s: got %v", name, err)
		}
	}
}
