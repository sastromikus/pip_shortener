package handler

import (
	"encoding/json"
	"net/http"
	"strings"

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
		badRequest(w)
		return
	}

	body, err := readBody(r, maxPOSTBody)
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

	baseURL = strings.TrimRight(baseURL, "/")

	out := make([]apiBatchResponseItem, 0, len(in))
	for _, item := range in {
		cid := strings.TrimSpace(item.CorrelationID)
		orig := strings.TrimSpace(item.OriginalURL)
		if cid == "" || orig == "" {
			badRequest(w)
			return
		}

		id, err := svc.Shorten(orig)
		if err != nil {
			badRequest(w)
			return
		}

		out = append(out, apiBatchResponseItem{
			CorrelationID: cid,
			ShortURL:      baseURL + "/" + id,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}