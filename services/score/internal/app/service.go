package app

import (
	"context"
	"errors"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/score/internal/domain"
	"github.com/drukleague/platform/services/score/internal/ports"
)

type Service struct {
	repo    ports.Repository
	matches ports.MatchReader
	newID   func() (string, error)
}

func New(repo ports.Repository, matches ports.MatchReader) *Service {
	return &Service{repo: repo, matches: matches, newID: core.NewUUID}
}

func (s *Service) UpdateLive(ctx context.Context, callerID, matchID string, home, away int32) (domain.Score, error) {
	if home < 0 || away < 0 || home > domain.MaxGoals || away > domain.MaxGoals {
		return domain.Score{}, domain.ErrInvalid
	}
	match, err := s.authorizedMatch(ctx, callerID, matchID)
	if err != nil {
		return domain.Score{}, err
	}
	if match.Status == "completed" {
		return domain.Score{}, domain.ErrConflict
	}
	return s.repo.UpdateLive(ctx, match.ID, home, away)
}

func (s *Service) Finalize(ctx context.Context, callerID, matchID, reason string) (domain.Score, error) {
	match, err := s.authorizedMatch(ctx, callerID, matchID)
	if err != nil {
		return domain.Score{}, err
	}
	if match.Status == "completed" {
		return domain.Score{}, domain.ErrConflict
	}
	eventID, err := s.newID()
	if err != nil {
		return domain.Score{}, err
	}
	return s.repo.Finalize(ctx, match, eventID, strings.TrimSpace(callerID), strings.TrimSpace(reason))
}

func (s *Service) Get(ctx context.Context, matchID string) (domain.Score, error) {
	matchID = strings.TrimSpace(matchID)
	if !core.IsUUID(matchID) {
		return domain.Score{}, domain.ErrInvalid
	}
	if _, err := s.matches.GetMatch(ctx, matchID); err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrInvalid) {
			return domain.Score{}, err
		}
		return domain.Score{}, domain.ErrDependency
	}
	score, err := s.repo.Get(ctx, matchID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Score{MatchID: matchID}, nil
	}
	return score, err
}

// authorizedMatch loads the match from the Match service and checks the caller is its assigned official.
func (s *Service) authorizedMatch(ctx context.Context, callerID, matchID string) (domain.Match, error) {
	callerID, matchID = strings.TrimSpace(callerID), strings.TrimSpace(matchID)
	if callerID == "" || !core.IsUUID(matchID) {
		return domain.Match{}, domain.ErrInvalid
	}
	match, err := s.matches.GetMatch(ctx, matchID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrInvalid) {
			return domain.Match{}, err
		}
		return domain.Match{}, domain.ErrDependency
	}
	if match.OfficialID == "" || match.OfficialID != callerID {
		return domain.Match{}, domain.ErrForbidden
	}
	return match, nil
}
