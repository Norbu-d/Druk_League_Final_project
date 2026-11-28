package grpcclients

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/services/score/internal/domain"
	"google.golang.org/grpc"
)

type fakeTournamentClient struct {
	v1.TournamentServiceClient
	sport string
	err   error
}

func (f fakeTournamentClient) GetTournament(context.Context, *v1.GetTournamentRequest, ...grpc.CallOption) (*v1.GetTournamentResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &v1.GetTournamentResponse{Tournament: &v1.Tournament{Sport: f.sport}}, nil
}

func TestTournamentReaderSport(t *testing.T) {
	if sport, err := (&TournamentReader{client: fakeTournamentClient{sport: "volleyball"}}).Sport(context.Background(), "t1"); err != nil || sport != "volleyball" {
		t.Fatalf("sport: %q %v", sport, err)
	}
	if _, err := (&TournamentReader{client: fakeTournamentClient{err: errors.New("down")}}).Sport(context.Background(), "t1"); !errors.Is(err, domain.ErrDependency) {
		t.Fatalf("lookup failure: %v", err)
	}
	if NewTournamentReader(nil) == nil {
		t.Fatal("constructor")
	}
}
