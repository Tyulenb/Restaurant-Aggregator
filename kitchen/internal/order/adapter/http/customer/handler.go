package adapterhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	infrahttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/http"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application/customer"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type CustomerOrderUsecase interface {
	CreateOrder(ctx context.Context, establishmentId, idempotencyKey string, orderItem []customer.CreateOrderItem) (*domain.Order, error)
	FindOrder(ctx context.Context, orderId string) (*domain.Order, error)
	CancelOrder(ctx context.Context, orderId string) (*domain.Order, error)
}

// CustomerHandler handles customer order routes.
type CustomerHandler struct {
	logger  *slog.Logger
	usecase CustomerOrderUsecase
}

func NewOrderHandler(logger *slog.Logger, usecase CustomerOrderUsecase) *CustomerHandler {
	return &CustomerHandler{logger: logger, usecase: usecase}
}

func (h *CustomerHandler) RegisterRoutes(router *infrahttp.Router) {
	router.Handle("POST /api/v1/orders", h.createOrder, infrahttp.IdempotencyKeyMiddleware)
	router.Handle("GET /api/v1/orders/{orderId}", h.getOrder)
	router.Handle("POST /api/v1/orders/{orderId}/cancel", h.cancelOrder)
}

func (h *CustomerHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	request := new(CreateOrderRequest)
	if err := infrahttp.ParseJSON(r, request); err != nil {
		writeBadRequest(w)
		return
	}

	idempotencyKey, ok := r.Context().Value(infrahttp.IdempotencyKeyContextKey).(string)
	if !ok {
		writeBadRequest(w)
		return
	}
	if _, err := uuid.Parse(idempotencyKey); err != nil {
		writeBadRequest(w)
		return
	}
	if !validCreateOrderRequest(request) {
		writeValidationError(w)
		return
	}

	items := make([]customer.CreateOrderItem, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, toDomainCreateOrderItem(item))
	}
	order, err := h.usecase.CreateOrder(r.Context(), request.EstablishmentID, idempotencyKey, items)
	if err != nil {
		h.writeUseCaseError(w, "create order", err)
		return
	}
	if order == nil {
		h.writeUseCaseError(w, "create order", errors.New("use case returned a nil order"))
		return
	}

	w.Header().Set("Location", "/api/v1/orders/"+order.ID)
	infrahttp.WriteJSON(w, http.StatusCreated, domainOrderToDTO(*order))
}

func (h *CustomerHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	order, err := h.usecase.FindOrder(r.Context(), r.PathValue("orderId"))
	if err != nil {
		h.writeUseCaseError(w, "get order", err)
		return
	}
	if order == nil {
		writeNotFound(w)
		return
	}
	infrahttp.WriteJSON(w, http.StatusOK, domainOrderToDTO(*order))
}

func (h *CustomerHandler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	order, err := h.usecase.CancelOrder(r.Context(), r.PathValue("orderId"))
	if err != nil {
		h.writeUseCaseError(w, "cancel order", err)
		return
	}
	if order == nil {
		writeNotFound(w)
		return
	}
	infrahttp.WriteJSON(w, http.StatusOK, domainOrderToDTO(*order))
}

func (h *CustomerHandler) writeUseCaseError(w http.ResponseWriter, operation string, err error) {
	h.logger.Error(operation, "error", err)
	switch {
	case errors.Is(err, domain.ErrOrderNotFound),
		errors.Is(err, domain.ErrEstablishmentNotFound),
		errors.Is(err, domain.ErrProductNotFound),
		errors.Is(err, domain.ErrProductDeleted),
		errors.Is(err, domain.ErrProductFromOtherEstablishment):
		writeNotFound(w)
	case errors.Is(err, domain.ErrEmptyOrder), errors.Is(err, domain.ErrInvalidItemQuantity):
		writeValidationError(w)
	case errors.Is(err, domain.ErrProductUnavailable):
		infrahttp.WriteError(w, http.StatusConflict, "PRODUCT_UNAVAILABLE", "Product is unavailable.")
	case errors.Is(err, domain.ErrEstablishmentUnavailable):
		infrahttp.WriteError(w, http.StatusConflict, "ESTABLISHMENT_UNAVAILABLE", "Establishment is not accepting orders.")
	case errors.Is(err, domain.ErrInsufficientStock):
		infrahttp.WriteError(w, http.StatusConflict, "INSUFFICIENT_STOCK", "Requested quantity is not available.")
	case errors.Is(err, domain.ErrIdempotencyConflict):
		infrahttp.WriteError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Idempotency key was already used for another request.")
	case errors.Is(err, domain.ErrOrderCannotBeCancelled):
		writeInvalidTransition(w)
	default:
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
	}
}

func validCreateOrderRequest(request *CreateOrderRequest) bool {
	if _, err := uuid.Parse(request.EstablishmentID); err != nil || len(request.Items) == 0 || len(request.Items) > 100 {
		return false
	}
	for _, item := range request.Items {
		if _, err := uuid.Parse(item.ProductID); err != nil || item.Quantity < 1 {
			return false
		}
	}
	return true
}

func writeBadRequest(w http.ResponseWriter) {
	infrahttp.WriteBadRequest(w)
}

func writeValidationError(w http.ResponseWriter) {
	infrahttp.WriteValidationError(w)
}

func writeNotFound(w http.ResponseWriter) {
	infrahttp.WriteNotFound(w)
}

func writeInvalidTransition(w http.ResponseWriter) {
	infrahttp.WriteError(w, http.StatusConflict, "INVALID_ORDER_STATE_TRANSITION", "Requested order state transition is not allowed.")
}

func toDomainCreateOrderItem(item CreateOrderItem) customer.CreateOrderItem {
	return customer.CreateOrderItem{ProductID: item.ProductID, Quantity: item.Quantity}
}

func domainOrderToDTO(order domain.Order) Order {
	items := make([]OrderItem, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, domainOrderItemToDTO(item))
	}
	return Order{
		ID: order.ID, EstablishmentID: order.EstablishmentID, CustomerID: order.CustomerID,
		DeliveryID: order.DeliveryID, StatusID: int(order.StatusID), TotalPrice: order.TotalPrice,
		RejectionReason: order.RejectionReason, Items: items, CreatedAt: order.CreatedAt, UpdatedAt: order.UpdatedAt,
	}
}

func domainOrderItemToDTO(item domain.OrderItem) OrderItem {
	return OrderItem{
		ID: item.ID, ProductID: item.ProductID, ExternalProductID: item.ExternalProductID,
		ProductName: item.ProductName, UnitPrice: item.UnitPrice, Quantity: item.Quantity, TotalSum: item.TotalSum,
	}
}
