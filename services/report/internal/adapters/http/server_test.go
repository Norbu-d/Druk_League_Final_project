package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/report/internal/app"
	"github.com/drukleague/platform/services/report/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	httpRequester  = "11111111-1111-4111-8111-111111111111"
	httpOtherUser  = "22222222-2222-4222-8222-222222222222"
	httpTournament = "33333333-3333-4333-8333-333333333333"
	httpReportID   = "44444444-4444-4444-8444-444444444444"
)

type httpRepo struct{ report domain.Report }

func (r *httpRepo) Create(_ context.Context, item domain.Report) (domain.Report, error) {
	item.ID, item.CreatedAt = httpReportID, time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	r.report = item
	return item, nil
}

func (r *httpRepo) Get(context.Context, string) (domain.Report, error) { return r.report, nil }

type httpStandings struct{}

func (httpStandings) Get(context.Context, string) ([]domain.Standing, error) {
	return []domain.Standing{{TeamName: "Thimphu", Points: 3}}, nil
}

type httpFixtures struct{}

func (httpFixtures) List(context.Context, string, int32, string) ([]domain.Fixture, string, error) {
	return nil, "", nil
}

func TestHTTPCreateGetAndDownloadReport(t *testing.T) {
	secret := []byte("test-secret")
	repo := &httpRepo{}
	service := app.New(repo, httpStandings{}, httpFixtures{}, "")
	handler := New(service, secret)
	token := signHTTPToken(t, secret, httpRequester, core.RoleSuperAdmin)
	request := httptest.NewRequest(http.MethodPost, "/api/reports", bytes.NewBufferString(`{"report_type":"standings","tournament_id":"`+httpTournament+`","format":"csv"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), "download_url") {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body)
	}
	var created reportResponse
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil || created.ID != httpReportID {
		t.Fatalf("create response=%+v err=%v", created, err)
	}
	getRequest := httptest.NewRequest(http.MethodGet, "/api/reports/"+httpReportID, nil)
	getRequest.Header.Set("Authorization", "Bearer "+token)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getResponse.Code, getResponse.Body)
	}
	downloadRequest := httptest.NewRequest(http.MethodGet, "/api/reports/files/"+httpReportID, nil)
	downloadRequest.Header.Set("Authorization", "Bearer "+token)
	downloadResponse := httptest.NewRecorder()
	handler.ServeHTTP(downloadResponse, downloadRequest)
	if downloadResponse.Code != http.StatusOK || !strings.Contains(downloadResponse.Header().Get("Content-Disposition"), "attachment") || !strings.Contains(downloadResponse.Body.String(), "team_name") {
		t.Fatalf("download status=%d headers=%v body=%s", downloadResponse.Code, downloadResponse.Header(), downloadResponse.Body)
	}
}

func TestHTTPAuthenticationAndOwnership(t *testing.T) {
	secret := []byte("test-secret")
	repo := &httpRepo{report: domain.Report{ID: httpReportID, RequesterID: httpRequester, ContentType: "text/csv", FileName: "export.csv", Format: "csv", Content: []byte("ok")}}
	handler := New(app.New(repo, httpStandings{}, httpFixtures{}, ""), secret)
	unauthenticated := httptest.NewRecorder()
	handler.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/api/reports/"+httpReportID, nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status=%d", unauthenticated.Code)
	}
	foreignToken := signHTTPToken(t, secret, httpOtherUser, core.RoleViewer)
	foreign := httptest.NewRequest(http.MethodGet, "/api/reports/files/"+httpReportID, nil)
	foreign.Header.Set("Authorization", "Bearer "+foreignToken)
	foreignResponse := httptest.NewRecorder()
	handler.ServeHTTP(foreignResponse, foreign)
	if foreignResponse.Code != http.StatusForbidden {
		t.Fatalf("foreign download status=%d body=%s", foreignResponse.Code, foreignResponse.Body)
	}
	invalid := httptest.NewRequest(http.MethodPost, "/api/reports", strings.NewReader(`{"report_type":"standings","tournament_id":"bad"}`))
	invalid.Header.Set("Authorization", "Bearer "+signHTTPToken(t, secret, httpRequester, core.RoleViewer))
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid create status=%d body=%s", invalidResponse.Code, invalidResponse.Body)
	}
	invalidToken := httptest.NewRequest(http.MethodGet, "/api/reports/"+httpReportID, nil)
	invalidToken.Header.Set("Authorization", "Bearer not-a-token")
	invalidTokenResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidTokenResponse, invalidToken)
	if invalidTokenResponse.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token status=%d", invalidTokenResponse.Code)
	}
	malformed := httptest.NewRequest(http.MethodPost, "/api/reports", strings.NewReader(`{"unexpected":true}`))
	malformed.Header.Set("Authorization", "Bearer "+signHTTPToken(t, secret, httpRequester, core.RoleViewer))
	malformedResponse := httptest.NewRecorder()
	handler.ServeHTTP(malformedResponse, malformed)
	if malformedResponse.Code != http.StatusBadRequest {
		t.Fatalf("malformed body status=%d", malformedResponse.Code)
	}
	trailing := httptest.NewRequest(http.MethodPost, "/api/reports", strings.NewReader(`{"report_type":"standings","tournament_id":"`+httpTournament+`"} {}`))
	trailing.Header.Set("Authorization", "Bearer "+signHTTPToken(t, secret, httpRequester, core.RoleViewer))
	trailingResponse := httptest.NewRecorder()
	handler.ServeHTTP(trailingResponse, trailing)
	if trailingResponse.Code != http.StatusBadRequest {
		t.Fatalf("trailing JSON body status=%d", trailingResponse.Code)
	}
}

func TestHTTPErrorStatusMapping(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{core.Invalid("bad"), http.StatusBadRequest},
		{core.Forbidden("no"), http.StatusForbidden},
		{core.ErrUnauthenticated, http.StatusUnauthorized},
		{core.NotFound("missing"), http.StatusNotFound},
		{core.Dependency("down"), http.StatusServiceUnavailable},
		{core.Precondition("invalid state"), http.StatusFailedDependency},
		{status.Error(codes.InvalidArgument, "bad"), http.StatusBadRequest},
		{status.Error(codes.PermissionDenied, "no"), http.StatusForbidden},
		{status.Error(codes.NotFound, "missing"), http.StatusNotFound},
		{status.Error(codes.Unavailable, "down"), http.StatusServiceUnavailable},
		{errors.New("internal"), http.StatusInternalServerError},
	}
	for _, item := range cases {
		response := httptest.NewRecorder()
		writeServiceError(response, item.err)
		if response.Code != item.want {
			t.Errorf("error %v produced HTTP %d, want %d", item.err, response.Code, item.want)
		}
	}
}

func signHTTPToken(t *testing.T, secret []byte, id, role string) string {
	t.Helper()
	token, _, err := servicekit.SignToken(secret, servicekit.Identity{ID: id, Role: role}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return token
}
