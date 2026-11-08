package httpadapter

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
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
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("match_id", mediaTestMatch)
	_ = writer.WriteField("media_kind", "photo")
	_ = writer.WriteField("caption", "Match day")
	file, err := writer.CreateFormFile("file", "asset.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
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
