package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/handler/middleware"
	"github.com/sastromikus/pip_shortener/internal/service"
)

type apiBatchRequestItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type apiBatchResponseItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

func handleAPIPostShortenBatchJSON(svc *service.Shortener, baseURL string, logger *slog.Logger, w http.ResponseWriter, r *http.Request) {
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

	var in []apiBatchRequestItem
	if err := json.Unmarshal(body, &in); err != nil {
		badRequest(w)
		return
	}
	if len(in) == 0 {
		badRequest(w)
		return
	}

	items := make([]service.BatchItem, 0, len(in))
	for _, item := range in {
		cid := strings.TrimSpace(item.CorrelationID)
		orig := strings.TrimSpace(item.OriginalURL)
		if cid == "" || orig == "" {
			badRequest(w)
			return
		}

		items = append(items, service.BatchItem{
			CorrelationID: cid,
			OriginalURL:   orig,
		})
	}

	userID, _ := middleware.UserIDFromContext(r.Context())

	results, err := svc.ShortenBatchForUser(items, userID)
	if err != nil {
		writeShortenError(logger, w, err)
		return
	}

	out := make([]apiBatchResponseItem, 0, len(results))
	for _, item := range results {
		shortURL, err := buildShortURL(baseURL, item.ID)
		if err != nil {
			internalServerError(logger, w, "build batch short url", err)
			return
		}

		out = append(out, apiBatchResponseItem{
			CorrelationID: item.CorrelationID,
			ShortURL:      shortURL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(out); err != nil {
		internalServerError(logger, w, "encode batch shorten response", err)
		return
	}
}
