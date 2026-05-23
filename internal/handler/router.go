package handler

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sastromikus/pip_shortener/internal/service"

	"github.com/sastromikus/pip_shortener/internal/handler/middleware"
	"github.com/sirupsen/logrus"
)

const maxPOSTBody = 8 << 10

func NewRouter(svc *service.Shortener, baseURL string, logger *logrus.Logger, db *sql.DB) http.Handler {
	baseURL = strings.TrimRight(baseURL, "/")

	r := chi.NewRouter()
	r.Use(middleware.Auth())
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

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		handlePing(db, w, r)
	})

	r.Get("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {
		handleGetUserURLs(svc, baseURL, w, r)
	})

	r.Delete("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {
		handleDeleteUserURLs(svc, w, r)
	})

	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		handleRedirect(svc, id, w, r)
	})

	return r
}

func handleShorten(svc *service.Shortener, baseURL string, w http.ResponseWriter, r *http.Request) {
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	ce := strings.ToLower(r.Header.Get("Content-Encoding"))
	if ct == "" || !(strings.HasPrefix(ct, "text/plain") || (strings.Contains(ce, "gzip") && strings.HasPrefix(ct, "application/x-gzip"))) {
		badRequest(w)
		return
	}

	body, err := readBody(r, maxPOSTBody)
	if err != nil {
		badRequest(w)
		return
	}

	raw := strings.TrimSpace(string(body))
	if raw == "" {
		badRequest(w)
		return
	}

	userID, _ := middleware.UserIDFromContext(r.Context())
	id, existed, err := svc.ShortenForUser(raw, userID)
	if err != nil {
		badRequest(w)
		return
	}

	shortURL := baseURL + "/" + id

	w.Header().Set("Content-Type", "text/plain")
	if existed {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	_, _ = w.Write([]byte(shortURL))
}

func handleRedirect(svc *service.Shortener, id string, w http.ResponseWriter, r *http.Request) {
	original, ok, deleted := svc.ResolveWithDeleted(id)
	if !ok {
		badRequest(w)
		return
	}
	if deleted {
		w.WriteHeader(http.StatusGone)
		return
	}

	w.Header().Set("Location", original)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func readBody(r *http.Request, limit int64) ([]byte, error) {
	defer r.Body.Close()
	lr := &io.LimitedReader{R: r.Body, N: limit + 1}
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("body too large")
	}
	return b, nil
}

func badRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
}
