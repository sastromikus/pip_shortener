package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/service"
)

type apiShortenRequest struct {
	URL string `json:"url"`
}

type apiShortenResponse struct {
	Result string `json:"result"`
}

func handleAPIPostShortenJSON(svc *service.Shortener, baseURL string, w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if ct == "" || !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		badRequest(w)
		return
	}

	var req apiShortenRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		badRequest(w)
		return
	}

	if dec.More() {
		badRequest(w)
		return
	}

	raw := strings.TrimSpace(req.URL)
	if raw == "" {
		badRequest(w)
		return
	}

	id, err := svc.Shorten(raw)
	if err != nil {
		badRequest(w)
		return
	}

	shortURL := strings.TrimRight(baseURL, "/") + "/" + id
	resp := apiShortenResponse{Result: shortURL}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(w)
	_ = enc.Encode(resp)
}