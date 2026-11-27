package grpcadapter

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/configuration/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var validKey = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,254}$`)

type Server struct {
	v1.UnimplementedConfigurationServiceServer
	db *sql.DB
}

func New(db *sql.DB) *Server { return &Server{db: db} }

func (s *Server) GetConfiguration(ctx context.Context, request *v1.GetConfigurationRequest) (*v1.GetConfigurationResponse, error) {
	if request == nil || !validKey.MatchString(strings.TrimSpace(request.GetKey())) {
		return nil, status.Error(codes.InvalidArgument, "configuration key is invalid")
	}
	var entry v1.ConfigurationEntry
	err := s.db.QueryRowContext(ctx, `SELECT key, value FROM configuration_entries WHERE key=$1`, strings.TrimSpace(request.GetKey())).Scan(&entry.Key, &entry.Value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "configuration entry not found")
	}
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.GetConfigurationResponse{Entry: &entry}, nil
}

func (s *Server) SetConfiguration(ctx context.Context, request *v1.SetConfigurationRequest) (*v1.SetConfigurationResponse, error) {
	if err := servicekit.RequireRole(ctx, servicekit.RoleSuperAdmin); err != nil {
		return nil, err
	}
	key := ""
	value := ""
	if request != nil {
		key = strings.TrimSpace(request.GetKey())
		value = request.GetValue()
	}
	if !validKey.MatchString(key) || len(value) > 65536 {
		return nil, status.Error(codes.InvalidArgument, "configuration key or value is invalid")
	}
	var entry v1.ConfigurationEntry
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO configuration_entries(key,value,value_type)
		VALUES($1,$2,'string')
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, value_type='string', updated_at=now()
		RETURNING key,value`, key, value).Scan(&entry.Key, &entry.Value)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.SetConfigurationResponse{Entry: &entry}, nil
}
