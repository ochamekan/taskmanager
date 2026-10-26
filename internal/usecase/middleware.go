package usecase

import (
	"context"
	"net/http"
	"strings"

	"github.com/example/taskmanager/internal/auth"
)

type contextKey int

const userIDKey contextKey = iota

func currentUserID(r *http.Request) int64 {
	id, _ := r.Context().Value(userIDKey).(int64)
	return id
}

func (u *Usecase) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeErrorBody(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			return
		}
		userID, err := auth.ParseToken(strings.TrimSpace(token), u.jwtSecret)
		if err != nil {
			writeErrorBody(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userIDKey, userID)))
	}
}
