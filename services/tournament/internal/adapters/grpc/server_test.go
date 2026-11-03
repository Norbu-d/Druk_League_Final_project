package grpcadapter

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/testdb"
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

func TestTournamentCRUDAndPagination(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	server := New(db)
	ctx := context.Background()
	starts := time.Date(2026, time.November, 3, 8, 30, 0, 0, time.UTC)
	first, err := server.CreateTournament(ctx, &v1.CreateTournamentRequest{Name: " Cup ", Sport: " Football ", Season: " 2026 ", Venue: " Stadium ", Dzongkhag: " Thimphu ", StartsAt: timestamppb.New(starts)})
	if err != nil {
		t.Fatalf("CreateTournament() error = %v", err)
	}
	if first.GetTournament().GetName() != "Cup" || first.GetTournament().GetSport() != "Football" || first.GetTournament().GetVenue() != "Stadium" || first.GetTournament().GetStartsAt() == nil || !first.GetTournament().GetStartsAt().AsTime().Equal(starts) {
		t.Fatalf("CreateTournament() returned incomplete tournament: %+v", first.GetTournament())
	}
	second, err := server.CreateTournament(ctx, &v1.CreateTournamentRequest{Name: "League", Sport: "Football", Season: "2026", Dzongkhag: "Paro"})
	if err != nil {
		t.Fatalf("CreateTournament() second error = %v", err)
	}

	got, err := server.GetTournament(ctx, &v1.GetTournamentRequest{Id: first.GetTournament().GetId()})
	if err != nil || got.GetTournament().GetName() != "Cup" {
		t.Fatalf("GetTournament() = (%+v, %v), want Cup", got, err)
	}
	if _, err := server.GetTournament(ctx, &v1.GetTournamentRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("GetTournament() empty id error = %v, want InvalidArgument", err)
	}
	if _, err := server.GetTournament(ctx, &v1.GetTournamentRequest{Id: "bad-id"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("GetTournament() invalid id error = %v, want InvalidArgument", err)
	}
	if _, err := server.GetTournament(ctx, &v1.GetTournamentRequest{Id: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"}); status.Code(err) != codes.NotFound {
		t.Fatalf("GetTournament() missing id error = %v, want NotFound", err)
	}

	page1, err := server.ListTournaments(ctx, &v1.ListTournamentsRequest{Season: "2026", PageSize: 1})
	if err != nil || len(page1.GetTournaments()) != 1 || page1.GetNextPageToken() != "1" {
		t.Fatalf("ListTournaments() first page = (%+v, %v), want one tournament and next token 1", page1, err)
	}
	page2, err := server.ListTournaments(ctx, &v1.ListTournamentsRequest{Season: "2026", PageSize: 1, PageToken: page1.GetNextPageToken()})
	if err != nil || len(page2.GetTournaments()) != 1 || page2.GetNextPageToken() != "" || page2.GetTournaments()[0].GetId() != second.GetTournament().GetId() {
		t.Fatalf("ListTournaments() second page = (%+v, %v), want second tournament and no next token", page2, err)
	}
	defaultPage, err := server.ListTournaments(ctx, &v1.ListTournamentsRequest{Season: "2026"})
	if err != nil || len(defaultPage.GetTournaments()) != 2 {
		t.Fatalf("ListTournaments() default page = (%+v, %v), want both tournaments", defaultPage, err)
	}
	cappedPage, err := server.ListTournaments(ctx, &v1.ListTournamentsRequest{Season: "2026", PageSize: 500})
	if err != nil || len(cappedPage.GetTournaments()) != 2 {
		t.Fatalf("ListTournaments() capped page = (%+v, %v), want both tournaments", cappedPage, err)
	}
}

func TestTournamentDatabaseErrors(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	if err := db.Close(); err != nil {
		t.Fatalf("close test database: %v", err)
	}
	server := New(db)
	ctx := context.Background()
	if _, err := server.CreateTournament(ctx, &v1.CreateTournamentRequest{Name: "Cup", Sport: "Football", Season: "2026", Dzongkhag: "Thimphu"}); status.Code(err) != codes.Internal {
		t.Fatalf("CreateTournament() database error = %v, want Internal", err)
	}
	if _, err := server.GetTournament(ctx, &v1.GetTournamentRequest{Id: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}); status.Code(err) != codes.Internal {
		t.Fatalf("GetTournament() database error = %v, want Internal", err)
	}
	if _, err := server.ListTournaments(ctx, &v1.ListTournamentsRequest{}); status.Code(err) != codes.Internal {
		t.Fatalf("ListTournaments() database error = %v, want Internal", err)
	}
}

func TestScanTournamentPropagatesScanError(t *testing.T) {
	want := errors.New("scan failed")
	if _, err := scanTournament(testScanner(func(...any) error { return want })); !errors.Is(err, want) {
		t.Fatalf("scanTournament() error = %v, want %v", err, want)
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
