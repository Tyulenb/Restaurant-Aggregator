package adapterhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	infrahttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/http"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application/customer"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type customerOrderUsecaseStub struct {
	findOrder   *domain.Order
	cancelOrder *domain.Order
	orderId     string
}

func (s *customerOrderUsecaseStub) CreateOrder(context.Context, string, string, []customer.CreateOrderItem) (*domain.Order, error) {
	return nil, nil
}

func (s *customerOrderUsecaseStub) FindOrder(_ context.Context, orderId string) (*domain.Order, error) {
	s.orderId = orderId
	return s.findOrder, nil
}

func (s *customerOrderUsecaseStub) CancelOrder(_ context.Context, orderId string) (*domain.Order, error) {
	s.orderId = orderId
	return s.cancelOrder, nil
}

func TestCustomerHandler_GetOrder(t *testing.T) {
	t.Parallel()

	usecase := &customerOrderUsecaseStub{findOrder: &domain.Order{ID: "order-id", StatusID: 1}}
	handler := &CustomerHandler{logger: slog.Default(), usecase: usecase}
	router := infrahttp.NewRouter()
	handler.RegisterRoutes(router)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/orders/order-id", nil))

	if response.Code != http.StatusOK {
		t.Errorf("StatusID = %d, want %d", response.Code, http.StatusOK)
	}
	if usecase.orderId != "order-id" {
		t.Errorf("order ID = %q, want order-id", usecase.orderId)
	}
}

func TestCustomerHandler_CancelOrder(t *testing.T) {
	t.Parallel()

	usecase := &customerOrderUsecaseStub{cancelOrder: &domain.Order{ID: "order-id", StatusID: 7}}
	handler := &CustomerHandler{logger: slog.Default(), usecase: usecase}
	router := infrahttp.NewRouter()
	handler.RegisterRoutes(router)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/orders/order-id/cancel", nil))

	if response.Code != http.StatusOK {
		t.Errorf("StatusID = %d, want %d", response.Code, http.StatusOK)
	}
	if usecase.orderId != "order-id" {
		t.Errorf("order ID = %q, want order-id", usecase.orderId)
	}
}

func TestCustomerHandler_WriteUseCaseError_OrderCreationBusinessErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "establishment not found",
			err:        domain.ErrEstablishmentNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "NOT_FOUND",
		},
		{
			name:       "product not found through wrapped use case error",
			err:        fmt.Errorf("create order: %w", domain.ErrProductNotFound),
			wantStatus: http.StatusNotFound,
			wantCode:   "NOT_FOUND",
		},
		{
			name:       "establishment unavailable",
			err:        domain.ErrEstablishmentUnavailable,
			wantStatus: http.StatusConflict,
			wantCode:   "ESTABLISHMENT_UNAVAILABLE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := &CustomerHandler{
				logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			}
			response := httptest.NewRecorder()

			handler.writeUseCaseError(response, "create order", tt.err)

			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			var payload struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if payload.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", payload.Code, tt.wantCode)
			}
		})
	}
}
