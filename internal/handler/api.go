package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/audit"
	"github.com/sastromikus/pip_shortener/internal/handler/middleware"
	"github.com/sastromikus/pip_shortener/internal/service"
)

type apiShortenRequest struct {
	URL string `json:"url"`
}

type apiShortenResponse struct {
	Result string `json:"result"`
}

func handleAPIPostShortenJSON(svc *service.Shortener, baseURL string, logger *slog.Logger, w http.ResponseWriter, r *http.Request, auditor *audit.Notifier) {
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

	var req apiShortenRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeStatus(w, http.StatusBadRequest)
		return
	}

	raw := strings.TrimSpace(req.URL)
	if raw == "" {
		writeStatus(w, http.StatusBadRequest)
		return
	}

	userID, _ := middleware.UserIDFromContext(r.Context())
	id, existed, err := svc.ShortenForUserContext(r.Context(), raw, userID)
	if err != nil {
		status := statusFromServiceError(err)
		if status == http.StatusInternalServerError {
			logger.Error("api shorten failed", "error", err)
		}
		writeStatus(w, status)
		return
	}

	notifyAudit(logger, auditor, audit.Event{Action: "shorten", UserID: userID, URL: raw})

	w.Header().Set("Content-Type", "application/json")
	if existed {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	if err := json.NewEncoder(w).Encode(apiShortenResponse{Result: joinURL(baseURL, id)}); err != nil {
		internalServerError(logger, w, "encode api shorten response", err)
		return
	}
}
