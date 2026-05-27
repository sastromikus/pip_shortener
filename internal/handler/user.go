package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/sastromikus/pip_shortener/internal/service"
)

const userCookieName = "user_id"

type userURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func handleUserURLs(svc *service.Shortener, baseURL string, w http.ResponseWriter, r *http.Request) {
	userID, err := getOrCreateUserID(w, r)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	items := svc.UserURLs(userID)
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]userURLResponse, 0, len(items))
	for _, item := range items {
		shortURL, err := buildShortURL(baseURL, item.ID)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		resp = append(resp, userURLResponse{
			ShortURL:    shortURL,
			OriginalURL: item.OriginalURL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func getOrCreateUserID(w http.ResponseWriter, r *http.Request) (string, error) {
	if c, err := r.Cookie(userCookieName); err == nil && c.Value != "" {
		return c.Value, nil
	}

	userID, err := randomUserID()
	if err != nil {
		return "", err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     userCookieName,
		Value:    userID,
		Path:     "/",
		HttpOnly: true,
	})

	return userID, nil
}

func randomUserID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
