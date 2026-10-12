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
	v1.UnimplementedMatchServiceServer
	db *sql.DB
}

func New(db *sql.DB) *Server { return &Server{db: db} }
func (s *Server) CreateMatch(ctx context.Context, req *v1.CreateMatchRequest) (*v1.CreateMatchResponse, error) {
	tournament, home, away := strings.TrimSpace(req.GetTournamentId()), strings.TrimSpace(req.GetHomeTeamId()), strings.TrimSpace(req.GetAwayTeamId())
	venue := strings.TrimSpace(req.GetVenue())
	if tournament == "" || home == "" || away == "" || req.GetKickoff() == nil {
		return nil, status.Error(codes.InvalidArgument, "tournament, teams, and kickoff are required")
	}
	if home == away {
		return nil, status.Error(codes.InvalidArgument, "a team cannot play itself")
	}
	if err := req.GetKickoff().CheckValid(); err != nil {
		return nil, status.Error(codes.InvalidArgument, "kickoff is invalid")
	}
	kickoff := req.GetKickoff().AsTime()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not schedule match")
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, kickoff.UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, status.Error(codes.Internal, "could not lock kickoff slot")
	}
	var clash bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM matches WHERE kickoff=$1 AND (home_team_id IN ($2,$3) OR away_team_id IN ($2,$3) OR ($4<>'' AND venue=$4)))`, kickoff, home, away, venue).Scan(&clash)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not check scheduling conflicts")
	}
	if clash {
		return nil, status.Error(codes.AlreadyExists, "team or venue already has a match at this kickoff")
	}
	m := new(v1.Match)
	var start time.Time
	err = tx.QueryRowContext(ctx, `INSERT INTO matches(tournament_id,home_team_id,away_team_id,venue,kickoff) VALUES($1,$2,$3,$4,$5) RETURNING id::text,tournament_id::text,home_team_id::text,away_team_id::text,venue,kickoff,status,COALESCE(official_id::text,'')`, tournament, home, away, venue, kickoff).Scan(&m.Id, &m.TournamentId, &m.HomeTeamId, &m.AwayTeamId, &m.Venue, &start, &m.Status, &m.OfficialId)
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "ids must be UUIDs")
		}
		return nil, status.Error(codes.Internal, "could not create match")
	}
	if err = tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "could not commit match")
	}
	m.Kickoff = timestamppb.New(start)
	return &v1.CreateMatchResponse{Match: m}, nil
}
func (s *Server) GetMatch(ctx context.Context, req *v1.GetMatchRequest) (*v1.GetMatchResponse, error) {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "match id is required")
	}
	m, err := scanMatch(s.db.QueryRowContext(ctx, `SELECT id::text,tournament_id::text,home_team_id::text,away_team_id::text,venue,kickoff,status,COALESCE(official_id::text,'') FROM matches WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "match not found")
	}
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "match id must be a UUID")
		}
		return nil, status.Error(codes.Internal, "could not read match")
	}
	return &v1.GetMatchResponse{Match: m}, nil
}
func (s *Server) ListMatches(ctx context.Context, req *v1.ListMatchesRequest) (*v1.ListMatchesResponse, error) {
	size, offset, err := page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	var after any
	if req.GetAfter() != nil {
		if err := req.GetAfter().CheckValid(); err != nil {
			return nil, status.Error(codes.InvalidArgument, "after timestamp is invalid")
		}
		after = req.GetAfter().AsTime()
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id::text,tournament_id::text,home_team_id::text,away_team_id::text,venue,kickoff,status,COALESCE(official_id::text,'') FROM matches WHERE ($1='' OR tournament_id::text=$1) AND ($2::timestamptz IS NULL OR kickoff >= $2) ORDER BY kickoff,id LIMIT $3 OFFSET $4`, strings.TrimSpace(req.GetTournamentId()), after, size+1, offset)
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "tournament id must be a UUID")
		}
		return nil, status.Error(codes.Internal, "could not list matches")
	}
	defer rows.Close()
	result := &v1.ListMatchesResponse{Matches: make([]*v1.Match, 0, size)}
	for rows.Next() {
		m, e := scanMatch(rows)
		if e != nil {
			return nil, status.Error(codes.Internal, "could not read matches")
		}
		if int32(len(result.Matches)) == size {
			result.NextPageToken = strconv.Itoa(offset + int(size))
			break
		}
		result.Matches = append(result.Matches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "could not read matches")
	}
	return result, nil
}
func (s *Server) AssignOfficial(ctx context.Context, req *v1.AssignOfficialRequest) (*v1.AssignOfficialResponse, error) {
	matchID, officialID := strings.TrimSpace(req.GetMatchId()), strings.TrimSpace(req.GetOfficialId())
	if matchID == "" || officialID == "" {
		return nil, status.Error(codes.InvalidArgument, "match and official ids are required")
	}
	m, err := scanMatch(s.db.QueryRowContext(ctx, `UPDATE matches SET official_id=$2 WHERE id=$1 RETURNING id::text,tournament_id::text,home_team_id::text,away_team_id::text,venue,kickoff,status,COALESCE(official_id::text,'')`, matchID, officialID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "match not found")
	}
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "ids must be UUIDs")
		}
		return nil, status.Error(codes.Internal, "could not assign official")
	}
	return &v1.AssignOfficialResponse{Match: m}, nil
}

type scanner interface{ Scan(...any) error }

func scanMatch(row scanner) (*v1.Match, error) {
	m := new(v1.Match)
	var kickoff sql.NullTime
	err := row.Scan(&m.Id, &m.TournamentId, &m.HomeTeamId, &m.AwayTeamId, &m.Venue, &kickoff, &m.Status, &m.OfficialId)
	if err != nil {
		return nil, err
	}
	if kickoff.Valid {
		m.Kickoff = timestamppb.New(kickoff.Time)
	}
	return m, nil
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
