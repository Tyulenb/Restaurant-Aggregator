package infrahttp

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
)

const (
	EstablishmentIDContextKey = "token"
	IdempotencyKeyContextKey  = "idempotencyKey"
	idempotencyKeyHeader      = "Idempotency-Key"
)

func EstablishmentAuthMiddleware(expectedHash, establishmentID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			incomingHash, hasBearerPrefix := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !hasBearerPrefix || subtle.ConstantTimeCompare([]byte(incomingHash), []byte(expectedHash)) != 1 {
				WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid partner API key.")
				return
			}

			ctx := context.WithValue(r.Context(), EstablishmentIDContextKey, establishmentID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func IdempotencyKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), IdempotencyKeyContextKey, r.Header.Get(idempotencyKeyHeader))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
