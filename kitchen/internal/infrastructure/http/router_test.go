package infrahttp

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRouter_Handle_AppliesMiddlewareInDeclarationOrder(t *testing.T) {
	var calls []string
	middleware := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, name+" before")
				next.ServeHTTP(w, r)
				calls = append(calls, name+" after")
			})
		}
	}

	router := NewRouter()
	router.Handle("GET /test", func(http.ResponseWriter, *http.Request) {
		calls = append(calls, "handler")
	}, middleware("first"), middleware("second"))

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/test", nil))

	want := []string{"first before", "second before", "handler", "second after", "first after"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}
