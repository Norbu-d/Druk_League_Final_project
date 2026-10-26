package core

import (
	"errors"
	"testing"
)

func TestRolesAndUUIDs(t *testing.T) {
	for _, role := range []string{RoleSuperAdmin, RoleMatchOfficial, RoleTeamManager, RoleViewer} {
		if !IsUserRole(role) {
			t.Fatalf("%s should be a user role", role)
		}
	}
	if IsUserRole(RoleService) || IsUserRole("") {
		t.Fatal("non-user role accepted")
	}
	id, err := NewUUID()
	if err != nil || !IsUUID(id) || id[14] != '4' {
		t.Fatalf("bad v4 uuid %q", id)
	}
	if IsUUID("123") || IsUUID("zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz") {
		t.Fatal("invalid uuid accepted")
	}
}

func TestErrorHelpers(t *testing.T) {
	for err, sentinel := range map[error]error{Invalid("x"): ErrInvalid, NotFound("x"): ErrNotFound, Conflict("x"): ErrConflict, Forbidden("x"): ErrForbidden, Dependency("x"): ErrDependency, Precondition("x"): ErrPrecondition} {
		if !errors.Is(err, sentinel) {
			t.Fatalf("%v does not wrap %v", err, sentinel)
		}
	}
}
