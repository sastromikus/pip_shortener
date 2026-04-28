package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/handler/middleware"
	"github.com/sastromikus/pip_shortener/internal/service"
)

type apiShortenRequest struct {
	URL string `json:"url"`
}

type apiShortenResponse struct {
	Result string `json:"result"`
}

func handleAPIPostShortenJSON(svc *service.Shortener, baseURL string, logger *slog.Logger, w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if ct == "" || !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		badRequest(w)
		return
	}

	body, err := readBody(w, r, maxPOSTBody)
	if err != nil {
		badRequest(w)
		return
	}

	var req apiShortenRequest
	if err := json.Unmarshal(body, &req); err != nil {
		badRequest(w)
		return
	}

	raw := strings.TrimSpace(req.URL)
	if raw == "" {
		badRequest(w)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		internalServerError(logger, w, "get user id from request context", service.ErrStorage)
		return
	}

	id, existed, err := svc.ShortenWithExistingForUser(raw, userID)
	if err != nil {
		writeShortenError(logger, w, err)
		return
	}

	shortURL, err := buildShortURL(baseURL, id)
	if err != nil {
		internalServerError(logger, w, "build short url", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if existed {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}

	if err := json.NewEncoder(w).Encode(apiShortenResponse{Result: shortURL}); err != nil {
		internalServerError(logger, w, "encode api shorten response", err)
		return
	}
}
