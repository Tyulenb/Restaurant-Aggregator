package infrahttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheck_ReturnsOK(t *testing.T) {
	router := NewRouter()
	registerHealthcheck(router)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}
	if body := response.Body.String(); body != "{\"status\":\"ok\"}\n" {
		t.Errorf("body = %q, want %q", body, "{\"status\":\"ok\"}\n")
	}
}

func TestHealthcheck_RejectsUnsupportedMethod(t *testing.T) {
	router := NewRouter()
	registerHealthcheck(router)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/health", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
