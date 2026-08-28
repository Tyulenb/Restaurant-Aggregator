package adapterhttp

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type customerOrderUsecaseStub struct {
	findOrder   *domain.Order
	cancelOrder *domain.Order
	orderId     string
}

func (s *customerOrderUsecaseStub) CreateOrder(context.Context, string, string, []domain.CreateOrderItem) (*domain.Order, error) {
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
	router := http.NewServeMux()
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
	router := http.NewServeMux()
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
