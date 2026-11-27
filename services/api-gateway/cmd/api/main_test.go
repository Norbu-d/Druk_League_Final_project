package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTokenIssueAndVerify(t *testing.T) {
	s := &server{secret: []byte("test-secret")}
	want := principal{ID: "user-1", Username: "admin", Role: "super_admin", Expires: time.Now().Add(time.Minute).Unix()}
	token, err := s.issue(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.verify(token)
	if err != nil || got != want {
		t.Fatalf("verify() = %#v, %v; want %#v", got, err, want)
	}
	if _, err := s.verify(token + "x"); err == nil {
		t.Fatal("verify() accepted a token with a changed signature")
	}
	if _, err := s.verify("bad-token"); err == nil {
		t.Fatal("verify() accepted a malformed token")
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	s := &server{secret: []byte("test-secret")}
	token, err := s.issue(principal{ID: "user-1", Expires: time.Now().Add(-time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.verify(token); err == nil {
		t.Fatal("verify() accepted an expired token")
	}
}

func TestDecodeRejectsInvalidAndUnknownFields(t *testing.T) {
	for _, body := range []string{`{`, `{"name":"ok","extra":true}`} {
		t.Run(body, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			w := httptest.NewRecorder()
			var target struct {
				Name string `json:"name"`
			}
			if decode(w, r, &target) {
				t.Fatal("decode() accepted an invalid body")
			}
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
		})
	}
}

func TestDecodeAcceptsKnownFields(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"team"}`))
	w := httptest.NewRecorder()
	var target struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &target) || target.Name != "team" {
		t.Fatalf("decode() = %v, %#v", w.Code, target)
	}
}

func TestManageAndNullableHelpers(t *testing.T) {
	if !canManage(principal{Role: "super_admin"}) || canManage(principal{Role: "team_manager"}) {
		t.Fatal("canManage() did not enforce the super_admin role")
	}
	if nullableDate(sql.NullTime{}) != nil || nullableInt(sql.NullInt64{}) != nil {
		t.Fatal("invalid nullable database values must map to nil")
	}
	now := time.Now()
	if got := nullableDate(sql.NullTime{Time: now, Valid: true}); got != now {
		t.Fatalf("nullableDate() = %#v", got)
	}
	if got := nullableInt(sql.NullInt64{Int64: 9, Valid: true}); got != int64(9) {
		t.Fatalf("nullableInt() = %#v", got)
	}
}

func TestResponseHelpers(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusForbidden, "denied")
	var payload apiError
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusForbidden || payload.Error != "denied" || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("unexpected error response: status=%d payload=%#v headers=%v", w.Code, payload, w.Header())
	}

	w = httptest.NewRecorder()
	methodNotAllowed(w)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("methodNotAllowed() status = %d", w.Code)
	}

	w = httptest.NewRecorder()
	serverError(w, errors.New("private database detail"))
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "private database detail") {
		t.Fatalf("serverError() leaked details or returned wrong status: %d %s", w.Code, w.Body.String())
	}
}

func TestCORSOptionsShortCircuitsHandler(t *testing.T) {
	called := false
	handler := cors(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodOptions, "/api/teams", nil))
	if called || w.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS: called=%v status=%d", called, w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Headers") != "Authorization, Content-Type" {
		t.Fatalf("unexpected CORS headers: %v", w.Header())
	}
}
