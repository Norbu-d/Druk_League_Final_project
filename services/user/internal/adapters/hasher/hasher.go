// Package hasher stores passwords with bcrypt and still verifies the PBKDF2-SHA256 hashes
// written by earlier versions, so existing accounts keep working and are upgraded on login.
package hasher

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

type Bcrypt struct {
	cost  int
	dummy []byte
}

// New returns a bcrypt hasher; cost below bcrypt.MinCost uses the default of 12.
func New(cost int) *Bcrypt {
	if cost < bcrypt.MinCost {
		cost = 12
	}
	dummy, _ := bcrypt.GenerateFromPassword([]byte("timing-equaliser-not-a-password"), cost)
	return &Bcrypt{cost: cost, dummy: dummy}
}

func (b *Bcrypt) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	return string(hash), err
}

func (b *Bcrypt) Verify(hash, password string) (bool, bool) {
	switch {
	case hash == "":
		_ = bcrypt.CompareHashAndPassword(b.dummy, []byte(password))
		return false, false
	case strings.HasPrefix(hash, "pbkdf2$"):
		return verifyPBKDF2(hash, password), true
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return false, false
	}
	cost, err := bcrypt.Cost([]byte(hash))
	return true, err == nil && cost < b.cost
}

// verifyPBKDF2 checks the legacy format pbkdf2$<iterations>$<salt>$<key>.
func verifyPBKDF2(stored, password string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 4 {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations < 1 {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[2])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[3])
	if err1 != nil || err2 != nil || len(want) == 0 {
		return false
	}
	got := pbkdf2.Key([]byte(password), salt, iterations, len(want), sha256.New)
	return hmac.Equal(got, want)
}
