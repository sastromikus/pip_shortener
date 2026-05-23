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
		writeStatus(w, http.StatusUnauthorized)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		writeStatus(w, http.StatusUnauthorized)
		return
	}

	items, err := svc.ListUserURLs(userID)
	if err != nil {
		internalServerError(logger, w, "list user urls", err)
		return
	}
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]userURLResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, userURLResponse{
			ShortURL:    joinURL(baseURL, item.ShortID),
			OriginalURL: item.Original,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		internalServerError(logger, w, "encode user urls response", err)
		return
	}
}
