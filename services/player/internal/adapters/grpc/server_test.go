package grpcadapter

import (
	"context"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/player/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreatePlayerRequiresTeamAndNames(t *testing.T) {
	server := New(nil)
	tests := map[string]*v1.CreatePlayerRequest{
		"empty":             {},
		"missing team":      {GivenName: "Sonam", FamilyName: "Choki"},
		"missing given":     {TeamId: "team-id", FamilyName: "Choki"},
		"missing family":    {TeamId: "team-id", GivenName: "Sonam"},
		"whitespace values": {TeamId: " ", GivenName: " ", FamilyName: " "},
	}
	for name, req := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := server.CreatePlayer(context.Background(), req); status.Code(err) != codes.InvalidArgument {
				t.Fatalf("CreatePlayer error = %v, want InvalidArgument", err)
			}
		})
	}
}

func TestUpdatePlayerRequiresIDAndNames(t *testing.T) {
	server := New(nil)
	tests := map[string]*v1.UpdatePlayerRequest{
		"empty":             {},
		"missing id":        {GivenName: "Sonam", FamilyName: "Choki"},
		"missing given":     {Id: "player-id", FamilyName: "Choki"},
		"missing family":    {Id: "player-id", GivenName: "Sonam"},
		"whitespace values": {Id: " ", GivenName: " ", FamilyName: " "},
	}
	for name, req := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := server.UpdatePlayer(context.Background(), req); status.Code(err) != codes.InvalidArgument {
				t.Fatalf("UpdatePlayer error = %v, want InvalidArgument", err)
			}
		})
	}
}

func TestListPlayersRejectsInvalidPageTokenBeforeQuery(t *testing.T) {
	server := New(nil)
	for _, token := range []string{"invalid", "-1"} {
		t.Run(token, func(t *testing.T) {
			if _, err := server.ListPlayers(context.Background(), &v1.ListPlayersRequest{PageToken: token}); status.Code(err) != codes.InvalidArgument {
				t.Fatalf("ListPlayers error = %v, want InvalidArgument", err)
			}
		})
	}
}

func TestPageDefaultsCapsAndParsesOffset(t *testing.T) {
	tests := []struct {
		name       string
		size       int32
		token      string
		wantSize   int32
		wantOffset int
	}{
		{name: "default", wantSize: 50},
		{name: "cap", size: 500, wantSize: 100},
		{name: "offset", size: 25, token: "75", wantSize: 25, wantOffset: 75},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, offset, err := page(tt.size, tt.token)
			if err != nil || size != tt.wantSize || offset != tt.wantOffset {
				t.Fatalf("page() = (%d, %d, %v), want (%d, %d, nil)", size, offset, err, tt.wantSize, tt.wantOffset)
			}
		})
	}
	for _, token := range []string{"invalid", "-1"} {
		if _, _, err := page(10, token); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("page(%q) error = %v, want InvalidArgument", token, err)
		}
	}
}
