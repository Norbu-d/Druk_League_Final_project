package httpadapter

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
)

const maxUploadBytes = 10 << 20

type Server struct {
	db       *sql.DB
	secret   []byte
	dataRoot string
}

type asset struct {
	ID              string    `json:"id"`
	MatchID         string    `json:"match_id"`
	OwnerID         string    `json:"owner_id"`
	MediaKind       string    `json:"media_kind"`
	Caption         string    `json:"caption"`
	ContentType     string    `json:"content_type"`
	SizeBytes       int64     `json:"size_bytes"`
	DurationSeconds *int64    `json:"duration_seconds,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

func New(db *sql.DB, secret []byte, dataRoot string) *Server {
	return &Server{db: db, secret: secret, dataRoot: dataRoot}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/media", s.upload)
	mux.HandleFunc("GET /api/media/{id}", s.download)
	mux.HandleFunc("DELETE /api/media/{id}", s.remove)
	mux.HandleFunc("GET /api/matches/{matchID}/media", s.list)
	return mux
}

func (s *Server) identity(r *http.Request) (servicekit.Identity, error) {
	token := ""
	if cookie, err := r.Cookie("drukleague-session"); err == nil {
		token = cookie.Value
	}
	if token == "" {
		authorization := r.Header.Get("Authorization")
		if strings.HasPrefix(authorization, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
		}
	}
	if token == "" {
		return servicekit.Identity{}, errors.New("authentication required")
	}
	identity, err := servicekit.VerifyToken(token, s.secret)
	if err != nil {
		return servicekit.Identity{}, errors.New("invalid or expired session")
	}
	return identity, nil
}

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	identity, err := s.identity(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil { // #nosec G120 -- MaxBytesReader above caps the complete request body.
		writeError(w, http.StatusBadRequest, "upload must be a valid multipart form under 10 MiB")
		return
	}
	defer r.MultipartForm.RemoveAll()
	matchID := strings.TrimSpace(r.FormValue("match_id"))
	kind := strings.TrimSpace(r.FormValue("media_kind"))
	caption := strings.TrimSpace(r.FormValue("caption"))
	if !core.IsUUID(matchID) || (kind != "photo" && kind != "highlight") || len(caption) > 1000 {
		writeError(w, http.StatusBadRequest, "match_id, media_kind (photo or highlight), or caption is invalid")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil || len(content) == 0 || len(content) > maxUploadBytes {
		writeError(w, http.StatusBadRequest, "file must contain between 1 byte and 10 MiB")
		return
	}
	contentType := http.DetectContentType(content[:min(len(content), 512)])
	if !allowedContentType(contentType, kind) {
		writeError(w, http.StatusUnsupportedMediaType, "only JPEG, PNG, WebP images and MP4 highlights are supported")
		return
	}
	if kind == "photo" && contentType == "video/mp4" || kind == "highlight" && contentType != "video/mp4" {
		writeError(w, http.StatusUnsupportedMediaType, "photo uploads must be images and highlights must be MP4 video")
		return
	}
	key, err := randomKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not allocate media storage key")
		return
	}
	if err := os.MkdirAll(s.dataRoot, 0o750); err != nil {
		writeError(w, http.StatusInternalServerError, "media storage is unavailable")
		return
	}
	storage, err := os.OpenRoot(s.dataRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "media storage is unavailable")
		return
	}
	defer storage.Close()
	storedFile, err := storage.OpenFile(key, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not store media file")
		return
	}
	if _, err = storedFile.Write(content); err != nil {
		_ = storedFile.Close()
		_ = storage.Remove(key)
		writeError(w, http.StatusInternalServerError, "could not store media file")
		return
	}
	if err := storedFile.Close(); err != nil {
		_ = storage.Remove(key)
		writeError(w, http.StatusInternalServerError, "could not store media file")
		return
	}
	var saved asset
	err = s.db.QueryRowContext(r.Context(), `
		INSERT INTO media_assets (owner_id, object_key, content_type, size_bytes, match_id, media_kind, caption)
		VALUES ($1::uuid,$2,$3,$4,$5::uuid,$6,$7)
		RETURNING id::text,match_id::text,owner_id::text,media_kind,caption,content_type,size_bytes,duration_seconds,created_at`,
		identity.ID, key, contentType, len(content), matchID, kind, caption,
	).Scan(&saved.ID, &saved.MatchID, &saved.OwnerID, &saved.MediaKind, &saved.Caption, &saved.ContentType, &saved.SizeBytes, &saved.DurationSeconds, &saved.CreatedAt)
	if err != nil {
		_ = storage.Remove(key)
		writeError(w, http.StatusInternalServerError, "could not save media metadata")
		return
	}
	w.Header().Set("Location", "/api/media/"+saved.ID)
	writeJSON(w, http.StatusCreated, saved)
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	if _, err := s.identity(r); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	matchID := r.PathValue("matchID")
	if !core.IsUUID(matchID) {
		writeError(w, http.StatusBadRequest, "match id is invalid")
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id::text,match_id::text,owner_id::text,media_kind,caption,content_type,size_bytes,duration_seconds,created_at
		FROM media_assets WHERE match_id=$1::uuid ORDER BY created_at DESC,id`, matchID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list match media")
		return
	}
	defer rows.Close()
	assets := make([]asset, 0)
	for rows.Next() {
		var item asset
		if err := rows.Scan(&item.ID, &item.MatchID, &item.OwnerID, &item.MediaKind, &item.Caption, &item.ContentType, &item.SizeBytes, &item.DurationSeconds, &item.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read match media")
			return
		}
		assets = append(assets, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read match media")
		return
	}
	writeJSON(w, http.StatusOK, assets)
}

func (s *Server) download(w http.ResponseWriter, r *http.Request) {
	if _, err := s.identity(r); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	mediaID := r.PathValue("id")
	if !core.IsUUID(mediaID) {
		writeError(w, http.StatusBadRequest, "media id is invalid")
		return
	}
	var item asset
	var key string
	err := s.db.QueryRowContext(r.Context(), `
		SELECT id::text,match_id::text,owner_id::text,media_kind,caption,content_type,size_bytes,duration_seconds,created_at,object_key
		FROM media_assets WHERE id=$1::uuid`, mediaID).Scan(&item.ID, &item.MatchID, &item.OwnerID, &item.MediaKind, &item.Caption, &item.ContentType, &item.SizeBytes, &item.DurationSeconds, &item.CreatedAt, &key)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "media not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read media metadata")
		return
	}
	if !validStorageKey(key) {
		writeError(w, http.StatusInternalServerError, "media storage key is invalid")
		return
	}
	storage, err := os.OpenRoot(s.dataRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "media storage is unavailable")
		return
	}
	defer storage.Close()
	file, err := storage.Open(key)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "media file is missing")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read media file")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil || len(data) > maxUploadBytes {
		writeError(w, http.StatusInternalServerError, "could not read media file")
		return
	}
	w.Header().Set("Content-Type", item.ContentType)
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) remove(w http.ResponseWriter, r *http.Request) {
	identity, err := s.identity(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	mediaID := r.PathValue("id")
	if !core.IsUUID(mediaID) {
		writeError(w, http.StatusBadRequest, "media id is invalid")
		return
	}
	var owner, key string
	err = s.db.QueryRowContext(r.Context(), `SELECT owner_id::text,object_key FROM media_assets WHERE id=$1::uuid`, mediaID).Scan(&owner, &key)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "media not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read media metadata")
		return
	}
	if owner != identity.ID && identity.Role != servicekit.RoleSuperAdmin {
		writeError(w, http.StatusForbidden, "only the uploader or an administrator can delete this media")
		return
	}
	result, err := s.db.ExecContext(r.Context(), `DELETE FROM media_assets WHERE id=$1::uuid AND owner_id=$2::uuid`, mediaID, owner)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete media metadata")
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil || deleted != 1 {
		writeError(w, http.StatusConflict, "media changed during deletion")
		return
	}
	if !validStorageKey(key) {
		writeError(w, http.StatusInternalServerError, "media storage key is invalid")
		return
	}
	storage, err := os.OpenRoot(s.dataRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "media storage is unavailable")
		return
	}
	defer storage.Close()
	if err := storage.Remove(key); err != nil && !errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusInternalServerError, "media metadata was deleted but stored file cleanup failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func allowedContentType(contentType, kind string) bool {
	if kind == "highlight" {
		return contentType == "video/mp4"
	}
	return contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/webp"
}

func randomKey() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(id[:]), nil
}

func validStorageKey(key string) bool {
	decoded, err := hex.DecodeString(key)
	return err == nil && len(decoded) == 16
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
