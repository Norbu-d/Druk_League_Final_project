package app

import (
	"context"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/official/internal/domain"
	"github.com/drukleague/platform/services/official/internal/ports"
)

type Caller struct{ ID, Role string }

type Service struct {
	repo    ports.Repository
	matches ports.Matches
}

func New(repo ports.Repository, matches ports.Matches) *Service {
	return &Service{repo: repo, matches: matches}
}

// Assign makes officialUserID the official of matchID. The Match service is updated first
// because it authorises scoring; repeating the call after a partial failure is safe.
func (s *Service) Assign(ctx context.Context, caller Caller, matchID, officialUserID string) (domain.Assignment, error) {
	if caller.Role != core.RoleSuperAdmin {
		return domain.Assignment{}, core.Forbidden("only the tournament admin can assign officials")
	}
	matchID, officialUserID = strings.TrimSpace(matchID), strings.TrimSpace(officialUserID)
	if !core.IsUUID(matchID) || !core.IsUUID(officialUserID) {
		return domain.Assignment{}, core.Invalid("match id and official user id must be UUIDs")
	}
	match, err := s.matches.AssignOfficial(ctx, matchID, officialUserID)
	if err != nil {
		return domain.Assignment{}, err
	}
	return s.repo.Record(ctx, domain.Assignment{MatchID: match.ID, OfficialUserID: officialUserID, TournamentID: match.TournamentID, AssignedBy: caller.ID})
}

// List returns assignments. Officials see their own; the admin may filter by official or tournament.
func (s *Service) List(ctx context.Context, caller Caller, officialUserID, tournamentID string) ([]domain.Assignment, error) {
	officialUserID, tournamentID = strings.TrimSpace(officialUserID), strings.TrimSpace(tournamentID)
	switch caller.Role {
	case core.RoleSuperAdmin, core.RoleService:
	case core.RoleMatchOfficial:
		if officialUserID == "" {
			officialUserID = caller.ID
		}
		if officialUserID != caller.ID {
			return nil, core.Forbidden("officials can only list their own assignments")
		}
	default:
		return nil, core.Forbidden("not allowed to list official assignments")
	}
	if (officialUserID != "" && !core.IsUUID(officialUserID)) || (tournamentID != "" && !core.IsUUID(tournamentID)) {
		return nil, core.Invalid("filters must be UUIDs")
	}
	return s.repo.List(ctx, officialUserID, tournamentID)
}
