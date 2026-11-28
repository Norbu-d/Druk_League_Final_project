package app

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/services/score/internal/domain"
)

type fakeTournaments struct {
	sport string
	err   error
}

func (f fakeTournaments) Sport(context.Context, string) (string, error) { return f.sport, f.err }

func sportMatch(sport string) (*Service, *fakeRepo) {
	repo := &fakeRepo{scores: map[string]domain.Score{}}
	matches := fakeMatches{match: domain.Match{ID: matchID, TournamentID: "t1", OfficialID: official, Status: "live"}}
	return New(repo, matches).WithTournaments(fakeTournaments{sport: sport}), repo
}

func TestBasketballAllowsHighScoresButNoDraw(t *testing.T) {
	s, repo := sportMatch("basketball")
	if _, err := s.UpdateLive(context.Background(), official, matchID, 102, 98); err != nil {
		t.Fatalf("102-98 is a normal basketball score: %v", err)
	}
	repo.scores[matchID] = domain.Score{MatchID: matchID, HomeScore: 88, AwayScore: 88}
	_, err := s.Finalize(context.Background(), official, matchID, "")
	var rule domain.RuleError
	if !errors.As(err, &rule) || rule.Reason != "a basketball match cannot end in a draw" || !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("a basketball draw must be refused with a reason: %v", err)
	}
	if len(repo.finalized) != 0 {
		t.Fatal("a refused result must not be finalised")
	}
}

func TestFootballStillRejectsOver99(t *testing.T) {
	s, _ := sportMatch("football")
	if _, err := s.UpdateLive(context.Background(), official, matchID, 102, 0); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("102 goals is not a football score: %v", err)
	}
	if _, err := s.UpdateLive(context.Background(), official, matchID, 300, 0); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("300 is above every sport's limit: %v", err)
	}
}

func TestVolleyballMustEndAtThreeSets(t *testing.T) {
	s, repo := sportMatch("volleyball")
	repo.scores[matchID] = domain.Score{MatchID: matchID, HomeScore: 2, AwayScore: 1}
	if _, err := s.Finalize(context.Background(), official, matchID, ""); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("2-1 is not a finished volleyball match: %v", err)
	}
	repo.scores[matchID] = domain.Score{MatchID: matchID, HomeScore: 3, AwayScore: 2}
	if _, err := s.Finalize(context.Background(), official, matchID, "full time"); err != nil || len(repo.finalized) != 1 {
		t.Fatalf("3-2 is a finished volleyball match: %v", err)
	}
}

func TestSportLookupFailureIsADependencyError(t *testing.T) {
	repo := &fakeRepo{}
	matches := fakeMatches{match: domain.Match{ID: matchID, TournamentID: "t1", OfficialID: official, Status: "live"}}
	s := New(repo, matches).WithTournaments(fakeTournaments{err: errors.New("down")})
	if _, err := s.UpdateLive(context.Background(), official, matchID, 1, 0); !errors.Is(err, domain.ErrDependency) {
		t.Fatalf("an unknown sport must not be guessed: %v", err)
	}
}
