package handler

import (
	"encoding/json"
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

func handleAPIPostShortenBatchJSON(svc *service.Shortener, baseURL string, w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if ct == "" || !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		writeStatus(w, http.StatusBadRequest)
		return
	}

	body, err := readBody(w, r, maxPOSTBody)
	if err != nil {
		writeStatus(w, http.StatusBadRequest)
		return
	}

	var in []apiBatchRequestItem
	if err := json.Unmarshal(body, &in); err != nil {
		writeStatus(w, http.StatusBadRequest)
		return
	}
	if len(in) == 0 {
		writeStatus(w, http.StatusBadRequest)
		return
	}

	items := make([]service.BatchItem, 0, len(in))
	for _, item := range in {
		cid := strings.TrimSpace(item.CorrelationID)
		orig := strings.TrimSpace(item.OriginalURL)
		if cid == "" || orig == "" {
			writeStatus(w, http.StatusBadRequest)
			return
		}
		items = append(items, service.BatchItem{CorrelationID: cid, OriginalURL: orig})
	}

	userID, _ := middleware.UserIDFromContext(r.Context())
	results, err := svc.ShortenBatch(items, userID)
	if err != nil {
		writeStatus(w, statusFromServiceError(err))
		return
	}

	out := make([]apiBatchResponseItem, 0, len(results))
	for _, result := range results {
		out = append(out, apiBatchResponseItem{
			CorrelationID: result.CorrelationID,
			ShortURL:      joinURL(baseURL, result.ID),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}
