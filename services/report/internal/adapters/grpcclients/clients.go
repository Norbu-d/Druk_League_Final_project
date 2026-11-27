package grpcclients

import (
	"context"
	"time"

	fixturev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/fixture/v1"
	standingsv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/standings/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/report/internal/domain"
	"google.golang.org/grpc"
)

type Standings struct {
	client standingsv1.StandingsServiceClient
}
type Fixtures struct {
	client fixturev1.FixtureServiceClient
}

func NewStandings(conn grpc.ClientConnInterface) *Standings {
	return &Standings{client: standingsv1.NewStandingsServiceClient(conn)}
}

func NewFixtures(conn grpc.ClientConnInterface) *Fixtures {
	return &Fixtures{client: fixturev1.NewFixtureServiceClient(conn)}
}

func (s *Standings) Get(ctx context.Context, tournamentID string) ([]domain.Standing, error) {
	response, err := s.client.GetStandings(ctx, &standingsv1.GetStandingsRequest{TournamentId: tournamentID})
	if err != nil {
		return nil, servicekit.FromStatus(err, "standings")
	}
	rows := make([]domain.Standing, 0, len(response.GetStandings()))
	for _, item := range response.GetStandings() {
		rows = append(rows, domain.Standing{
			TeamID: item.GetTeamId(), TeamName: item.GetTeamName(), Dzongkhag: item.GetDzongkhag(),
			Played: item.GetPlayed(), Won: item.GetWon(), Drawn: item.GetDrawn(), Lost: item.GetLost(),
			GoalsFor: item.GetGoalsFor(), GoalsAgainst: item.GetGoalsAgainst(), GoalDifference: item.GetGoalDifference(), Points: item.GetPoints(),
		})
	}
	return rows, nil
}

func (f *Fixtures) List(ctx context.Context, tournamentID string, pageSize int32, pageToken string) ([]domain.Fixture, string, error) {
	response, err := f.client.ListFixtures(ctx, &fixturev1.ListFixturesRequest{TournamentId: tournamentID, PageSize: pageSize, PageToken: pageToken})
	if err != nil {
		return nil, "", servicekit.FromStatus(err, "fixtures")
	}
	rows := make([]domain.Fixture, 0, len(response.GetFixtures()))
	for _, item := range response.GetFixtures() {
		var kickoff time.Time
		if item.GetKickoff() != nil {
			kickoff = item.GetKickoff().AsTime()
		}
		rows = append(rows, domain.Fixture{
			ID: item.GetId(), TournamentID: item.GetTournamentId(), HomeTeamID: item.GetHomeTeamId(),
			AwayTeamID: item.GetAwayTeamId(), VenueID: item.GetVenueId(), MatchID: item.GetMatchId(),
			Kickoff: kickoff, Round: item.GetRound(),
		})
	}
	return rows, response.GetNextPageToken(), nil
}
