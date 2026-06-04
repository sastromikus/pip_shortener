package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

func handlePing(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if db == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		writeStatus(w, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
