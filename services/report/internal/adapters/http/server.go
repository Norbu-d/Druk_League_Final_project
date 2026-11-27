package httpadapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/report/internal/app"
	"github.com/drukleague/platform/services/report/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	service *app.Service
	secret  []byte
}

func New(service *app.Service, secret []byte) http.Handler {
	s := &Server{service: service, secret: secret}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/reports", s.create)
	mux.HandleFunc("GET /api/reports/{id}", s.get)
	mux.HandleFunc("GET /api/reports/files/{id}", s.download)
	return s.authenticate(mux)
}

type reportResponse struct {
	ID           string `json:"id"`
	ReportType   string `json:"report_type"`
	Status       string `json:"status"`
	DownloadURL  string `json:"download_url"`
	CreatedAt    string `json:"created_at"`
	TournamentID string `json:"tournament_id"`
	Format       string `json:"format"`
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var request struct {
		ReportType   string `json:"report_type"`
		TournamentID string `json:"tournament_id"`
		Format       string `json:"format"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return
	}
	caller := callerFrom(r)
	item, err := s.service.Create(r.Context(), caller, domain.CreateRequest{ReportType: request.ReportType, TournamentID: request.TournamentID, Format: request.Format})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(item))
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	item, err := s.service.Get(r.Context(), callerFrom(r), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(item))
}

func (s *Server) download(w http.ResponseWriter, r *http.Request) {
	item, err := s.service.Download(r.Context(), callerFrom(r), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", item.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, app.SafeFileName(item)))
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(item.Content)
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		token, found := strings.CutPrefix(authorization, "Bearer ")
		if !found || strings.TrimSpace(token) == "" {
			writeError(w, http.StatusUnauthorized, "bearer token required")
			return
		}
		identity, err := servicekit.VerifyToken(strings.TrimSpace(token), s.secret)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired bearer token")
			return
		}
		ctx := servicekit.WithIdentity(r.Context(), identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func callerFrom(r *http.Request) app.Caller {
	identity, _ := servicekit.IdentityFromContext(r.Context())
	return app.Caller{ID: identity.ID, Role: identity.Role}
}

func toResponse(item domain.Report) reportResponse {
	return reportResponse{ID: item.ID, ReportType: item.ReportType, Status: item.Status, DownloadURL: item.DownloadURL,
		CreatedAt: item.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"), TournamentID: item.TournamentID, Format: item.Format}
}

func writeServiceError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch {
	case errors.Is(err, core.ErrInvalid):
		code = http.StatusBadRequest
	case errors.Is(err, core.ErrForbidden):
		code = http.StatusForbidden
	case errors.Is(err, core.ErrUnauthenticated):
		code = http.StatusUnauthorized
	case errors.Is(err, core.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, core.ErrDependency):
		code = http.StatusServiceUnavailable
	case errors.Is(err, core.ErrPrecondition):
		code = http.StatusFailedDependency
	default:
		if grpcCode := status.Code(err); grpcCode != codes.Unknown {
			switch grpcCode {
			case codes.InvalidArgument:
				code = http.StatusBadRequest
			case codes.PermissionDenied:
				code = http.StatusForbidden
			case codes.NotFound:
				code = http.StatusNotFound
			case codes.Unavailable, codes.DeadlineExceeded:
				code = http.StatusServiceUnavailable
			}
		}
	}
	writeError(w, code, http.StatusText(code))
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}
