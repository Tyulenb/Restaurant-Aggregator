package adapterhttp

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

func TestOptionalOrderStatus(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    *int
		wantErr bool
	}{
		{name: "omitted"},
		{name: "pending", query: "?status=1", want: intPointer(int(domain.OrderCreated))},
		{name: "completed", query: "?status=5", want: intPointer(int(domain.OrderCompleted))},
		{name: "not an integer", query: "?status=pending", wantErr: true},
		{name: "outside enum", query: "?status=8", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/api/partner/v1/orders"+tt.query, nil)
			status, err := optionalOrderStatus(request)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, status)
		})
	}
}

func intPointer(value int) *int {
	return &value
}
