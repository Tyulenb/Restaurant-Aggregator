package adapterhttp

import (
	"encoding/json"
	"testing"
)

func TestUpdateCategoryRequest_TracksNullableDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		payload   string
		wantSet   bool
		wantNil   bool
		wantValue string
	}{
		{name: "omitted", payload: `{"name":"Updated"}`, wantNil: true},
		{name: "null", payload: `{"description":null}`, wantSet: true, wantNil: true},
		{name: "value", payload: `{"description":"New"}`, wantSet: true, wantValue: "New"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var request UpdateCategoryRequest
			if err := json.Unmarshal([]byte(tt.payload), &request); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if request.DescriptionSet != tt.wantSet {
				t.Errorf("DescriptionSet = %t, want %t", request.DescriptionSet, tt.wantSet)
			}
			if tt.wantNil && request.Description != nil {
				t.Errorf("Description = %v, want nil", request.Description)
			}
			if !tt.wantNil && (request.Description == nil || *request.Description != tt.wantValue) {
				t.Errorf("Description = %v, want %q", request.Description, tt.wantValue)
			}
		})
	}
}
