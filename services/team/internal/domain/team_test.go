package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
)

func TestNormalize(t *testing.T) {
	team, err := Team{Name: "  Thimphu   City ", ShortCode: " thc1 ", Dzongkhag: " Thimphu "}.Normalize()
	if err != nil || team.Name != "Thimphu City" || team.ShortCode != "THC1" || team.Dzongkhag != "Thimphu" {
		t.Fatalf("got %+v %v", team, err)
	}
	for name, bad := range map[string]Team{
		"short name":     {Name: "T", ShortCode: "TC", Dzongkhag: "Paro"},
		"long name":      {Name: strings.Repeat("x", 81), ShortCode: "TC", Dzongkhag: "Paro"},
		"short code":     {Name: "Team", ShortCode: "T", Dzongkhag: "Paro"},
		"code symbols":   {Name: "Team", ShortCode: "T-C", Dzongkhag: "Paro"},
		"long code":      {Name: "Team", ShortCode: "ABCDEFGHI", Dzongkhag: "Paro"},
		"no dzongkhag":   {Name: "Team", ShortCode: "TC"},
		"bad tournament": {Name: "Team", ShortCode: "TC", Dzongkhag: "Paro", TournamentID: "x"},
	} {
		if _, err := bad.Normalize(); !errors.Is(err, core.ErrInvalid) {
			t.Fatalf("%s: got %v", name, err)
		}
	}
}
