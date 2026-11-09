package httpadapter

import (
	"bytes"
	"context"
	"database/sql"
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

func TestMediaNotFoundAndInvalidStorageMetadata(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	secret := []byte("media-metadata-test-secret")
	token := mediaTestToken(t, secret, mediaTestOwner, servicekit.RoleTeamManager)
	handler := New(db, secret, t.TempDir()).Handler()
	missing := httptest.NewRecorder()
	missingGet := httptest.NewRequest(http.MethodGet, "/api/media/66666666-6666-4666-8666-666666666666", nil)
	missingGet.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(missing, missingGet)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing get status=%d body=%s", missing.Code, missing.Body)
	}
	missingDelete := httptest.NewRecorder()
	missingDeleteRequest := httptest.NewRequest(http.MethodDelete, "/api/media/66666666-6666-4666-8666-666666666666", nil)
	missingDeleteRequest.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(missingDelete, missingDeleteRequest)
	if missingDelete.Code != http.StatusNotFound {
		t.Fatalf("missing delete status=%d body=%s", missingDelete.Code, missingDelete.Body)
	}

	unsafeDownloadID := "77777777-7777-4777-8777-777777777771"
	unsafeDeleteID := "77777777-7777-4777-8777-777777777772"
	seedMediaAsset(t, db, unsafeDownloadID, mediaTestOwner, "../outside")
	seedMediaAsset(t, db, unsafeDeleteID, mediaTestOwner, "../outside-too")
	unsafeGet := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/media/"+unsafeDownloadID, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(unsafeGet, request)
	if unsafeGet.Code != http.StatusInternalServerError {
		t.Fatalf("unsafe key get status=%d", unsafeGet.Code)
	}
	unsafeDelete := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodDelete, "/api/media/"+unsafeDeleteID, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(unsafeDelete, request)
	if unsafeDelete.Code != http.StatusInternalServerError {
		t.Fatalf("unsafe key delete status=%d", unsafeDelete.Code)
	}

	missingFileID := "88888888-8888-4888-8888-888888888881"
	seedMediaAsset(t, db, missingFileID, mediaTestOwner, strings.Repeat("a", 32))
	missingFileGet := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/media/"+missingFileID, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(missingFileGet, request)
	if missingFileGet.Code != http.StatusNotFound {
		t.Fatalf("missing file get status=%d", missingFileGet.Code)
	}
	missingFileDelete := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodDelete, "/api/media/"+missingFileID, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(missingFileDelete, request)
	if missingFileDelete.Code != http.StatusNoContent {
		t.Fatalf("missing file delete status=%d body=%s", missingFileDelete.Code, missingFileDelete.Body)
	}
}

func TestMediaAdminCanDeleteAndEmptyMatchListsAreArrays(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	secret := []byte("media-admin-test-secret")
	root := t.TempDir()
	handler := New(db, secret, root).Handler()
	list := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/matches/"+mediaTestMatch+"/media", nil)
	request.Header.Set("Authorization", "Bearer "+mediaTestToken(t, secret, mediaTestOwner, servicekit.RoleTeamManager))
	handler.ServeHTTP(list, request)
	if list.Code != http.StatusOK || list.Body.String() != "[]\n" {
		t.Fatalf("empty list status=%d body=%s", list.Code, list.Body)
	}

	adminID := "99999999-9999-4999-8999-999999999991"
	assetID := "99999999-9999-4999-8999-999999999992"
	key := strings.Repeat("b", 32)
	seedMediaAsset(t, db, assetID, mediaTestOwner, key)
	adminToken := mediaTestToken(t, secret, adminID, servicekit.RoleSuperAdmin)
	delete := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodDelete, "/api/media/"+assetID, nil)
	request.Header.Set("Authorization", "Bearer "+adminToken)
	handler.ServeHTTP(delete, request)
	if delete.Code != http.StatusNoContent {
		t.Fatalf("admin delete status=%d body=%s", delete.Code, delete.Body)
	}
}

func TestMediaUploadCleansUpWhenMetadataSaveFails(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	secret := []byte("media-db-failure-test-secret")
	root := t.TempDir()
	handler := New(db, secret, root).Handler()
	request := mediaUploadRequest(t, mediaTestToken(t, secret, mediaTestOwner, servicekit.RoleTeamManager), tinyPNG)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("upload status=%d body=%s", response.Code, response.Body)
	}
	files, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("uploaded object left behind after database failure: %v", files)
	}
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/matches/" + mediaTestMatch + "/media"},
		{http.MethodGet, "/api/media/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"},
		{http.MethodDelete, "/api/media/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(tc.method, tc.path, nil)
		request.Header.Set("Authorization", "Bearer "+mediaTestToken(t, secret, mediaTestOwner, servicekit.RoleTeamManager))
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("%s %s status=%d body=%s", tc.method, tc.path, response.Code, response.Body)
		}
	}
}

func seedMediaAsset(t *testing.T, db *sql.DB, id, owner, key string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `INSERT INTO media_assets
		(id,owner_id,object_key,content_type,size_bytes,match_id,media_kind,caption)
		VALUES($1::uuid,$2::uuid,$3,'image/png',8,$4::uuid,'photo','test')`, id, owner, key, mediaTestMatch)
	if err != nil {
		t.Fatal(err)
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
