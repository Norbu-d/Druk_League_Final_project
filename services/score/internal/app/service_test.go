package app

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/services/score/internal/domain"
)

const (
	matchID  = "11111111-1111-4111-8111-111111111111"
	official = "22222222-2222-4222-8222-222222222222"
	other    = "33333333-3333-4333-8333-333333333333"
)

type fakeRepo struct {
	scores    map[string]domain.Score
	finalized []string
	err       error
}

func (f *fakeRepo) UpdateLive(_ context.Context, id string, home, away int32) (domain.Score, error) {
	if f.err != nil {
		return domain.Score{}, f.err
	}
	return domain.Score{MatchID: id, HomeScore: home, AwayScore: away}, nil
}
func (f *fakeRepo) Finalize(_ context.Context, m domain.Match, eventID, by, reason string) (domain.Score, error) {
	f.finalized = append(f.finalized, eventID+"|"+by+"|"+reason)
	return domain.Score{MatchID: m.ID, Finalized: true}, nil
}
func (f *fakeRepo) Get(_ context.Context, id string) (domain.Score, error) {
	score, ok := f.scores[id]
	if !ok {
		return domain.Score{}, domain.ErrNotFound
	}
	return score, nil
}

type fakeMatches struct {
	match domain.Match
	err   error
}

func (f fakeMatches) GetMatch(context.Context, string) (domain.Match, error) { return f.match, f.err }

func assigned(status string) fakeMatches {
	return fakeMatches{match: domain.Match{ID: matchID, OfficialID: official, Status: status}}
}

func TestUpdateLiveByAssignedOfficial(t *testing.T) {
	s := New(&fakeRepo{}, assigned("scheduled"))
	score, err := s.UpdateLive(context.Background(), official, matchID, 2, 1)
	if err != nil || score.HomeScore != 2 || score.AwayScore != 1 {
		t.Fatalf("update: %+v %v", score, err)
	}
}

func TestUpdateLiveRejections(t *testing.T) {
	cases := []struct {
		name       string
		caller, id string
		home, away int32
		matches    fakeMatches
		want       error
	}{
		{"negative", official, matchID, -1, 0, assigned("scheduled"), domain.ErrInvalid},
		{"too large", official, matchID, 100, 0, assigned("scheduled"), domain.ErrInvalid},
		{"bad id", official, "x", 1, 0, assigned("scheduled"), domain.ErrInvalid},
		{"anonymous", "", matchID, 1, 0, assigned("scheduled"), domain.ErrInvalid},
		{"other official", other, matchID, 1, 0, assigned("scheduled"), domain.ErrForbidden},
		{"unassigned", official, matchID, 1, 0, fakeMatches{match: domain.Match{ID: matchID}}, domain.ErrForbidden},
		{"completed", official, matchID, 1, 0, assigned("completed"), domain.ErrConflict},
		{"missing match", official, matchID, 1, 0, fakeMatches{err: domain.ErrNotFound}, domain.ErrNotFound},
		{"match service down", official, matchID, 1, 0, fakeMatches{err: errors.New("down")}, domain.ErrDependency},
	}
	for _, c := range cases {
		if _, err := New(&fakeRepo{}, c.matches).UpdateLive(context.Background(), c.caller, c.id, c.home, c.away); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v want %v", c.name, err, c.want)
		}
	}
}

func TestFinalizeRecordsActorAndReason(t *testing.T) {
	repo := &fakeRepo{}
	s := New(repo, assigned("scheduled"))
	s.newID = func() (string, error) { return "event-1", nil }
	if _, err := s.Finalize(context.Background(), official, matchID, " full time "); err != nil {
		t.Fatal(err)
	}
	if len(repo.finalized) != 1 || repo.finalized[0] != "event-1|"+official+"|full time" {
		t.Fatalf("unexpected finalize call %v", repo.finalized)
	}
	if _, err := New(repo, assigned("completed")).Finalize(context.Background(), official, matchID, ""); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("finalized twice")
	}
	if _, err := New(repo, assigned("scheduled")).Finalize(context.Background(), other, matchID, ""); !errors.Is(err, domain.ErrForbidden) {
		t.Fatal("non-assigned official finalized")
	}
	failing := New(repo, assigned("scheduled"))
	failing.newID = func() (string, error) { return "", errors.New("no entropy") }
	if _, err := failing.Finalize(context.Background(), official, matchID, ""); err == nil {
		t.Fatal("event id failure ignored")
	}
}

func TestGet(t *testing.T) {
	repo := &fakeRepo{scores: map[string]domain.Score{matchID: {MatchID: matchID, HomeScore: 3}}}
	if score, err := New(repo, assigned("scheduled")).Get(context.Background(), matchID); err != nil || score.HomeScore != 3 {
		t.Fatalf("get: %+v %v", score, err)
	}
	empty := &fakeRepo{scores: map[string]domain.Score{}}
	if score, err := New(empty, assigned("scheduled")).Get(context.Background(), matchID); err != nil || score.MatchID != matchID || score.Finalized {
		t.Fatalf("unscored match should read as 0-0: %+v %v", score, err)
	}
	if _, err := New(empty, assigned("scheduled")).Get(context.Background(), "x"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal("bad id accepted")
	}
	if _, err := New(empty, fakeMatches{err: domain.ErrNotFound}).Get(context.Background(), matchID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("missing match not reported")
	}
	if _, err := New(empty, fakeMatches{err: errors.New("down")}).Get(context.Background(), matchID); !errors.Is(err, domain.ErrDependency) {
		t.Fatal("outage not reported")
	}
}
