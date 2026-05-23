package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/handler/middleware"
	"github.com/sastromikus/pip_shortener/internal/service"
)

type userURLResponseItem struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func handleGetUserURLs(svc *service.Shortener, baseURL string, w http.ResponseWriter, r *http.Request) {
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
		writeStatus(w, http.StatusInternalServerError)
		return
	}
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	out := make([]userURLResponseItem, 0, len(items))
	for _, it := range items {
		out = append(out, userURLResponseItem{
			ShortURL:    joinURL(baseURL, it.ShortID),
			OriginalURL: it.Original,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}
