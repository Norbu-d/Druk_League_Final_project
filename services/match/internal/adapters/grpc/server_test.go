package grpcadapter

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/match/internal/app"
	"github.com/drukleague/platform/services/match/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	tournament = "11111111-1111-4111-8111-111111111111"
	home       = "22222222-2222-4222-8222-222222222222"
	away       = "33333333-3333-4333-8333-333333333333"
	matchID    = "55555555-5555-4555-8555-555555555555"
)

type repo struct{ err error }

func (r repo) Create(_ context.Context, m domain.Match) (domain.Match, error) {
	m.ID = matchID
	return m, r.err
}
func (r repo) Get(_ context.Context, id string) (domain.Match, error) {
	if id != matchID {
		return domain.Match{}, domain.ErrNotFound
	}
	return domain.Match{ID: matchID, Status: domain.StatusScheduled, Kickoff: time.Now()}, nil
}
func (r repo) List(_ context.Context, _ string, _ *time.Time, limit, offset int) ([]domain.Match, bool, error) {
	return []domain.Match{{ID: matchID}, {ID: matchID}}, true, r.err
}
func (r repo) AssignOfficial(_ context.Context, id, official string) (domain.Match, error) {
	return domain.Match{ID: id, OfficialID: official}, r.err
}
func (r repo) MarkCompleted(context.Context, string, string) (bool, error) { return true, nil }

type teams struct{}

func (teams) TournamentForTeam(context.Context, string) (string, error) { return tournament, nil }

type users struct{}

func (users) RoleForUser(context.Context, string) (string, error) { return "match_official", nil }

func server(r repo) *Server { return New(app.New(r, teams{}, users{})) }

func admin() context.Context {
	return servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "a", Role: "super_admin"})
}

func TestCreateMatch(t *testing.T) {
	s := server(repo{})
	response, err := s.CreateMatch(admin(), &v1.CreateMatchRequest{TournamentId: tournament, HomeTeamId: home, AwayTeamId: away, Kickoff: timestamppb.Now()})
	if err != nil || response.GetMatch().GetId() != matchID {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.CreateMatch(admin(), &v1.CreateMatchRequest{TournamentId: tournament}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing kickoff: %v", err)
	}
	if _, err := s.CreateMatch(context.Background(), &v1.CreateMatchRequest{TournamentId: tournament, HomeTeamId: home, AwayTeamId: away, Kickoff: timestamppb.Now()}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("anonymous: %v", err)
	}
	if _, err := server(repo{err: domain.ErrConflict}).CreateMatch(admin(), &v1.CreateMatchRequest{TournamentId: tournament, HomeTeamId: home, AwayTeamId: away, Kickoff: timestamppb.Now()}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("clash: %v", err)
	}
	clash := domain.Clash("venue Changlimithang", time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC))
	if _, err := server(repo{err: clash}).CreateMatch(admin(), &v1.CreateMatchRequest{TournamentId: tournament, HomeTeamId: home, AwayTeamId: away, Kickoff: timestamppb.Now()}); status.Code(err) != codes.AlreadyExists || status.Convert(err).Message() != "venue Changlimithang already has a match at 2026-11-01 04:00 UTC" {
		t.Fatalf("clash detail should reach the caller: %v", err)
	}
}

func TestGetAndListMatches(t *testing.T) {
	s := server(repo{})
	if _, err := s.GetMatch(admin(), &v1.GetMatchRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatal("empty id accepted")
	}
	if _, err := s.GetMatch(admin(), &v1.GetMatchRequest{Id: home}); status.Code(err) != codes.NotFound {
		t.Fatal("missing match not NotFound")
	}
	if response, err := s.GetMatch(admin(), &v1.GetMatchRequest{Id: matchID}); err != nil || response.GetMatch().GetStatus() != "scheduled" {
		t.Fatalf("get: %v", err)
	}
	response, err := s.ListMatches(admin(), &v1.ListMatchesRequest{PageSize: 2, After: timestamppb.Now()})
	if err != nil || len(response.GetMatches()) != 2 || response.GetNextPageToken() != "2" {
		t.Fatalf("list: %v %+v", err, response)
	}
	if _, err := s.ListMatches(admin(), &v1.ListMatchesRequest{PageToken: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatal("bad page token accepted")
	}
	if _, err := s.ListMatches(admin(), &v1.ListMatchesRequest{After: &timestamppb.Timestamp{Seconds: -1 << 62}}); status.Code(err) != codes.InvalidArgument {
		t.Fatal("invalid timestamp accepted")
	}
	if _, err := server(repo{err: errors.New("db")}).ListMatches(admin(), &v1.ListMatchesRequest{}); status.Code(err) != codes.Internal {
		t.Fatal("db error not Internal")
	}
}

func TestAssignOfficialRPC(t *testing.T) {
	response, err := server(repo{}).AssignOfficial(admin(), &v1.AssignOfficialRequest{MatchId: matchID, OfficialId: home})
	if err != nil || response.GetMatch().GetOfficialId() != home {
		t.Fatalf("assign: %v", err)
	}
}

func TestTranslate(t *testing.T) {
	for err, code := range map[error]codes.Code{
		domain.ErrInvalid: codes.InvalidArgument, domain.ErrNotFound: codes.NotFound, domain.ErrConflict: codes.AlreadyExists,
		domain.ErrForbidden: codes.PermissionDenied, domain.ErrDependency: codes.Unavailable,
		errors.New(`invalid input syntax for type uuid: "x"`): codes.InvalidArgument, errors.New("boom"): codes.Internal,
	} {
		if status.Code(translate(err)) != code {
			t.Fatalf("%v: want %s", err, code)
		}
	}
	if size, offset, err := page(500, "10"); err != nil || size != 100 || offset != 10 {
		t.Fatal("page clamp wrong")
	}
	if size, _, _ := page(0, ""); size != 50 {
		t.Fatal("default page size wrong")
	}
}
