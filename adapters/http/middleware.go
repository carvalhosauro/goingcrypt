package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/carvalhosauro/goingcrypt/internal/ports"
)

// AuthMiddleware injects the authenticated userID into the request context when
// a valid Bearer token is present. Routes that only optionally need auth use this.
func AuthMiddleware(tm ports.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if token, ok := strings.CutPrefix(header, "Bearer "); ok {
				if userID, err := tm.ValidateAccessToken(r.Context(), token); err == nil {
					r = r.WithContext(context.WithValue(r.Context(), userIDKey, userID))
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
