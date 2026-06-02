package handler

import (
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sastromikus/pip_shortener/internal/audit"
	"github.com/sastromikus/pip_shortener/internal/handler/middleware"
	"github.com/sastromikus/pip_shortener/internal/service"
)

const maxPOSTBody = 8 << 10

// NewRouter builds the HTTP router with all application handlers and middleware.
func NewRouter(svc *service.Shortener, baseURL string, logger *slog.Logger, db *sql.DB, auditor *audit.Notifier) http.Handler {
	baseURL = strings.TrimRight(baseURL, "/")

	r := chi.NewRouter()
	r.Use(middleware.Auth())
	r.Use(middleware.Gzip())
	r.Use(middleware.Logger(logger))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) { writeStatus(w, http.StatusBadRequest) })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) { writeStatus(w, http.StatusBadRequest) })

	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		handleShorten(svc, baseURL, logger, w, r, auditor)
	})
	r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		handleAPIPostShortenJSON(svc, baseURL, logger, w, r, auditor)
	})
	r.Post("/api/shorten/batch", func(w http.ResponseWriter, r *http.Request) {
		handleAPIPostShortenBatchJSON(svc, baseURL, logger, w, r)
	})
	r.Get("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {
		handleUserURLs(svc, baseURL, logger, w, r)
	})
	r.Delete("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {
		handleDeleteUserURLs(svc, logger, w, r)
	})
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		handlePing(db, w, r)
	})
	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		handleRedirect(svc, chi.URLParam(r, "id"), logger, w, r, auditor)
	})

	return r
}

func handleShorten(svc *service.Shortener, baseURL string, logger *slog.Logger, w http.ResponseWriter, r *http.Request, auditor *audit.Notifier) {
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	ce := strings.ToLower(r.Header.Get("Content-Encoding"))
	if ct != "" && !(strings.HasPrefix(ct, "text/plain") || (strings.Contains(ce, "gzip") && strings.HasPrefix(ct, "application/x-gzip"))) {
		writeStatus(w, http.StatusBadRequest)
		return
	}

	body, err := readBody(w, r, maxPOSTBody)
	if err != nil {
		writeStatus(w, http.StatusBadRequest)
		return
	}

	raw := strings.TrimSpace(string(body))
	if raw == "" {
		writeStatus(w, http.StatusBadRequest)
		return
	}

	userID, _ := middleware.UserIDFromContext(r.Context())
	id, existed, err := svc.ShortenForUserContext(r.Context(), raw, userID)
	if err != nil {
		status := statusFromServiceError(err)
		if status == http.StatusInternalServerError {
			logger.Error("shorten failed", "error", err)
		}
		writeStatus(w, status)
		return
	}

	notifyAudit(logger, r, auditor, audit.Event{Action: "shorten", UserID: userID, URL: raw})

	w.Header().Set("Content-Type", "text/plain")
	if existed {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}

	_, _ = w.Write([]byte(joinURL(baseURL, id)))
}

func handleRedirect(svc *service.Shortener, id string, logger *slog.Logger, w http.ResponseWriter, r *http.Request, auditor *audit.Notifier) {
	id = strings.TrimSpace(id)
	if id == "" || strings.ContainsAny(id, " \t\r\n") {
		writeStatus(w, http.StatusBadRequest)
		return
	}

	original, ok, deleted := svc.ResolveWithDeleted(r.Context(), id)
	if !ok {
		writeStatus(w, http.StatusBadRequest)
		return
	}

	if deleted {
		writeStatus(w, http.StatusGone)
		return
	}

	userID, _ := middleware.UserIDFromContext(r.Context())
	notifyAudit(logger, r, auditor, audit.Event{Action: "follow", UserID: userID, URL: original})

	w.Header().Set("Location", original)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func notifyAudit(logger *slog.Logger, r *http.Request, auditor *audit.Notifier, event audit.Event) {
	if auditor == nil || !auditor.Enabled() {
		return
	}

	if err := auditor.NotifyAll(r.Context(), event); err != nil && logger != nil {
		logger.Error("audit notify failed", "error", err)
	}
}

func readBody(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, error) {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	return io.ReadAll(r.Body)
}

func joinURL(baseURL string, id string) string {
	joined, err := url.JoinPath(baseURL, id)
	if err != nil {
		return strings.TrimRight(baseURL, "/") + "/" + id
	}
	return joined
}

func statusFromServiceError(err error) int {
	if errors.Is(err, service.ErrEmptyURL) ||
		errors.Is(err, service.ErrUnsupportedScheme) ||
		errors.Is(err, service.ErrEmptyHost) {
		return http.StatusBadRequest
	}

	return http.StatusInternalServerError
}

func writeStatus(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func internalServerError(logger *slog.Logger, w http.ResponseWriter, msg string, err error) {
	if logger != nil && err != nil {
		logger.Error(msg, "error", err)
	}

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
