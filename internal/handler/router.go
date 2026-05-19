package handler

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/sastromikus/pip_shortener/internal/handler/middleware"
	"github.com/sastromikus/pip_shortener/internal/service"
)

const maxPOSTBody = 8 << 10

func NewRouter(svc *service.Shortener, baseURL string, logger *logrus.Logger, db *sql.DB) http.Handler {
	baseURL = strings.TrimRight(baseURL, "/")

	r := chi.NewRouter()
	r.Use(middleware.Gzip())
	r.Use(middleware.Logger(logger))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) { badRequest(w) })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) { badRequest(w) })

	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		handleShorten(svc, baseURL, w, r)
	})

	r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		handleAPIPostShortenJSON(svc, baseURL, w, r)
	})

	r.Post("/api/shorten/batch", func(w http.ResponseWriter, r *http.Request) {
		handleAPIPostShortenBatchJSON(svc, baseURL, w, r)
	})

	r.Get("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {
		handleUserURLs(svc, baseURL, w, r)
	})

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		handlePing(db, w, r)
	})

	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		handleRedirect(svc, id, w, r)
	})

	return r
}

func handleShorten(svc *service.Shortener, baseURL string, w http.ResponseWriter, r *http.Request) {
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if ct != "" && !strings.HasPrefix(ct, "text/plain") && !strings.HasPrefix(ct, "application/x-gzip") {
		badRequest(w)
		return
	}

	body, err := readBody(w, r, maxPOSTBody)
	if err != nil {
		badRequest(w)
		return
	}

	raw := strings.TrimSpace(string(body))
	if raw == "" {
		badRequest(w)
		return
	}

	userID, err := getOrCreateUserID(w, r)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	id, existed, err := svc.ShortenWithExistingForUser(raw, userID)
	if err != nil {
		writeShortenError(w, err)
		return
	}

	shortURL, err := buildShortURL(baseURL, id)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if existed {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}

	_, _ = w.Write([]byte(shortURL))
}

func handleRedirect(svc *service.Shortener, id string, w http.ResponseWriter, r *http.Request) {
	id = strings.TrimSpace(id)
	if id == "" || strings.ContainsAny(id, " \t\r\n") {
		badRequest(w)
		return
	}

	original, ok := svc.Resolve(id)
	if !ok {
		badRequest(w)
		return
	}

	w.Header().Set("Location", original)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func readBody(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, error) {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	return io.ReadAll(r.Body)
}

func buildShortURL(baseURL string, id string) (string, error) {
	return url.JoinPath(baseURL, id)
}

func writeShortenError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrGenerateID) || errors.Is(err, service.ErrStorage) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	badRequest(w)
}

func badRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
}
