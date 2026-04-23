package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const guideIDKey contextKey = "guide_id"

// GuideAuth returns middleware that validates a Bearer JWT and injects the
// guide's ID into the request context. Responds 401 if the token is missing
// or invalid.
func GuideAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			guideID, _ := claims["sub"].(string)
			ctx := context.WithValue(r.Context(), guideIDKey, guideID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GuideIDFromContext returns the guide ID injected by GuideAuth middleware.
func GuideIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(guideIDKey).(string)
	return id
}
