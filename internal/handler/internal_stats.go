package handler

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/service"
)

type statsResponse struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}

func trustedSubnetOnly(cidr *net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP")))
			if ip == nil || cidr == nil || !cidr.Contains(ip) {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func handleInternalStats(svc *service.Shortener, logger *slog.Logger, w http.ResponseWriter, r *http.Request) {
	urls, users, err := svc.Stats(r.Context())
	if err != nil {
		internalServerError(logger, w, "get internal stats", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(statsResponse{URLs: urls, Users: users})
}
