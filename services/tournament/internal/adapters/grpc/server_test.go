package grpcadapter

import (
	"context"
	"database/sql"
	"testing"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type testScanner func(...any) error

func (f testScanner) Scan(dest ...any) error { return f(dest...) }

func TestCreateTournamentRequiresFieldsAndValidTimestamp(t *testing.T) {
	server := New(nil)
	if _, err := server.CreateTournament(context.Background(), &v1.CreateTournamentRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty request error = %v, want InvalidArgument", err)
	}
	if _, err := server.CreateTournament(context.Background(), &v1.CreateTournamentRequest{
		Name: "Cup", Sport: "Football", Season: "2026", Dzongkhag: "Thimphu",
		StartsAt: &timestamppb.Timestamp{Seconds: 253402300800},
	}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid timestamp error = %v, want InvalidArgument", err)
	}
}

func TestListTournamentsRejectsInvalidPageTokenBeforeQuery(t *testing.T) {
	server := New(nil)
	for _, token := range []string{"invalid", "-1"} {
		t.Run(token, func(t *testing.T) {
			if _, err := server.ListTournaments(context.Background(), &v1.ListTournamentsRequest{PageToken: token}); status.Code(err) != codes.InvalidArgument {
				t.Fatalf("ListTournaments error = %v, want InvalidArgument", err)
			}
		})
	}
}

func TestScanTournamentConvertsNullableStartTime(t *testing.T) {
	starts := time.Date(2026, time.November, 3, 8, 30, 0, 0, time.UTC)
	for _, tt := range []struct {
		name      string
		start     sql.NullTime
		wantStart bool
	}{
		{name: "null start"},
		{name: "start set", start: sql.NullTime{Time: starts, Valid: true}, wantStart: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			row := testScanner(func(dest ...any) error {
				*dest[0].(*string) = "tournament-id"
				*dest[1].(*string) = "Cup"
				*dest[2].(*string) = "Football"
				*dest[3].(*string) = "2026"
				*dest[4].(*string) = "registration_open"
				*dest[5].(*string) = "Stadium"
				*dest[6].(*sql.NullTime) = tt.start
				*dest[7].(*string) = "Thimphu"
				return nil
			})
			got, err := scanTournament(row)
			if err != nil {
				t.Fatalf("scanTournament() error = %v", err)
			}
			if got.GetId() != "tournament-id" || got.GetName() != "Cup" || got.GetDzongkhag() != "Thimphu" {
				t.Fatalf("scanTournament() returned unexpected fields: %+v", got)
			}
			if (got.GetStartsAt() != nil) != tt.wantStart {
				t.Fatalf("StartsAt present = %t, want %t", got.GetStartsAt() != nil, tt.wantStart)
			}
			if tt.wantStart && !got.GetStartsAt().AsTime().Equal(starts) {
				t.Fatalf("StartsAt = %v, want %v", got.GetStartsAt().AsTime(), starts)
			}
		})
	}
}

func TestPageDefaultsCapsAndParsesOffset(t *testing.T) {
	tests := []struct {
		name       string
		size       int32
		token      string
		wantSize   int32
		wantOffset int
	}{
		{name: "default", wantSize: 50},
		{name: "cap", size: 500, wantSize: 100},
		{name: "offset", size: 25, token: "75", wantSize: 25, wantOffset: 75},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, offset, err := page(tt.size, tt.token)
			if err != nil || size != tt.wantSize || offset != tt.wantOffset {
				t.Fatalf("page() = (%d, %d, %v), want (%d, %d, nil)", size, offset, err, tt.wantSize, tt.wantOffset)
			}
		})
	}
	for _, token := range []string{"invalid", "-1"} {
		if _, _, err := page(10, token); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("page(%q) error = %v, want InvalidArgument", token, err)
		}
	}
}
