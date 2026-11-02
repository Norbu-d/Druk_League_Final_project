package grpcadapter

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/registration/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedRegistrationServiceServer
	db *sql.DB
}

func New(db *sql.DB) *Server { return &Server{db: db} }

func (s *Server) RegisterTeam(ctx context.Context, req *v1.RegisterTeamRequest) (*v1.RegisterTeamResponse, error) {
	tournamentID, teamID := strings.TrimSpace(req.GetTournamentId()), strings.TrimSpace(req.GetTeamId())
	if tournamentID == "" || teamID == "" {
		return nil, status.Error(codes.InvalidArgument, "tournament id and team id are required")
	}
	reg, err := scanRegistration(s.db.QueryRowContext(ctx, `INSERT INTO registrations(tournament_id,team_id) VALUES($1,$2) RETURNING id::text,tournament_id::text,team_id::text,status,submitted_at`, tournamentID, teamID))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, status.Error(codes.AlreadyExists, "team already registered for this tournament")
		}
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "tournament and team ids must be UUIDs")
		}
		return nil, status.Error(codes.Internal, "could not register team")
	}
	return &v1.RegisterTeamResponse{Registration: reg}, nil
}

func (s *Server) ReviewRegistration(ctx context.Context, req *v1.ReviewRegistrationRequest) (*v1.ReviewRegistrationResponse, error) {
	id := strings.TrimSpace(req.GetRegistrationId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "registration id is required")
	}
	decision := "rejected"
	if req.GetApproved() {
		decision = "approved"
	}
	reg, err := scanRegistration(s.db.QueryRowContext(ctx, `UPDATE registrations SET status=$2,review_reason=$3,reviewed_at=now() WHERE id=$1 AND status='pending' RETURNING id::text,tournament_id::text,team_id::text,status,submitted_at`, id, decision, strings.TrimSpace(req.GetReason())))
	if err == nil {
		return &v1.ReviewRegistrationResponse{Registration: reg}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "registration id must be a UUID")
		}
		return nil, status.Error(codes.Internal, "could not review registration")
	}
	reg, err = scanRegistration(s.db.QueryRowContext(ctx, `SELECT id::text,tournament_id::text,team_id::text,status,submitted_at FROM registrations WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "registration not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "could not read registration")
	}
	if reg.Status != decision {
		return nil, status.Error(codes.FailedPrecondition, "registration has already been reviewed")
	}
	return &v1.ReviewRegistrationResponse{Registration: reg}, nil
}

func (s *Server) ListRegistrations(ctx context.Context, req *v1.ListRegistrationsRequest) (*v1.ListRegistrationsResponse, error) {
	size, offset, err := page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	statusFilter := strings.TrimSpace(req.GetStatus())
	if statusFilter != "" && statusFilter != "pending" && statusFilter != "approved" && statusFilter != "rejected" {
		return nil, status.Error(codes.InvalidArgument, "status must be pending, approved, or rejected")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id::text,tournament_id::text,team_id::text,status,submitted_at FROM registrations WHERE ($1='' OR tournament_id::text=$1) AND ($2='' OR status=$2) ORDER BY submitted_at DESC,id LIMIT $3 OFFSET $4`, strings.TrimSpace(req.GetTournamentId()), statusFilter, size+1, offset)
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "tournament id must be a UUID")
		}
		return nil, status.Error(codes.Internal, "could not list registrations")
	}
	defer rows.Close()
	result := &v1.ListRegistrationsResponse{Registrations: make([]*v1.Registration, 0, size)}
	for rows.Next() {
		reg, e := scanRegistration(rows)
		if e != nil {
			return nil, status.Error(codes.Internal, "could not read registrations")
		}
		if len(result.Registrations) == int(size) {
			result.NextPageToken = strconv.Itoa(offset + int(size))
			break
		}
		result.Registrations = append(result.Registrations, reg)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "could not read registrations")
	}
	return result, nil
}

type scanner interface{ Scan(...any) error }

func scanRegistration(row scanner) (*v1.Registration, error) {
	reg := new(v1.Registration)
	var submitted time.Time
	err := row.Scan(&reg.Id, &reg.TournamentId, &reg.TeamId, &reg.Status, &submitted)
	if err != nil {
		return nil, err
	}
	reg.SubmittedAt = timestamppb.New(submitted)
	return reg, nil
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
