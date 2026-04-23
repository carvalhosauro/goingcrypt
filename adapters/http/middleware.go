package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports"
)

// AuthMiddleware injects the authenticated userID and role into the request
// context when a valid Bearer token is present.
func AuthMiddleware(tm ports.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if token, ok := strings.CutPrefix(header, "Bearer "); ok {
				if userID, role, err := tm.ValidateAccessToken(r.Context(), token); err == nil {
					ctx := context.WithValue(r.Context(), userIDKey, userID)
					ctx = context.WithValue(ctx, userRoleKey, role)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth rejects the request with 401 if no authenticated userID is
// present in the context. Use it on sub-routers that require a valid session.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value(userIDKey).(interface{ String() string }); !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin rejects the request with 403 if the authenticated user does not
// have the admin role. Reads role from context set by AuthMiddleware.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if role, _ := r.Context().Value(userRoleKey).(domain.UserRole); role != domain.UserRoleAdmin {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}
