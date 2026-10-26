package grpcadapter

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/player/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedPlayerServiceServer
	db *sql.DB
}

func New(db *sql.DB) *Server { return &Server{db: db} }

func (s *Server) CreatePlayer(ctx context.Context, req *v1.CreatePlayerRequest) (*v1.CreatePlayerResponse, error) {
	teamID := strings.TrimSpace(req.GetTeamId())
	given := strings.TrimSpace(req.GetGivenName())
	family := strings.TrimSpace(req.GetFamilyName())
	if teamID == "" || given == "" || family == "" {
		return nil, status.Error(codes.InvalidArgument, "team id, given name, and family name are required")
	}
	p := new(v1.Player)
	var created time.Time
	err := s.db.QueryRowContext(ctx, `INSERT INTO players(team_id,given_name,family_name,shirt_number) VALUES($1,$2,$3,$4) RETURNING id::text,team_id::text,given_name,family_name,shirt_number,created_at`, teamID, given, family, strings.TrimSpace(req.GetShirtNumber())).Scan(&p.Id, &p.TeamId, &p.GivenName, &p.FamilyName, &p.ShirtNumber, &created)
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "team id must be a UUID")
		}
		return nil, status.Error(codes.Internal, "could not create player")
	}
	p.CreatedAt = timestamppb.New(created)
	return &v1.CreatePlayerResponse{Player: p}, nil
}
func (s *Server) GetPlayer(ctx context.Context, req *v1.GetPlayerRequest) (*v1.GetPlayerResponse, error) {
	p, err := s.read(ctx, `SELECT id::text,team_id::text,given_name,family_name,shirt_number,created_at FROM players WHERE id=$1`, strings.TrimSpace(req.GetId()))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "player not found")
	}
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "player id must be a UUID")
		}
		return nil, status.Error(codes.Internal, "could not read player")
	}
	return &v1.GetPlayerResponse{Player: p}, nil
}
func (s *Server) ListPlayers(ctx context.Context, req *v1.ListPlayersRequest) (*v1.ListPlayersResponse, error) {
	size, offset, err := page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id::text,team_id::text,given_name,family_name,shirt_number,created_at FROM players WHERE ($1='' OR team_id::text=$1) ORDER BY family_name,given_name,id LIMIT $2 OFFSET $3`, strings.TrimSpace(req.GetTeamId()), size+1, offset)
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "team id must be a UUID")
		}
		return nil, status.Error(codes.Internal, "could not list players")
	}
	defer rows.Close()
	result := &v1.ListPlayersResponse{Players: make([]*v1.Player, 0, size)}
	for rows.Next() {
		p := new(v1.Player)
		var created time.Time
		if err := rows.Scan(&p.Id, &p.TeamId, &p.GivenName, &p.FamilyName, &p.ShirtNumber, &created); err != nil {
			return nil, status.Error(codes.Internal, "could not read players")
		}
		if len(result.Players) == int(size) {
			result.NextPageToken = strconv.Itoa(offset + int(size))
			break
		}
		p.CreatedAt = timestamppb.New(created)
		result.Players = append(result.Players, p)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "could not read players")
	}
	return result, nil
}
func (s *Server) UpdatePlayer(ctx context.Context, req *v1.UpdatePlayerRequest) (*v1.UpdatePlayerResponse, error) {
	id := strings.TrimSpace(req.GetId())
	given := strings.TrimSpace(req.GetGivenName())
	family := strings.TrimSpace(req.GetFamilyName())
	if id == "" || given == "" || family == "" {
		return nil, status.Error(codes.InvalidArgument, "player id, given name, and family name are required")
	}
	p := new(v1.Player)
	var created time.Time
	err := s.db.QueryRowContext(ctx, `UPDATE players SET given_name=$2,family_name=$3,shirt_number=$4 WHERE id=$1 RETURNING id::text,team_id::text,given_name,family_name,shirt_number,created_at`, id, given, family, strings.TrimSpace(req.GetShirtNumber())).Scan(&p.Id, &p.TeamId, &p.GivenName, &p.FamilyName, &p.ShirtNumber, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "player not found")
	}
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "player id must be a UUID")
		}
		return nil, status.Error(codes.Internal, "could not update player")
	}
	p.CreatedAt = timestamppb.New(created)
	return &v1.UpdatePlayerResponse{Player: p}, nil
}
func (s *Server) read(ctx context.Context, q string, args ...any) (*v1.Player, error) {
	p := new(v1.Player)
	var created time.Time
	err := s.db.QueryRowContext(ctx, q, args...).Scan(&p.Id, &p.TeamId, &p.GivenName, &p.FamilyName, &p.ShirtNumber, &created)
	if err == nil {
		p.CreatedAt = timestamppb.New(created)
	}
	return p, err
}
func page(size int32, token string) (int32, int, error) {
	if size <= 0 {
		size = 50
	}
	if size > 100 {
		size = 100
	}
	offset := 0
	if token != "" {
		n, err := strconv.Atoi(token)
		if err != nil || n < 0 {
			return 0, 0, status.Error(codes.InvalidArgument, "invalid page token")
		}
		offset = n
	}
	return size, offset, nil
}
