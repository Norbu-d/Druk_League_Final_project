package httpadapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/packages/servicekit/testdb"
)

const (
	mediaTestOwner = "11111111-1111-4111-8111-111111111111"
	mediaTestMatch = "22222222-2222-4222-8222-222222222222"
	mediaTestOther = "33333333-3333-4333-8333-333333333333"
)

func TestMediaUploadListRetrieveAndDelete(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	secret := []byte("media-integration-test-secret")
	root := t.TempDir()
	handler := New(db, secret, root).Handler()
	ownerToken := mediaTestToken(t, secret, mediaTestOwner, servicekit.RoleTeamManager)
	otherToken := mediaTestToken(t, secret, mediaTestOther, servicekit.RoleTeamManager)

	assetID := uploadTestPhoto(t, handler, ownerToken)

	list := httptest.NewRequest(http.MethodGet, "/api/matches/"+mediaTestMatch+"/media", nil)
	list.Header.Set("Authorization", "Bearer "+ownerToken)
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, list)
	var items []asset
	if listResponse.Code != http.StatusOK || json.Unmarshal(listResponse.Body.Bytes(), &items) != nil || len(items) != 1 || items[0].ID != assetID {
		t.Fatalf("list status=%d body=%s", listResponse.Code, listResponse.Body)
	}

	get := httptest.NewRequest(http.MethodGet, "/api/media/"+assetID, nil)
	get.Header.Set("Authorization", "Bearer "+otherToken)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK || getResponse.Header().Get("Content-Type") != "image/png" || !bytes.Equal(getResponse.Body.Bytes(), tinyPNG) {
		t.Fatalf("get status=%d headers=%v size=%d", getResponse.Code, getResponse.Header(), getResponse.Body.Len())
	}

	foreignDelete := httptest.NewRequest(http.MethodDelete, "/api/media/"+assetID, nil)
	foreignDelete.Header.Set("Authorization", "Bearer "+otherToken)
	foreignDeleteResponse := httptest.NewRecorder()
	handler.ServeHTTP(foreignDeleteResponse, foreignDelete)
	if foreignDeleteResponse.Code != http.StatusForbidden {
		t.Fatalf("foreign delete status=%d body=%s", foreignDeleteResponse.Code, foreignDeleteResponse.Body)
	}

	delete := httptest.NewRequest(http.MethodDelete, "/api/media/"+assetID, nil)
	delete.Header.Set("Authorization", "Bearer "+ownerToken)
	deleteResponse := httptest.NewRecorder()
	handler.ServeHTTP(deleteResponse, delete)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleteResponse.Code, deleteResponse.Body)
	}
}

func TestMediaRejectsUnauthenticatedAndUnsupportedUploads(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	secret := []byte("media-integration-test-secret")
	handler := New(db, secret, t.TempDir()).Handler()
	unauthenticated := httptest.NewRecorder()
	handler.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodPost, "/api/media", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated upload status=%d", unauthenticated.Code)
	}

	token := mediaTestToken(t, secret, mediaTestOwner, servicekit.RoleTeamManager)
	request := mediaUploadRequest(t, token, []byte("not an image"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("unsupported upload status=%d body=%s", response.Code, response.Body)
	}
}

func TestMediaValidationPathsWithoutDatabase(t *testing.T) {
	secret := []byte("media-validation-test-secret")
	handler := New(nil, secret, t.TempDir()).Handler()
	token := mediaTestToken(t, secret, mediaTestOwner, servicekit.RoleTeamManager)

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/media/not-a-uuid", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated get status=%d", unauthorized.Code)
	}

	invalidToken := httptest.NewRequest(http.MethodGet, "/api/media/not-a-uuid", nil)
	invalidToken.Header.Set("Authorization", "Bearer invalid")
	invalidToken.Header.Set("Cookie", "drukleague-session="+token)
	invalidTokenResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidTokenResponse, invalidToken)
	if invalidTokenResponse.Code != http.StatusBadRequest {
		t.Fatalf("cookie fallback status=%d body=%s", invalidTokenResponse.Code, invalidTokenResponse.Body)
	}

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(method, "/api/media/not-a-uuid", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s invalid media id status=%d", method, response.Code)
		}
	}
	list := httptest.NewRequest(http.MethodGet, "/api/matches/not-a-uuid/media", nil)
	list.Header.Set("Authorization", "Bearer "+token)
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, list)
	if listResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid match id status=%d", listResponse.Code)
	}

	badMultipart := httptest.NewRequest(http.MethodPost, "/api/media", strings.NewReader("not multipart"))
	badMultipart.Header.Set("Authorization", "Bearer "+token)
	badMultipart.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
	badMultipartResponse := httptest.NewRecorder()
	handler.ServeHTTP(badMultipartResponse, badMultipart)
	if badMultipartResponse.Code != http.StatusBadRequest {
		t.Fatalf("malformed multipart status=%d", badMultipartResponse.Code)
	}

	cases := []struct {
		name, matchID, kind, caption string
		payload                      []byte
		includeFile                  bool
		want                         int
	}{
		{"bad match id", "bad", "photo", "", tinyPNG, true, http.StatusBadRequest},
		{"bad media kind", mediaTestMatch, "audio", "", tinyPNG, true, http.StatusBadRequest},
		{"long caption", mediaTestMatch, "photo", strings.Repeat("x", 1001), tinyPNG, true, http.StatusBadRequest},
		{"missing file", mediaTestMatch, "photo", "", nil, false, http.StatusBadRequest},
		{"empty file", mediaTestMatch, "photo", "", nil, true, http.StatusBadRequest},
		{"invalid image bytes", mediaTestMatch, "photo", "", []byte("not an image"), true, http.StatusUnsupportedMediaType},
		{"photo required for image", mediaTestMatch, "highlight", "", tinyPNG, true, http.StatusUnsupportedMediaType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := mediaUploadRequestWithFields(t, token, tc.matchID, tc.kind, tc.caption, tc.payload, tc.includeFile)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.want {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tc.want, response.Body)
			}
		})
	}

	fileRoot := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(fileRoot, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	storageFailure := New(nil, secret, fileRoot).Handler()
	request := mediaUploadRequest(t, token, tinyPNG)
	response := httptest.NewRecorder()
	storageFailure.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("storage setup status=%d body=%s", response.Code, response.Body)
	}
}

func TestMediaStorageHelpers(t *testing.T) {
	if !validStorageKey(strings.Repeat("ab", 16)) || validStorageKey("../outside") || validStorageKey("short") || validStorageKey(strings.Repeat("z", 32)) {
		t.Fatal("storage key validation did not enforce a 16-byte hex key")
	}
	for _, tc := range []struct {
		contentType, kind string
		want              bool
	}{
		{"video/mp4", "highlight", true}, {"image/png", "photo", true}, {"image/jpeg", "photo", true}, {"image/webp", "photo", true},
		{"video/mp4", "photo", false}, {"image/png", "highlight", false}, {"text/plain", "photo", false},
	} {
		t.Run(fmt.Sprintf("%s_%s", tc.contentType, tc.kind), func(t *testing.T) {
			if got := allowedContentType(tc.contentType, tc.kind); got != tc.want {
				t.Fatalf("allowed=%v want=%v", got, tc.want)
			}
		})
	}
	first, err := randomKey()
	if err != nil {
		t.Fatal(err)
	}
	second, err := randomKey()
	if err != nil {
		t.Fatal(err)
	}
	if !validStorageKey(first) || first == second {
		t.Fatalf("random keys invalid or reused: %q %q", first, second)
	}
}

var tinyPNG = []byte{137, 80, 78, 71, 13, 10, 26, 10}

func uploadTestPhoto(t *testing.T, handler http.Handler, token string) string {
	t.Helper()
	request := mediaUploadRequest(t, token, tinyPNG)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload status=%d body=%s", response.Code, response.Body)
	}
	var created asset
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("upload response=%s err=%v", response.Body, err)
	}
	return created.ID
}

func mediaUploadRequest(t *testing.T, token string, content []byte) *http.Request {
	return mediaUploadRequestWithFields(t, token, mediaTestMatch, "photo", "Match day", content, true)
}

func mediaUploadRequestWithFields(t *testing.T, token, matchID, kind, caption string, content []byte, includeFile bool) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("match_id", matchID)
	_ = writer.WriteField("media_kind", kind)
	_ = writer.WriteField("caption", caption)
	if includeFile {
		file, err := writer.CreateFormFile("file", "asset.bin")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/media", &body)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func mediaTestToken(t *testing.T, secret []byte, userID, role string) string {
	t.Helper()
	token, _, err := servicekit.SignToken(secret, servicekit.Identity{ID: userID, Role: role}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return token
}
