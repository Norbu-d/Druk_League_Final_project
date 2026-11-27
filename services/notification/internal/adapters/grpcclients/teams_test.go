package grpcclients

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeTeamClient struct {
	response *v1.GetTeamResponse
	err      error
}

func (f fakeTeamClient) CreateTeam(context.Context, *v1.CreateTeamRequest, ...grpc.CallOption) (*v1.CreateTeamResponse, error) {
	return nil, errors.New("unused")
}
func (f fakeTeamClient) GetTeam(context.Context, *v1.GetTeamRequest, ...grpc.CallOption) (*v1.GetTeamResponse, error) {
	return f.response, f.err
}
func (f fakeTeamClient) ListTeams(context.Context, *v1.ListTeamsRequest, ...grpc.CallOption) (*v1.ListTeamsResponse, error) {
	return nil, errors.New("unused")
}

func TestTeamDirectoryGet(t *testing.T) {
	client := &TeamDirectory{client: fakeTeamClient{response: &v1.GetTeamResponse{Team: &v1.Team{Id: "11111111-1111-4111-8111-111111111111", Name: "Thimphu FC", Dzongkhag: "Thimphu", ManagerId: "22222222-2222-4222-8222-222222222222"}}}}
	team, err := client.Get(context.Background(), "11111111-1111-4111-8111-111111111111")
	if err != nil || team.ManagerID != "22222222-2222-4222-8222-222222222222" || team.Name != "Thimphu FC" {
		t.Fatalf("team lookup: team=%+v err=%v", team, err)
	}
	client.client = fakeTeamClient{err: status.Error(codes.Unavailable, "team offline")}
	if _, err := client.Get(context.Background(), "11111111-1111-4111-8111-111111111111"); !errors.Is(err, core.ErrDependency) {
		t.Fatalf("unavailable team service error = %v", err)
	}
	client.client = fakeTeamClient{}
	if _, err := client.Get(context.Background(), "11111111-1111-4111-8111-111111111111"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("missing team response error = %v", err)
	}
}
