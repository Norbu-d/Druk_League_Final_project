package grpcadapter

import (
	"context"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/player/v1"
	"github.com/drukleague/platform/packages/servicekit/testdb"
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

func TestPlayerCRUDAndPagination(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	server := New(db)
	ctx := context.Background()
	teamID := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

	first, err := server.CreatePlayer(ctx, &v1.CreatePlayerRequest{TeamId: teamID, GivenName: " Sonam ", FamilyName: " Choki ", ShirtNumber: " 7 "})
	if err != nil {
		t.Fatalf("CreatePlayer() error = %v", err)
	}
	if first.GetPlayer().GetGivenName() != "Sonam" || first.GetPlayer().GetFamilyName() != "Choki" || first.GetPlayer().GetShirtNumber() != "7" || first.GetPlayer().GetCreatedAt() == nil {
		t.Fatalf("CreatePlayer() returned untrimmed or incomplete player: %+v", first.GetPlayer())
	}
	second, err := server.CreatePlayer(ctx, &v1.CreatePlayerRequest{TeamId: teamID, GivenName: "Pema", FamilyName: "Wangmo"})
	if err != nil {
		t.Fatalf("CreatePlayer() second player error = %v", err)
	}

	got, err := server.GetPlayer(ctx, &v1.GetPlayerRequest{Id: first.GetPlayer().GetId()})
	if err != nil || got.GetPlayer().GetId() != first.GetPlayer().GetId() {
		t.Fatalf("GetPlayer() = (%+v, %v), want first player", got, err)
	}
	updated, err := server.UpdatePlayer(ctx, &v1.UpdatePlayerRequest{Id: first.GetPlayer().GetId(), GivenName: "Sonam D.", FamilyName: "Choki", ShirtNumber: "10"})
	if err != nil || updated.GetPlayer().GetGivenName() != "Sonam D." || updated.GetPlayer().GetShirtNumber() != "10" {
		t.Fatalf("UpdatePlayer() = (%+v, %v), want updated player", updated, err)
	}

	page1, err := server.ListPlayers(ctx, &v1.ListPlayersRequest{TeamId: teamID, PageSize: 1})
	if err != nil || len(page1.GetPlayers()) != 1 || page1.GetNextPageToken() != "1" {
		t.Fatalf("ListPlayers() first page = (%+v, %v), want one player and next token 1", page1, err)
	}
	page2, err := server.ListPlayers(ctx, &v1.ListPlayersRequest{TeamId: teamID, PageSize: 1, PageToken: page1.GetNextPageToken()})
	if err != nil || len(page2.GetPlayers()) != 1 || page2.GetNextPageToken() != "" || page2.GetPlayers()[0].GetId() != second.GetPlayer().GetId() {
		t.Fatalf("ListPlayers() second page = (%+v, %v), want second player and no next token", page2, err)
	}

	if _, err := server.GetPlayer(ctx, &v1.GetPlayerRequest{Id: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"}); status.Code(err) != codes.NotFound {
		t.Fatalf("GetPlayer() missing error = %v, want NotFound", err)
	}
	if _, err := server.GetPlayer(ctx, &v1.GetPlayerRequest{Id: "bad-id"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("GetPlayer() invalid id error = %v, want InvalidArgument", err)
	}
	if _, err := server.UpdatePlayer(ctx, &v1.UpdatePlayerRequest{Id: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", GivenName: "No", FamilyName: "Player"}); status.Code(err) != codes.NotFound {
		t.Fatalf("UpdatePlayer() missing error = %v, want NotFound", err)
	}
	if _, err := server.UpdatePlayer(ctx, &v1.UpdatePlayerRequest{Id: "bad-id", GivenName: "Bad", FamilyName: "ID"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("UpdatePlayer() invalid id error = %v, want InvalidArgument", err)
	}
	if _, err := server.CreatePlayer(ctx, &v1.CreatePlayerRequest{TeamId: "bad-id", GivenName: "Bad", FamilyName: "Team"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("CreatePlayer() invalid team error = %v, want InvalidArgument", err)
	}
}

func TestPlayerDatabaseErrors(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	if err := db.Close(); err != nil {
		t.Fatalf("close test database: %v", err)
	}
	server := New(db)
	ctx := context.Background()
	if _, err := server.CreatePlayer(ctx, &v1.CreatePlayerRequest{TeamId: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", GivenName: "A", FamilyName: "B"}); status.Code(err) != codes.Internal {
		t.Fatalf("CreatePlayer() database error = %v, want Internal", err)
	}
	if _, err := server.GetPlayer(ctx, &v1.GetPlayerRequest{Id: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}); status.Code(err) != codes.Internal {
		t.Fatalf("GetPlayer() database error = %v, want Internal", err)
	}
	if _, err := server.ListPlayers(ctx, &v1.ListPlayersRequest{}); status.Code(err) != codes.Internal {
		t.Fatalf("ListPlayers() database error = %v, want Internal", err)
	}
	if _, err := server.UpdatePlayer(ctx, &v1.UpdatePlayerRequest{Id: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", GivenName: "A", FamilyName: "B"}); status.Code(err) != codes.Internal {
		t.Fatalf("UpdatePlayer() database error = %v, want Internal", err)
	}
}
