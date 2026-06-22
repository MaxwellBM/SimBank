package middleware

import (
	"context"
	"net/http"
	"strings"

	"banca-backend/internal/auth"
	"banca-backend/internal/db"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const UserEmailKey contextKey = "user_email"

func AuthMiddleware(store *db.PostgresStore, jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, `{"error":"invalid authorization format, expected Bearer <token>"}`, http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			claims, err := auth.ParseToken(tokenString, jwtSecret)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			tokenHash := auth.HashToken(tokenString)
			valid, err := store.IsSessionValid(r.Context(), tokenHash)
			if err != nil || !valid {
				http.Error(w, `{"error":"session was closed or expired"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
