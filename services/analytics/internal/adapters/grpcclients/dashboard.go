package grpcclients

import (
	"context"
	"errors"
	"fmt"
	"strings"

	playerv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/player/v1"
	registrationv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/registration/v1"
	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/services/analytics/internal/domain"
	"google.golang.org/grpc"
)

const dashboardPageSize = 100

var errDashboardCountOverflow = errors.New("dashboard metric count exceeds the supported range")

// Dashboard reads current totals from the services that own those records.
type Dashboard struct {
	tournaments   tournamentLister
	teams         teamLister
	matches       matchLister
	players       playerLister
	registrations registrationLister
}

type tournamentLister interface {
	ListTournaments(context.Context, *v1.ListTournamentsRequest, ...grpc.CallOption) (*v1.ListTournamentsResponse, error)
}

type teamLister interface {
	ListTeams(context.Context, *v1.ListTeamsRequest, ...grpc.CallOption) (*v1.ListTeamsResponse, error)
}

type matchLister interface {
	ListMatches(context.Context, *v1.ListMatchesRequest, ...grpc.CallOption) (*v1.ListMatchesResponse, error)
}

type playerLister interface {
	ListPlayers(context.Context, *playerv1.ListPlayersRequest, ...grpc.CallOption) (*playerv1.ListPlayersResponse, error)
}

type registrationLister interface {
	ListRegistrations(context.Context, *registrationv1.ListRegistrationsRequest, ...grpc.CallOption) (*registrationv1.ListRegistrationsResponse, error)
}

func NewDashboard(tournaments, teams, matches, players, registrations grpc.ClientConnInterface) *Dashboard {
	return &Dashboard{
		tournaments:   v1.NewTournamentServiceClient(tournaments),
		teams:         v1.NewTeamServiceClient(teams),
		matches:       v1.NewMatchServiceClient(matches),
		players:       playerv1.NewPlayerServiceClient(players),
		registrations: registrationv1.NewRegistrationServiceClient(registrations),
	}
}

var _ interface {
	DashboardMetrics(context.Context, string, string) (domain.DashboardMetrics, error)
} = (*Dashboard)(nil)

func (d *Dashboard) DashboardMetrics(ctx context.Context, season, dzongkhag string) (domain.DashboardMetrics, error) {
	var metrics domain.DashboardMetrics
	tournaments, err := d.listTournaments(ctx, season)
	if err != nil {
		return metrics, fmt.Errorf("list tournaments for analytics: %w", err)
	}

	teamIDs := make(map[string]struct{})
	playerIDs := make(map[string]struct{})
	venueNames := make(map[string]struct{})
	officialIDs := make(map[string]struct{})
	for _, tournament := range tournaments {
		if dzongkhag != "" && !strings.EqualFold(strings.TrimSpace(tournament.GetDzongkhag()), strings.TrimSpace(dzongkhag)) {
			continue
		}
		if err := increment(&metrics.Tournaments); err != nil {
			return metrics, err
		}
		addDistinct(venueNames, tournament.GetVenue())

		teams, err := d.listTeams(ctx, tournament.GetId())
		if err != nil {
			return metrics, fmt.Errorf("list teams for tournament %s: %w", tournament.GetId(), err)
		}
		for _, team := range teams {
			addDistinct(teamIDs, team.GetId())
		}

		approved, err := d.listRegistrations(ctx, tournament.GetId(), "approved")
		if err != nil {
			return metrics, fmt.Errorf("list approved registrations for tournament %s: %w", tournament.GetId(), err)
		}
		for _, registration := range approved {
			addDistinct(teamIDs, registration.GetTeamId())
		}

		pending, err := d.listRegistrations(ctx, tournament.GetId(), "pending")
		if err != nil {
			return metrics, fmt.Errorf("list pending registrations for tournament %s: %w", tournament.GetId(), err)
		}
		for range pending {
			if err := increment(&metrics.PendingRegistrations); err != nil {
				return metrics, err
			}
		}

		matches, err := d.listMatches(ctx, tournament.GetId())
		if err != nil {
			return metrics, fmt.Errorf("list matches for tournament %s: %w", tournament.GetId(), err)
		}
		for _, match := range matches {
			if err := increment(&metrics.Matches); err != nil {
				return metrics, err
			}
			if strings.EqualFold(strings.TrimSpace(match.GetStatus()), "completed") {
				if err := increment(&metrics.CompletedMatches); err != nil {
					return metrics, err
				}
			}
			addDistinct(teamIDs, match.GetHomeTeamId())
			addDistinct(teamIDs, match.GetAwayTeamId())
			addDistinct(venueNames, match.GetVenue())
			addDistinct(officialIDs, match.GetOfficialId())
		}
	}

	for range teamIDs {
		if err := increment(&metrics.Teams); err != nil {
			return metrics, err
		}
	}
	for teamID := range teamIDs {
		players, err := d.listPlayers(ctx, teamID)
		if err != nil {
			return metrics, fmt.Errorf("list players for team %s: %w", teamID, err)
		}
		for _, player := range players {
			addDistinct(playerIDs, player.GetId())
		}
	}
	for range playerIDs {
		if err := increment(&metrics.Players); err != nil {
			return metrics, err
		}
	}
	for range venueNames {
		if err := increment(&metrics.Venues); err != nil {
			return metrics, err
		}
	}
	for range officialIDs {
		if err := increment(&metrics.Officials); err != nil {
			return metrics, err
		}
	}
	return metrics, nil
}

func (d *Dashboard) listTournaments(ctx context.Context, season string) ([]*v1.Tournament, error) {
	return collectPages(func(token string) ([]*v1.Tournament, string, error) {
		response, err := d.tournaments.ListTournaments(ctx, &v1.ListTournamentsRequest{Season: season, PageSize: dashboardPageSize, PageToken: token})
		if err != nil {
			return nil, "", err
		}
		return response.GetTournaments(), response.GetNextPageToken(), nil
	})
}

func (d *Dashboard) listTeams(ctx context.Context, tournamentID string) ([]*v1.Team, error) {
	return collectPages(func(token string) ([]*v1.Team, string, error) {
		response, err := d.teams.ListTeams(ctx, &v1.ListTeamsRequest{TournamentId: tournamentID, PageSize: dashboardPageSize, PageToken: token})
		if err != nil {
			return nil, "", err
		}
		return response.GetTeams(), response.GetNextPageToken(), nil
	})
}

func (d *Dashboard) listMatches(ctx context.Context, tournamentID string) ([]*v1.Match, error) {
	return collectPages(func(token string) ([]*v1.Match, string, error) {
		response, err := d.matches.ListMatches(ctx, &v1.ListMatchesRequest{TournamentId: tournamentID, PageSize: dashboardPageSize, PageToken: token})
		if err != nil {
			return nil, "", err
		}
		return response.GetMatches(), response.GetNextPageToken(), nil
	})
}

func (d *Dashboard) listPlayers(ctx context.Context, teamID string) ([]*playerv1.Player, error) {
	return collectPages(func(token string) ([]*playerv1.Player, string, error) {
		response, err := d.players.ListPlayers(ctx, &playerv1.ListPlayersRequest{TeamId: teamID, PageSize: dashboardPageSize, PageToken: token})
		if err != nil {
			return nil, "", err
		}
		return response.GetPlayers(), response.GetNextPageToken(), nil
	})
}

func (d *Dashboard) listRegistrations(ctx context.Context, tournamentID, status string) ([]*registrationv1.Registration, error) {
	return collectPages(func(token string) ([]*registrationv1.Registration, string, error) {
		response, err := d.registrations.ListRegistrations(ctx, &registrationv1.ListRegistrationsRequest{
			TournamentId: tournamentID, Status: status, PageSize: dashboardPageSize, PageToken: token,
		})
		if err != nil {
			return nil, "", err
		}
		return response.GetRegistrations(), response.GetNextPageToken(), nil
	})
}

func collectPages[T any](fetch func(string) ([]T, string, error)) ([]T, error) {
	var all []T
	token := ""
	seenTokens := make(map[string]struct{})
	for {
		items, next, err := fetch(token)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if next == "" {
			return all, nil
		}
		if _, exists := seenTokens[next]; exists {
			return nil, errors.New("service returned a repeated pagination token")
		}
		seenTokens[next] = struct{}{}
		token = next
	}
}

func addDistinct(values map[string]struct{}, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		values[strings.ToLower(value)] = struct{}{}
	}
}

func increment(value *int32) error {
	if *value == int32(^uint32(0)>>1) {
		return errDashboardCountOverflow
	}
	*value = *value + 1
	return nil
}
