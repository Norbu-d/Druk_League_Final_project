package grpcadapter

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/score/internal/app"
	"github.com/drukleague/platform/services/score/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	matchID  = "11111111-1111-4111-8111-111111111111"
	official = "22222222-2222-4222-8222-222222222222"
)

type repo struct{}

func (repo) UpdateLive(_ context.Context, id string, h, a int32) (domain.Score, error) {
	return domain.Score{MatchID: id, HomeScore: h, AwayScore: a, UpdatedAt: time.Now()}, nil
}
func (repo) Finalize(_ context.Context, m domain.Match, _, _, _ string) (domain.Score, error) {
	return domain.Score{MatchID: m.ID, Finalized: true}, nil
}
func (repo) Get(context.Context, string) (domain.Score, error) {
	return domain.Score{}, domain.ErrNotFound
}

type matches struct{}

func (matches) GetMatch(context.Context, string) (domain.Match, error) {
	return domain.Match{ID: matchID, OfficialID: official, Status: "scheduled"}, nil
}

func ctxFor(id string) context.Context {
	return servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: id, Role: "match_official"})
}

func TestScoreRPCs(t *testing.T) {
	s := New(app.New(repo{}, matches{}))
	live, err := s.UpdateLiveScore(ctxFor(official), &v1.UpdateLiveScoreRequest{MatchId: matchID, HomeScore: 2, AwayScore: 1})
	if err != nil || live.GetScore().GetHomeScore() != 2 || live.GetScore().GetUpdatedAt() == nil {
		t.Fatalf("live: %v", err)
	}
	if _, err := s.UpdateLiveScore(ctxFor("someone-else"), &v1.UpdateLiveScoreRequest{MatchId: matchID}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("wrong official: %v", err)
	}
	final, err := s.FinalizeMatch(ctxFor(official), &v1.FinalizeMatchRequest{MatchId: matchID, Reason: "full time"})
	if err != nil || !final.GetScore().GetFinalized() {
		t.Fatalf("finalize: %v", err)
	}
	if _, err := s.FinalizeMatch(ctxFor(official), &v1.FinalizeMatchRequest{MatchId: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad id: %v", err)
	}
	got, err := s.GetScore(ctxFor(official), &v1.GetScoreRequest{MatchId: matchID})
	if err != nil || got.GetScore().GetMatchId() != matchID || got.GetScore().GetUpdatedAt() != nil {
		t.Fatalf("get: %v", err)
	}
	if _, err := s.GetScore(ctxFor(official), &v1.GetScoreRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty id: %v", err)
	}
}

func TestTranslate(t *testing.T) {
	for err, code := range map[error]codes.Code{
		domain.ErrInvalid: codes.InvalidArgument, domain.ErrNotFound: codes.NotFound, domain.ErrConflict: codes.AlreadyExists,
		domain.ErrForbidden: codes.PermissionDenied, domain.ErrDependency: codes.Unavailable,
		errors.New("invalid input syntax for type uuid"): codes.InvalidArgument, errors.New("boom"): codes.Internal,
	} {
		if status.Code(translate(err)) != code {
			t.Fatalf("%v: want %s", err, code)
		}
	}
}
