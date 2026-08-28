package infrahttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEstablishmentAuthMiddleware_ValidBearerToken(t *testing.T) {
	t.Parallel()

	called := false
	handler := EstablishmentAuthMiddleware("partner-hash", "establishment-id")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := r.Context().Value(EstablishmentIDContextKey); got != "establishment-id" {
			t.Errorf("context establishment ID = %v, want establishment-id", got)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer partner-hash")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("next handler was not called")
	}
}

func TestEstablishmentAuthMiddleware_InvalidToken(t *testing.T) {
	t.Parallel()

	handler := EstablishmentAuthMiddleware("partner-hash", "establishment-id")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler must not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer incorrect-hash")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}

func TestIdempotencyKeyMiddleware_AddsHeaderToContext(t *testing.T) {
	t.Parallel()

	handler := IdempotencyKeyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Context().Value(IdempotencyKeyContextKey); got != "request-key" {
			t.Errorf("context idempotency key = %v, want request-key", got)
		}
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(idempotencyKeyHeader, "request-key")
	handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestIdempotencyKeyMiddleware_RejectsEmptyHeader(t *testing.T) {
	t.Parallel()

	handler := IdempotencyKeyMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler must not be called")
	}))

	for _, key := range []string{"", "   "} {
		t.Run("key="+key, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set(idempotencyKeyHeader, key)
			res := httptest.NewRecorder()

			handler.ServeHTTP(res, req)

			if res.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", res.Code, http.StatusBadRequest)
			}
		})
	}
}
