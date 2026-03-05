package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/service"
)

const maxPOSTBody = 8 << 10

type Router struct {
	svc *service.Shortener
}

func NewRouter(svc *service.Shortener) *Router {
	return &Router{svc: svc}
}

func (h *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if r.URL.Path != "/" {
			badRequest(w)
			return
		}
		h.handleShorten(w, r)

	case http.MethodGet:
		if r.URL.Path == "/" {
			badRequest(w)
			return
		}
		h.handleRedirect(w, r)

	default:
		badRequest(w)
	}
}

func (h *Router) handleShorten(w http.ResponseWriter, r *http.Request) {
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

	id, err := h.svc.Shorten(raw)
	if err != nil {
		badRequest(w)
		return
	}

	shortURL := fmt.Sprintf("http://localhost:8080/%s", id)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(shortURL))
}

func (h *Router) handleRedirect(w http.ResponseWriter, r *http.Request) {
	if strings.Count(r.URL.Path, "/") != 1 {
		badRequest(w)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" || strings.ContainsAny(id, " \t\r\n") {
		badRequest(w)
		return
	}

	original, ok := h.svc.Resolve(id)
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