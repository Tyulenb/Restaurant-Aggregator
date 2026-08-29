package adapterhttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/domain"
	infrahttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/http"
)

type establishmentServiceStub struct {
	available *bool
}

func (s *establishmentServiceStub) FindEstablishments(_ context.Context, available *bool) ([]domain.Establishment, error) {
	s.available = available
	return []domain.Establishment{{ID: "id", Name: "Kitchen", Available: true}}, nil
}

func (*establishmentServiceStub) ChangeAvailability(context.Context, string, bool) (*domain.Establishment, error) {
	return nil, nil
}

func TestEstablishmentHandler_GetEstablishments_FiltersAndWrapsItems(t *testing.T) {
	service := new(establishmentServiceStub)
	handler := NewCustomerHandler(slog.Default(), service, func(next http.Handler) http.Handler { return next })
	router := infrahttp.NewRouter()
	handler.RegisterRoutes(router)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/establishments?available=false", nil))

	require.Equal(t, http.StatusOK, response.Code)
	require.NotNil(t, service.available)
	assert.False(t, *service.available)
	var payload EstablishmentListResponse
	require.NoError(t, json.NewDecoder(response.Body).Decode(&payload))
	require.Len(t, payload.Items, 1)
	assert.Equal(t, "id", payload.Items[0].ID)
}
