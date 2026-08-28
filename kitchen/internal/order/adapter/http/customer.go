package adapterhttp

import (
	"context"
	"log/slog"
	"net/http"

	infrahttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/http"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type CustomerOrderUsecase interface {
	CreateOrder(ctx context.Context, establishmentId, idempotencyKey string, orderItem []domain.CreateOrderItem) (*domain.Order, error)
	FindOrder(ctx context.Context, orderId string) (*domain.Order, error)
	CancelOrder(ctx context.Context, orderId string) (*domain.Order, error)
}

// Handle routes belong to customer.
type CustomerHandler struct {
	logger  *slog.Logger
	usecase CustomerOrderUsecase
}

func NewOrderHandler(logger *slog.Logger) *CustomerHandler {
	return &CustomerHandler{
		logger: logger,
	}
}

func (h *CustomerHandler) RegisterRoutes(router *http.ServeMux) {
	router.HandleFunc("POST /api/v1/orders", h.createOrder)
	router.HandleFunc("GET /api/v1/orders/{orderId}", h.getOrder)
	router.HandleFunc("GET /api/v1/orders/{orderId}/cancel", h.cancelOrder)
}

func (h *CustomerHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	orderReq := new(CreateOrderRequest)
	err := infrahttp.ParseJSON(r, orderReq)
	if err != nil {
		h.logger.Error("Create order parsing data", "error", err)
		infrahttp.WriteError(w, http.StatusBadRequest, "BAD REQUEST", "Error validating request body.")
		return
	}

	idempotencyKey, ok := r.Context().Value("idempotencyKey").(string)
	if !ok {
		h.logger.Error("Create order missing idempotencyKey", "error", err)
		infrahttp.WriteError(w, http.StatusBadRequest, "BAD REQUEST", "Error validating request body.")
		return
	}
	domainOrderItems := make([]domain.CreateOrderItem, 0, len(orderReq.Items))
	for _, item := range orderReq.Items {
		domainOrderItems = append(domainOrderItems, toDomainCreateOrderItem(item))
	}
	domainOrder, err := h.usecase.CreateOrder(r.Context(), orderReq.EstablishmentID, idempotencyKey, domainOrderItems)
	if err != nil {
		h.logger.Error("Create order parsing data", "error", err)
		infrahttp.WriteError(w, http.StatusInternalServerError, "BAD REQUEST", "Error validating request body.")
		return
	}
	dtoOrder := domainOrderToDTO(*domainOrder)
	infrahttp.WriteJSON(w, http.StatusCreated, dtoOrder)
}

func (h *CustomerHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	orderId := r.PathValue("orderId")
	domainOrder, err := h.usecase.FindOrder(r.Context(), orderId)
	if err != nil {
		h.logger.Error("Get order", "error", err)
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
		return
	}
	dtoOrder := domainOrderToDTO(*domainOrder)
	infrahttp.WriteJSON(w, http.StatusOK, dtoOrder)
}

func (h *CustomerHandler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	orderId := r.PathValue("orderId")
	domainOrder, err := h.usecase.CancelOrder(r.Context(), orderId)
	if err != nil {
		h.logger.Error("Cancel order", "error", err)
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
		return
	}
	dtoOrder := domainOrderToDTO(*domainOrder)
	infrahttp.WriteJSON(w, http.StatusOK, dtoOrder)
}

func toDomainCreateOrderItem(item CreateOrderItem) domain.CreateOrderItem {
	return domain.CreateOrderItem{
		ProductID: item.ProductID,
		Quantity:  item.Quantity,
	}
}

func domainOrderToDTO(order domain.Order) Order {
	items := make([]OrderItem, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, domainOrderItemToDTO(item))
	}
	return Order{
		ID:              order.ID,
		EstablishmentID: order.EstablishmentID,
		CustomerID:      order.CustomerID,
		DeliveryID:      order.DeliveryID,
		Status:          order.Status,
		TotalPrice:      order.TotalPrice,
		RejectionReason: order.RejectionReason,
		Items:           items,
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}
}

func domainOrderItemToDTO(item domain.OrderItem) OrderItem {
	return OrderItem{
		ID:                item.ID,
		ProductID:         item.ProductID,
		ExternalProductID: item.ExternalProductID,
		ProductName:       item.ProductName,
		UnitPrice:         item.UnitPrice,
		Quantity:          item.Quantity,
		TotalSum:          item.TotalSum,
	}
}
