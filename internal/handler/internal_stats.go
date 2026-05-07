package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/service"
)

type statsResponse struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}

func handleInternalStats(svc *service.Shortener, trustedSubnet string, w http.ResponseWriter, r *http.Request) {
	trustedSubnet = strings.TrimSpace(trustedSubnet)
	if trustedSubnet == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	_, cidr, err := net.ParseCIDR(trustedSubnet)
	if err != nil || cidr == nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	ipStr := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	ip := net.ParseIP(ipStr)
	if ip == nil || !cidr.Contains(ip) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	urls, users, err := svc.Stats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(statsResponse{URLs: urls, Users: users})
}