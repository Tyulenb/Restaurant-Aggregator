package infrahttp

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestEstablishmentID_ValidContextValue(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	request = request.WithContext(context.WithValue(request.Context(), EstablishmentIDContextKey, "establishment-id"))
	recorder := httptest.NewRecorder()

	id, ok := EstablishmentID(recorder, request)

	if !ok || id != "establishment-id" {
		t.Fatalf("EstablishmentID() = %q, %v; want %q, true", id, ok, "establishment-id")
	}
}

func TestEstablishmentID_MissingContextValue(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()

	_, ok := EstablishmentID(recorder, request)

	if ok {
		t.Fatal("EstablishmentID() ok = true; want false")
	}
	if recorder.Code != 401 {
		t.Fatalf("status = %d; want 401", recorder.Code)
	}
}

func TestOptionalBoolQuery(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		want    bool
		wantErr bool
	}{
		{name: "missing", target: "/", want: false},
		{name: "true", target: "/?include_deleted=true", want: true},
		{name: "false", target: "/?include_deleted=false", want: false},
		{name: "invalid", target: "/?include_deleted=invalid", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", test.target, nil)
			got, err := OptionalBoolQuery(request, "include_deleted")
			if (err != nil) != test.wantErr {
				t.Fatalf("OptionalBoolQuery() error = %v; wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("OptionalBoolQuery() = %v; want %v", got, test.want)
			}
		})
	}
}
