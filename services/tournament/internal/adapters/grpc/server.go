package grpcadapter

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedTournamentServiceServer
	db *sql.DB
}

func New(db *sql.DB) *Server { return &Server{db: db} }
func (s *Server) CreateTournament(ctx context.Context, req *v1.CreateTournamentRequest) (*v1.CreateTournamentResponse, error) {
	name, sport, season := strings.TrimSpace(req.GetName()), strings.TrimSpace(req.GetSport()), strings.TrimSpace(req.GetSeason())
	if name == "" || sport == "" || season == "" || strings.TrimSpace(req.GetDzongkhag()) == "" {
		return nil, status.Error(codes.InvalidArgument, "name, sport, season, and dzongkhag are required")
	}
	var starts any
	if req.GetStartsAt() != nil {
		if err := req.GetStartsAt().CheckValid(); err != nil {
			return nil, status.Error(codes.InvalidArgument, "starts_at is invalid")
		}
		starts = req.GetStartsAt().AsTime()
	}
	t, err := scanTournament(s.db.QueryRowContext(ctx, `INSERT INTO tournaments(name,sport,season,venue,starts_at,dzongkhag) VALUES($1,$2,$3,$4,$5,$6) RETURNING id::text,name,sport,season,status,venue,starts_at,dzongkhag`, name, sport, season, strings.TrimSpace(req.GetVenue()), starts, strings.TrimSpace(req.GetDzongkhag())))
	if err != nil {
		return nil, status.Error(codes.Internal, "could not create tournament")
	}
	return &v1.CreateTournamentResponse{Tournament: t}, nil
}
func (s *Server) GetTournament(ctx context.Context, req *v1.GetTournamentRequest) (*v1.GetTournamentResponse, error) {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "tournament id is required")
	}
	t, err := scanTournament(s.db.QueryRowContext(ctx, `SELECT id::text,name,sport,season,status,venue,starts_at,dzongkhag FROM tournaments WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "tournament not found")
	}
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "tournament id must be a UUID")
		}
		return nil, status.Error(codes.Internal, "could not read tournament")
	}
	return &v1.GetTournamentResponse{Tournament: t}, nil
}
func (s *Server) ListTournaments(ctx context.Context, req *v1.ListTournamentsRequest) (*v1.ListTournamentsResponse, error) {
	size, offset, err := page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id::text,name,sport,season,status,venue,starts_at,dzongkhag FROM tournaments WHERE ($1='' OR season=$1) ORDER BY season DESC,name,id LIMIT $2 OFFSET $3`, strings.TrimSpace(req.GetSeason()), size+1, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not list tournaments")
	}
	defer rows.Close()
	result := &v1.ListTournamentsResponse{Tournaments: make([]*v1.Tournament, 0, size)}
	for rows.Next() {
		t, e := scanTournament(rows)
		if e != nil {
			return nil, status.Error(codes.Internal, "could not read tournaments")
		}
		if int32(len(result.Tournaments)) == size {
			result.NextPageToken = strconv.Itoa(offset + int(size))
			break
		}
		result.Tournaments = append(result.Tournaments, t)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "could not read tournaments")
	}
	return result, nil
}

type scanner interface{ Scan(...any) error }

func scanTournament(row scanner) (*v1.Tournament, error) {
	t := new(v1.Tournament)
	var starts sql.NullTime
	err := row.Scan(&t.Id, &t.Name, &t.Sport, &t.Season, &t.Status, &t.Venue, &starts, &t.Dzongkhag)
	if err != nil {
		return nil, err
	}
	if starts.Valid {
		t.StartsAt = timestamppb.New(starts.Time)
	}
	return t, nil
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
