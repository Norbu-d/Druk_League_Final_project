package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
)

const teamID = "11111111-1111-4111-8111-111111111111"

func TestNormalize(t *testing.T) {
	p, err := Player{TeamID: " " + teamID, GivenName: "  Tandin   Wangchuk ", ShirtNumber: " 07 "}.Normalize()
	if err != nil || p.GivenName != "Tandin Wangchuk" || p.FamilyName != "" || p.ShirtNumber != "7" || p.TeamID != teamID {
		t.Fatalf("got %+v %v", p, err)
	}
	if p, err := (Player{TeamID: teamID, GivenName: "Sonam", ShirtNumber: ""}).Normalize(); err != nil || p.ShirtNumber != "" {
		t.Fatalf("shirt number should be optional: %+v %v", p, err)
	}
	for name, bad := range map[string]Player{
		"bad team":      {TeamID: "x", GivenName: "Sonam"},
		"no given name": {TeamID: teamID, GivenName: "  "},
		"long given":    {TeamID: teamID, GivenName: strings.Repeat("x", 61)},
		"long family":   {TeamID: teamID, GivenName: "Sonam", FamilyName: strings.Repeat("x", 61)},
		"shirt zero":    {TeamID: teamID, GivenName: "Sonam", ShirtNumber: "0"},
		"shirt 100":     {TeamID: teamID, GivenName: "Sonam", ShirtNumber: "100"},
		"shirt text":    {TeamID: teamID, GivenName: "Sonam", ShirtNumber: "ten"},
	} {
		if _, err := bad.Normalize(); !errors.Is(err, core.ErrInvalid) {
			t.Fatalf("%s: got %v", name, err)
		}
	}
}
