package servicekit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var testSecret = []byte("unit-test-secret-unit-test-secret")

func TestSignAndVerifyRoundTrip(t *testing.T) {
	token, expires, err := SignToken(testSecret, Identity{ID: "u1", Username: "karma", Role: RoleMatchOfficial, SessionID: "s1"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if time.Until(expires) <= 0 {
		t.Fatal("expiry must be in the future")
	}
	identity, err := VerifyToken(token, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	if identity != (Identity{ID: "u1", Username: "karma", Role: RoleMatchOfficial, SessionID: "s1"}) {
		t.Fatalf("unexpected identity %+v", identity)
	}
}

func TestSignTokenRejectsMissingInputs(t *testing.T) {
	cases := []struct {
		secret   []byte
		identity Identity
		ttl      time.Duration
	}{{nil, Identity{ID: "u"}, time.Minute}, {testSecret, Identity{}, time.Minute}, {testSecret, Identity{ID: "u"}, 0}}
	for _, c := range cases {
		if _, _, err := SignToken(c.secret, c.identity, c.ttl); err == nil {
			t.Fatalf("expected error for %+v", c)
		}
	}
}

func TestVerifyRejectsWrongSecretExpiredAndNone(t *testing.T) {
	token, _, _ := SignToken(testSecret, Identity{ID: "u1"}, time.Minute)
	if _, err := VerifyToken(token, []byte("another-secret")); err == nil {
		t.Fatal("wrong secret accepted")
	}
	if _, err := VerifyToken(token, nil); err == nil {
		t.Fatal("empty secret accepted")
	}
	expired := handRolled(t, map[string]any{"alg": "HS256", "typ": "JWT"}, map[string]any{"sub": "u1", "exp": time.Now().Add(-time.Minute).Unix()})
	if _, err := VerifyToken(expired, testSecret); err == nil {
		t.Fatal("expired token accepted")
	}
	none := handRolled(t, map[string]any{"alg": "none"}, map[string]any{"sub": "u1", "exp": time.Now().Add(time.Minute).Unix()})
	if _, err := VerifyToken(none, testSecret); err == nil {
		t.Fatal("alg=none accepted")
	}
	noSubject := handRolled(t, map[string]any{"alg": "HS256"}, map[string]any{"exp": time.Now().Add(time.Minute).Unix()})
	if _, err := VerifyToken(noSubject, testSecret); err == nil {
		t.Fatal("token without subject accepted")
	}
	noExpiry := handRolled(t, map[string]any{"alg": "HS256"}, map[string]any{"sub": "u1"})
	if _, err := VerifyToken(noExpiry, testSecret); err == nil {
		t.Fatal("token without expiry accepted")
	}
}

// The monolith signs tokens by hand; they must keep working against the library verifier.
func TestVerifyAcceptsMonolithTokens(t *testing.T) {
	token := handRolled(t, map[string]any{"alg": "HS256", "typ": "JWT"}, map[string]any{"sub": "admin-id", "username": "admin", "role": "super_admin", "exp": time.Now().Add(time.Hour).Unix()})
	identity, err := VerifyToken(token, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	if identity.ID != "admin-id" || identity.Role != RoleSuperAdmin || identity.Username != "admin" {
		t.Fatalf("unexpected identity %+v", identity)
	}
}

func handRolled(t *testing.T, header, claims map[string]any) string {
	t.Helper()
	h, _ := json.Marshal(header)
	c, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(c)
	mac := hmac.New(sha256.New, testSecret)
	mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestAuthInterceptor(t *testing.T) {
	interceptor := AuthUnaryInterceptor(testSecret)
	var seen Identity
	handler := func(ctx context.Context, _ any) (any, error) {
		seen, _ = IdentityFromContext(ctx)
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/drukleague.v1.TeamService/ListTeams"}

	if _, err := interceptor(context.Background(), nil, info, handler); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing metadata: got %v", err)
	}
	bad := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Basic abc"))
	if _, err := interceptor(bad, nil, info, handler); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("non-bearer: got %v", err)
	}
	invalid := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer nope"))
	if _, err := interceptor(invalid, nil, info, handler); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("invalid token: got %v", err)
	}
	token, _, _ := SignToken(testSecret, Identity{ID: "u1", Role: RoleViewer}, time.Minute)
	good := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	if _, err := interceptor(good, nil, info, handler); err != nil || seen.ID != "u1" {
		t.Fatalf("valid token: err=%v identity=%+v", err, seen)
	}
	for _, method := range []string{"/grpc.health.v1.Health/Check", "/drukleague.auth.v1.AuthService/Login", "/drukleague.auth.v1.AuthService/RefreshToken"} {
		if _, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: method}, handler); err != nil {
			t.Fatalf("public method %s rejected: %v", method, err)
		}
	}
}

func TestRequireRoleAndSelf(t *testing.T) {
	if status.Code(RequireRole(context.Background(), RoleSuperAdmin)) != codes.Unauthenticated {
		t.Fatal("anonymous caller must be unauthenticated")
	}
	if status.Code(RequireSelfOrAdmin(context.Background(), "u1")) != codes.Unauthenticated {
		t.Fatal("anonymous caller must be unauthenticated")
	}
	viewer := WithIdentity(context.Background(), Identity{ID: "u1", Role: RoleViewer})
	if status.Code(RequireRole(viewer, RoleSuperAdmin)) != codes.PermissionDenied {
		t.Fatal("viewer must be denied")
	}
	if RequireRole(viewer, RoleSuperAdmin, RoleViewer) != nil {
		t.Fatal("allowed role denied")
	}
	if RequireSelfOrAdmin(viewer, "u1") != nil || status.Code(RequireSelfOrAdmin(viewer, "u2")) != codes.PermissionDenied {
		t.Fatal("self check is wrong")
	}
	admin := WithIdentity(context.Background(), Identity{ID: "a", Role: RoleSuperAdmin})
	if RequireSelfOrAdmin(admin, "u2") != nil {
		t.Fatal("admin denied")
	}
}

func TestServiceTokensAndTokenSource(t *testing.T) {
	if _, err := NewServiceToken("", testSecret, time.Minute); err == nil {
		t.Fatal("empty service name accepted")
	}
	token, err := NewServiceToken("standings-service", testSecret, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := VerifyToken(token, testSecret)
	if err != nil || identity.Role != RoleService || identity.ID != "service:standings-service" {
		t.Fatalf("service token identity %+v err %v", identity, err)
	}

	source := NewTokenSource("standings-service", testSecret, 10*time.Minute)
	now := time.Now()
	source.now = func() time.Time { return now }
	first, err := source.Token()
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := source.Token(); again != first {
		t.Fatal("token should be cached while fresh")
	}
	// Nine minutes later less than a fifth of the lifetime is left, so a new token is signed.
	now = now.Add(9 * time.Minute)
	source.expires = now.Add(time.Minute)
	source.token = "stale"
	if renewed, _ := source.Token(); renewed == "stale" {
		t.Fatal("token should be renewed near expiry")
	}
	if NewTokenSource("x", testSecret, 0).ttl != 10*time.Minute {
		t.Fatal("default ttl not applied")
	}
	if _, err := NewTokenSource("", testSecret, time.Minute).Token(); err == nil {
		t.Fatal("invalid token source should fail")
	}
}

func TestIsUserRole(t *testing.T) {
	for _, role := range []string{RoleSuperAdmin, RoleMatchOfficial, RoleTeamManager, RoleViewer} {
		if !IsUserRole(role) {
			t.Fatalf("%s should be a user role", role)
		}
	}
	if IsUserRole(RoleService) || IsUserRole("dzongkhag_admin") {
		t.Fatal("non-user role accepted")
	}
}
