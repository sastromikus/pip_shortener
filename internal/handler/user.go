package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/handler/middleware"
	"github.com/sastromikus/pip_shortener/internal/service"
)

type userURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func handleUserURLs(svc *service.Shortener, baseURL string, logger *slog.Logger, w http.ResponseWriter, r *http.Request) {
	if middleware.BadCookieNoID(r.Context()) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	items := svc.UserURLs(userID)
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]userURLResponse, 0, len(items))
	for _, item := range items {
		shortURL, err := buildShortURL(baseURL, item.ID)
		if err != nil {
			internalServerError(logger, w, "build user short url", err)
			return
		}

		resp = append(resp, userURLResponse{
			ShortURL:    shortURL,
			OriginalURL: item.OriginalURL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		internalServerError(logger, w, "encode user urls response", err)
		return
	}
}
