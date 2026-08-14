package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/norest-mail/server/internal/response"
)

type contextKey string

const userContextKey contextKey = "userID"
const userIDStringKey contextKey = "user_id"

// RequireAuth middleware verifies the JWT and attaches the user ID to the context.
func RequireAuth(service *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Error(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			tokenStr := parts[1]
			userID, err := service.ValidateToken(tokenStr)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, userID)
			ctx = context.WithValue(ctx, userIDStringKey, userID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin middleware verifies the JWT, fetches the user, and ensures they have the 'admin' role.
func RequireAdmin(service *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Error(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			tokenStr := parts[1]
			userID, err := service.ValidateToken(tokenStr)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			user, err := service.GetUserByID(r.Context(), userID)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "user not found")
				return
			}

			if user.Role != "admin" {
				response.Error(w, http.StatusForbidden, "admin privileges required")
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, userID)
			ctx = context.WithValue(ctx, userIDStringKey, userID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext retrieves the user ID from the request context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userContextKey).(uuid.UUID)
	return userID, ok
}
