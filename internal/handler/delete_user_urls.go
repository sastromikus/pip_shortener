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
        w.WriteHeader(http.StatusUnauthorized)
        return
    }

    userID, ok := middleware.UserIDFromContext(r.Context())
    if !ok || strings.TrimSpace(userID) == "" {
        w.WriteHeader(http.StatusUnauthorized)
        return
    }

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

    var ids []string
    if err := json.Unmarshal(body, &ids); err != nil {
        badRequest(w)
        return
    }
    if len(ids) == 0 {
        badRequest(w)
        return
    }

    svc.EnqueueDelete(userID, ids)
    w.WriteHeader(http.StatusAccepted)
}