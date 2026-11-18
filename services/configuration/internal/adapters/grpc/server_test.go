package grpcadapter

import (
	"context"
	"strings"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/configuration/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/packages/servicekit/testdb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestConfigurationGetAndSetRequireValidKeysAndAdmin(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	server := New(db)
	ctx := context.Background()

	if _, err := server.GetConfiguration(ctx, nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("nil get code = %s", status.Code(err))
	}
	for _, key := range []string{"", " white space ", "bad/key", strings.Repeat("a", 256)} {
		if _, err := server.GetConfiguration(ctx, &v1.GetConfigurationRequest{Key: key}); status.Code(err) != codes.InvalidArgument {
			t.Errorf("get key %q code = %s", key, status.Code(err))
		}
	}
	if _, err := server.GetConfiguration(ctx, &v1.GetConfigurationRequest{Key: "missing.key"}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing get code = %s", status.Code(err))
	}
	if _, err := server.SetConfiguration(ctx, &v1.SetConfigurationRequest{Key: "feature.enabled", Value: "true"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("anonymous set code = %s", status.Code(err))
	}
	viewer := servicekit.WithIdentity(ctx, servicekit.Identity{ID: "viewer-1", Role: core.RoleViewer})
	if _, err := server.SetConfiguration(viewer, &v1.SetConfigurationRequest{Key: "feature.enabled", Value: "true"}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("viewer set code = %s", status.Code(err))
	}

	admin := servicekit.WithIdentity(ctx, servicekit.Identity{ID: "admin-1", Role: core.RoleSuperAdmin})
	if _, err := server.SetConfiguration(admin, nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("nil set code = %s", status.Code(err))
	}
	if _, err := server.SetConfiguration(admin, &v1.SetConfigurationRequest{Key: "bad/key", Value: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid key set code = %s", status.Code(err))
	}
	if _, err := server.SetConfiguration(admin, &v1.SetConfigurationRequest{Key: "feature.value", Value: strings.Repeat("x", 65537)}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("oversize value code = %s", status.Code(err))
	}

	first, err := server.SetConfiguration(admin, &v1.SetConfigurationRequest{Key: " feature.enabled ", Value: "false"})
	if err != nil || first.GetEntry().GetKey() != "feature.enabled" || first.GetEntry().GetValue() != "false" {
		t.Fatalf("first set = %+v, err=%v", first, err)
	}
	updated, err := server.SetConfiguration(admin, &v1.SetConfigurationRequest{Key: "feature.enabled", Value: "true"})
	if err != nil || updated.GetEntry().GetValue() != "true" {
		t.Fatalf("updated set = %+v, err=%v", updated, err)
	}
	got, err := server.GetConfiguration(ctx, &v1.GetConfigurationRequest{Key: "feature.enabled"})
	if err != nil || got.GetEntry().GetValue() != "true" {
		t.Fatalf("get after update = %+v, err=%v", got, err)
	}
}
