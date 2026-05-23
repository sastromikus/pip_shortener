package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/handler/middleware"
	"github.com/sastromikus/pip_shortener/internal/service"
)

func handleDeleteUserURLs(svc *service.Shortener, w http.ResponseWriter, r *http.Request) {
	if middleware.BadCookieNoID(r.Context()) {
		writeStatus(w, http.StatusUnauthorized)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		writeStatus(w, http.StatusUnauthorized)
		return
	}

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

	var ids []string
	if err := json.Unmarshal(body, &ids); err != nil {
		writeStatus(w, http.StatusBadRequest)
		return
	}
	if len(ids) == 0 {
		writeStatus(w, http.StatusBadRequest)
		return
	}

	svc.EnqueueDelete(userID, ids)
	w.WriteHeader(http.StatusAccepted)
}
