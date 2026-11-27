package servicekit

import (
	"context"
	"net"
	"testing"

	configurationv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/configuration/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type configurationLookup struct {
	configurationv1.UnimplementedConfigurationServiceServer
	entries map[string]string
	err     error
}

func (s configurationLookup) GetConfiguration(_ context.Context, req *configurationv1.GetConfigurationRequest) (*configurationv1.GetConfigurationResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	value, ok := s.entries[req.GetKey()]
	if !ok {
		return nil, status.Error(codes.NotFound, "missing")
	}
	return &configurationv1.GetConfigurationResponse{Entry: &configurationv1.ConfigurationEntry{Key: req.GetKey(), Value: value}}, nil
}

func serveConfigurationLookup(t *testing.T, lookup configurationLookup) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	configurationv1.RegisterConfigurationServiceServer(server, lookup)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	return listener.Addr().String()
}

func TestLoadConfigurationAppliesOnlyAllowlistedValues(t *testing.T) {
	const service = "fixture-service"
	const key = "service.fixture-service.GRPC_ADDR"
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("GRPC_ADDR", "environment:50051")
	t.Setenv("DATABASE_URL", "environment-database")
	t.Setenv("CONFIGURATION_SERVICE_ADDR", serveConfigurationLookup(t, configurationLookup{entries: map[string]string{
		key: "central:50051",
	}}))

	LoadConfiguration(service, "GRPC_ADDR", "DATABASE_URL", "not-a-setting")
	if got := Env("GRPC_ADDR", ""); got != "central:50051" {
		t.Fatalf("GRPC_ADDR = %q, want centralized value", got)
	}
	if got := Env("DATABASE_URL", ""); got != "environment-database" {
		t.Fatalf("DATABASE_URL was overwritten: %q", got)
	}
}

func TestLoadConfigurationKeepsEnvironmentForMissingOrUnavailableValues(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("GRPC_ADDR", "environment:50051")
	t.Setenv("CONFIGURATION_SERVICE_ADDR", serveConfigurationLookup(t, configurationLookup{entries: map[string]string{}}))
	LoadConfiguration("fixture-service", "GRPC_ADDR")
	if got := Env("GRPC_ADDR", ""); got != "environment:50051" {
		t.Fatalf("missing value changed GRPC_ADDR to %q", got)
	}

	t.Setenv("CONFIGURATION_SERVICE_ADDR", serveConfigurationLookup(t, configurationLookup{err: status.Error(codes.PermissionDenied, "denied")}))
	LoadConfiguration("fixture-service", "GRPC_ADDR")
	if got := Env("GRPC_ADDR", ""); got != "environment:50051" {
		t.Fatalf("failed lookup changed GRPC_ADDR to %q", got)
	}
}

func TestLoadConfigurationSkipsWhenNotConfiguredOrSelf(t *testing.T) {
	t.Setenv("CONFIGURATION_SERVICE_ADDR", "127.0.0.1:1")
	t.Setenv("GRPC_ADDR", "environment:50051")
	for _, test := range []struct {
		service  string
		settings []string
	}{
		{service: "fixture-service", settings: []string{"GRPC_ADDR"}},
		{service: "", settings: []string{"GRPC_ADDR"}},
		{service: "configuration-service", settings: []string{"GRPC_ADDR"}},
		{service: "fixture-service"},
	} {
		if test.service == "fixture-service" && len(test.settings) == 1 {
			t.Setenv("CONFIGURATION_SERVICE_ADDR", "")
		}
		LoadConfiguration(test.service, test.settings...)
	}
	if got := Env("GRPC_ADDR", ""); got != "environment:50051" {
		t.Fatalf("early return changed GRPC_ADDR to %q", got)
	}
}
