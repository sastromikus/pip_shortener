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

	userID, err := getOrCreateUserID(w, r)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	id, err := svc.ShortenForUser(raw, userID)
	if err != nil {
		writeShortenError(w, err)
		return
	}

	shortURL, err := buildShortURL(baseURL, id)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(apiShortenResponse{Result: shortURL})
}
