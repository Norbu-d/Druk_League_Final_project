package grpcadapter

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedTeamServiceServer
	db *sql.DB
}

func New(db *sql.DB) *Server { return &Server{db: db} }

func (s *Server) CreateTeam(ctx context.Context, req *v1.CreateTeamRequest) (*v1.CreateTeamResponse, error) {
	name, code, district := strings.TrimSpace(req.GetName()), strings.ToUpper(strings.TrimSpace(req.GetShortCode())), strings.TrimSpace(req.GetDzongkhag())
	if name == "" || code == "" || district == "" {
		return nil, status.Error(codes.InvalidArgument, "name, short code, and dzongkhag are required")
	}
	var team v1.Team
	var created time.Time
	err := s.db.QueryRowContext(ctx, `INSERT INTO teams(name,short_code,dzongkhag,tournament_id) VALUES($1,$2,$3,NULLIF($4,'')::uuid) RETURNING id::text,name,short_code,dzongkhag,COALESCE(tournament_id::text,''),created_at`, name, code, district, strings.TrimSpace(req.GetTournamentId())).Scan(&team.Id, &team.Name, &team.ShortCode, &team.Dzongkhag, &team.TournamentId, &created)
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "tournament id must be a UUID")
		}
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, status.Error(codes.AlreadyExists, "team short code already exists for this tournament")
		}
		return nil, status.Error(codes.Internal, "could not create team")
	}
	team.CreatedAt = timestamppb.New(created)
	return &v1.CreateTeamResponse{Team: &team}, nil
}

func (s *Server) GetTeam(ctx context.Context, req *v1.GetTeamRequest) (*v1.GetTeamResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "team id is required")
	}
	team, err := readTeam(ctx, s.db, `SELECT id::text,name,short_code,dzongkhag,COALESCE(tournament_id::text,''),created_at FROM teams WHERE id=$1`, req.GetId())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "team not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "could not read team")
	}
	return &v1.GetTeamResponse{Team: team}, nil
}

func (s *Server) ListTeams(ctx context.Context, req *v1.ListTeamsRequest) (*v1.ListTeamsResponse, error) {
	size, offset, err := page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id::text,name,short_code,dzongkhag,COALESCE(tournament_id::text,''),created_at FROM teams WHERE ($1='' OR tournament_id::text=$1) AND ($2='' OR dzongkhag=$2) ORDER BY name,id LIMIT $3 OFFSET $4`, strings.TrimSpace(req.GetTournamentId()), strings.TrimSpace(req.GetDzongkhag()), size+1, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not list teams")
	}
	defer rows.Close()
	result := &v1.ListTeamsResponse{Teams: make([]*v1.Team, 0, size)}
	for rows.Next() {
		team := new(v1.Team)
		var created time.Time
		if err := rows.Scan(&team.Id, &team.Name, &team.ShortCode, &team.Dzongkhag, &team.TournamentId, &created); err != nil {
			return nil, status.Error(codes.Internal, "could not read teams")
		}
		if int32(len(result.Teams)) == size {
			result.NextPageToken = strconv.Itoa(offset + int(size))
			break
		}
		team.CreatedAt = timestamppb.New(created)
		result.Teams = append(result.Teams, team)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "could not read teams")
	}
	return result, nil
}

func readTeam(ctx context.Context, db *sql.DB, query string, args ...any) (*v1.Team, error) {
	team := new(v1.Team)
	var created time.Time
	err := db.QueryRowContext(ctx, query, args...).Scan(&team.Id, &team.Name, &team.ShortCode, &team.Dzongkhag, &team.TournamentId, &created)
	if err != nil {
		return nil, err
	}
	team.CreatedAt = timestamppb.New(created)
	return team, nil
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
