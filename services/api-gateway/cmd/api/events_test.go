package main

import "testing"

func TestEventSubject(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "legacy finalized event", input: "MatchFinalized", want: "drukleague.match.finalized"},
		{name: "finalized event", input: "drukleague.match.finalized", want: "drukleague.match.finalized"},
		{name: "fixture event", input: "drukleague.fixtures.generated", want: "drukleague.fixtures.generated"},
		{name: "admin correction", input: "drukleague.admin.match.result.corrected", want: "drukleague.admin.match.result.corrected"},
		{name: "admin assignment", input: "drukleague.admin.match.official.assigned", want: "drukleague.admin.match.official.assigned"},
		{name: "reject wildcard", input: "drukleague.admin.>", wantErr: true},
		{name: "reject unrelated subject", input: "untrusted.subject", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eventSubject(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("eventSubject(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("eventSubject(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
