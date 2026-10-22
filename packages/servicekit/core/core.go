// Package core holds dependency-free helpers that domain and app layers may import:
// role names, UUID validation and generation. It must not import gRPC, SQL or NATS.
package core

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
)

// Role names shared by every service, the gateway and the BFF.
const (
	RoleSuperAdmin    = "super_admin"
	RoleMatchOfficial = "match_official"
	RoleTeamManager   = "team_manager"
	RoleViewer        = "viewer"
	// RoleService identifies an internal service-to-service token.
	RoleService = "service"
)

// IsUserRole reports whether role can be held by a human account.
func IsUserRole(role string) bool {
	switch role {
	case RoleSuperAdmin, RoleMatchOfficial, RoleTeamManager, RoleViewer:
		return true
	}
	return false
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s is a canonical UUID.
func IsUUID(s string) bool { return uuidPattern.MatchString(strings.TrimSpace(s)) }

// NewUUID returns a random version-4 UUID.
func NewUUID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw[:])
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:], nil
}
