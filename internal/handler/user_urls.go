package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/handler/middleware"
	"github.com/sastromikus/pip_shortener/internal/repository"
	"github.com/sastromikus/pip_shortener/internal/service"
)

type userURLResponseItem struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func handleGetUserURLs(svc *service.Shortener, baseURL string, w http.ResponseWriter, r *http.Request) {
	if middleware.BadCookieNoID(r.Context()) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	items, err := svc.ListUserURLs(userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	baseURL = strings.TrimRight(baseURL, "/")
	out := make([]userURLResponseItem, 0, len(items))
	for _, it := range items {
		out = append(out, userURLResponseItem{
			ShortURL:    baseURL + "/" + it.ShortID,
			OriginalURL: it.Original,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}

var _ = repository.UserURL{}