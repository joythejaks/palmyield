// Package middleware holds PalmYield-specific HTTP middleware (JWT auth,
// RBAC) as opposed to chi's generic stdlib-style middleware.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/joythejaks/palmyield/backend/internal/auth"
)

type contextKey string

const claimsContextKey contextKey = "claims"

func Authenticate(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || token == "" {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			claims, err := auth.ParseAccessToken(secret, token)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*auth.Claims)
	return claims, ok
}

// RequireRole must run after Authenticate.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if _, permitted := allowed[claims.Role]; !permitted {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
