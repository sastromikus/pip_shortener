package handler

import (
	"encoding/json"
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

func handleAPIPostShortenJSON(svc *service.Shortener, baseURL string, w http.ResponseWriter, r *http.Request) {
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

	userID, _ := middleware.UserIDFromContext(r.Context())
	id, existed, err := svc.ShortenForUser(raw, userID)
	if err != nil {
		badRequest(w)
		return
	}

	shortURL := strings.TrimRight(baseURL, "/") + "/" + id
	resp := apiShortenResponse{Result: shortURL}

	w.Header().Set("Content-Type", "application/json")
	if existed {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(resp)
}
