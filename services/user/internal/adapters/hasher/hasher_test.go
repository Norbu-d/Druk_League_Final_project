package hasher

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

func TestBcryptRoundTrip(t *testing.T) {
	h := New(bcrypt.MinCost)
	hash, err := h.Hash("correct horse battery")
	if err != nil || !strings.HasPrefix(hash, "$2a$") {
		t.Fatalf("hash %q %v", hash, err)
	}
	if ok, rehash := h.Verify(hash, "correct horse battery"); !ok || rehash {
		t.Fatalf("verify ok=%v rehash=%v", ok, rehash)
	}
	if ok, _ := h.Verify(hash, "wrong password!"); ok {
		t.Fatal("wrong password accepted")
	}
	if ok, _ := h.Verify("", "anything"); ok {
		t.Fatal("empty hash accepted")
	}
}

func TestLowerCostHashIsUpgraded(t *testing.T) {
	weak, _ := New(bcrypt.MinCost).Hash("correct horse battery")
	if ok, rehash := New(bcrypt.MinCost+1).Verify(weak, "correct horse battery"); !ok || !rehash {
		t.Fatalf("expected upgrade: ok=%v rehash=%v", ok, rehash)
	}
	if New(0).cost != 12 {
		t.Fatal("default cost not applied")
	}
}

// The monolith and the previous User service stored pbkdf2$120000$<salt>$<key>.
func TestLegacyPBKDF2HashesStillVerifyAndAreUpgraded(t *testing.T) {
	salt := []byte("0123456789abcdef")
	key := pbkdf2.Key([]byte("drukleague-dev"), salt, 1000, 32, sha256.New)
	legacy := fmt.Sprintf("pbkdf2$1000$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
	h := New(bcrypt.MinCost)
	if ok, rehash := h.Verify(legacy, "drukleague-dev"); !ok || !rehash {
		t.Fatalf("legacy hash: ok=%v rehash=%v", ok, rehash)
	}
	if ok, _ := h.Verify(legacy, "wrong"); ok {
		t.Fatal("wrong password accepted for legacy hash")
	}
	for _, broken := range []string{"pbkdf2$x$a$b", "pbkdf2$1000$!!$b", "pbkdf2$1000", "pbkdf2$0$YQ$YQ", "pbkdf2$1000$YQ$"} {
		if ok, _ := h.Verify(broken, "drukleague-dev"); ok {
			t.Fatalf("malformed hash %q accepted", broken)
		}
	}
}
