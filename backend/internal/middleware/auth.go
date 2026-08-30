package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/session"
	"github.com/Arifinwidy02/splitmate-backend/pkg/response"
)

type userIDKey struct{}

func RequireAuth(sessionService *session.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(session.AccessTokenCookie)
			if err != nil {
				response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
				return
			}

			userID, err := sessionService.ParseAccessToken(cookie.Value)
			if err != nil {
				response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired session")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// OptionalAuth attaches the user id to the request context when a valid
// access token cookie is present, but never rejects the request.
func OptionalAuth(sessionService *session.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(session.AccessTokenCookie)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			userID, err := sessionService.ParseAccessToken(cookie.Value)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	return userID, ok
}
