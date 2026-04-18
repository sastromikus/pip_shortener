package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sastromikus/pip_shortener/internal/service"

    "github.com/sirupsen/logrus"
    "github.com/sastromikus/pip_shortener/internal/handler/middleware"
)

const maxPOSTBody = 8 << 10 

func NewRouter(svc *service.Shortener, baseURL string, logger *logrus.Logger) http.Handler {
	baseURL = strings.TrimRight(baseURL, "/")

    r := chi.NewRouter()
    r.Use(middleware.Logger(logger))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) { badRequest(w) })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) { badRequest(w) })

	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		handleShorten(svc, baseURL, w, r)
	})

	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		handleRedirect(svc, id, w, r)
	})

	return r
}

func handleShorten(svc *service.Shortener, baseURL string, w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if ct == "" || !strings.HasPrefix(strings.ToLower(ct), "text/plain") {
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

	id, err := svc.Shorten(raw)
	if err != nil {
		badRequest(w)
		return
	}

	shortURL := baseURL + "/" + id

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
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