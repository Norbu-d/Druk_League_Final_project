package grpcclients

import (
	"context"
	"sort"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	venuev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/venue/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/fixture/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// maxPages stops a paging loop if a dependency keeps returning a next-page token.
const maxPages = 50

type Tournaments struct{ client v1.TournamentServiceClient }

func NewTournaments(conn grpc.ClientConnInterface) *Tournaments {
	return &Tournaments{client: v1.NewTournamentServiceClient(conn)}
}

func (t *Tournaments) Get(ctx context.Context, id string) (domain.Tournament, error) {
	response, err := t.client.GetTournament(ctx, &v1.GetTournamentRequest{Id: id})
	if err != nil {
		return domain.Tournament{}, servicekit.FromStatus(err, "tournament")
	}
	return domain.Tournament{ID: response.GetTournament().GetId(), Dzongkhag: response.GetTournament().GetDzongkhag()}, nil
}

type Teams struct{ client v1.TeamServiceClient }

func NewTeams(conn grpc.ClientConnInterface) *Teams {
	return &Teams{client: v1.NewTeamServiceClient(conn)}
}

func (t *Teams) IDsInTournament(ctx context.Context, tournamentID string) ([]string, error) {
	ids := []string{}
	token := ""
	for page := 0; page < maxPages; page++ {
		response, err := t.client.ListTeams(ctx, &v1.ListTeamsRequest{TournamentId: tournamentID, PageSize: 100, PageToken: token})
		if err != nil {
			return nil, servicekit.FromStatus(err, "teams")
		}
		for _, team := range response.GetTeams() {
			ids = append(ids, team.GetId())
		}
		if token = response.GetNextPageToken(); token == "" {
			break
		}
	}
	return ids, nil
}

type Venues struct{ client venuev1.VenueServiceClient }

func NewVenues(conn grpc.ClientConnInterface) *Venues {
	return &Venues{client: venuev1.NewVenueServiceClient(conn)}
}

// InDzongkhag returns the tournament district's venues. Venues are optional for a schedule,
// so a Venue service outage degrades to fixtures without a venue instead of failing.
func (v *Venues) InDzongkhag(ctx context.Context, dzongkhag string) ([]domain.Venue, error) {
	response, err := v.client.ListVenues(ctx, &venuev1.ListVenuesRequest{Dzongkhag: dzongkhag, PageSize: 100})
	if err != nil {
		if code := status.Code(err); code == codes.Unavailable || code == codes.DeadlineExceeded {
			return nil, nil
		}
		return nil, servicekit.FromStatus(err, "venues")
	}
	venues := make([]domain.Venue, 0, len(response.GetVenues()))
	for _, venue := range response.GetVenues() {
		venues = append(venues, domain.Venue{ID: venue.GetId(), Name: venue.GetName()})
	}
	sort.Slice(venues, func(i, j int) bool { return venues[i].Name < venues[j].Name })
	return venues, nil
}

type Matches struct{ client v1.MatchServiceClient }

func NewMatches(conn grpc.ClientConnInterface) *Matches {
	return &Matches{client: v1.NewMatchServiceClient(conn)}
}

func (m *Matches) Create(ctx context.Context, f domain.Fixture) (string, error) {
	response, err := m.client.CreateMatch(ctx, &v1.CreateMatchRequest{TournamentId: f.TournamentID, HomeTeamId: f.HomeTeamID, AwayTeamId: f.AwayTeamID, Venue: f.VenueName, Kickoff: timestamppb.New(f.Kickoff)})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			// Match names what is busy ("venue X already has a match at ..."); keep it so the
			// app can try another slot and report the last clash if none is free.
			return "", core.Conflict(status.Convert(err).Message())
		}
		return "", servicekit.FromStatus(err, "match")
	}
	return response.GetMatch().GetId(), nil
}
