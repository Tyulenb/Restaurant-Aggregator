package adapterhttp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"unicode/utf8"

	infrahttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/http"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type PartnerOrderUseCase interface {
	FindOrdersByEstablishmentID(ctx context.Context, establishmentId string, status *int) ([]domain.Order, error)
	FindOrderByOrderIdAndEstablishmentId(ctx context.Context, orderId, establishmentId string) (*domain.Order, error)
	ChangeOrderStatus(ctx context.Context, orderId, establishmentId string, statusID int) (*domain.Order, error)
	RejectOrder(ctx context.Context, orderId, establishmentId string, reason *string) (*domain.Order, error)
}

// PartnerHandler handles routes belonging to a partner establishment.
type PartnerHandler struct {
	logger         *slog.Logger
	usecase        PartnerOrderUseCase
	authMiddleware infrahttp.Middleware
}

func NewPartnerHandler(logger *slog.Logger, usecase PartnerOrderUseCase, authMiddleware infrahttp.Middleware) *PartnerHandler {
	return &PartnerHandler{logger: logger, usecase: usecase, authMiddleware: authMiddleware}
}

func (h *PartnerHandler) RegisterRoutes(router *infrahttp.Router) {
	router.Handle("GET /api/partner/v1/orders", h.getOrders, h.authMiddleware)
	router.Handle("GET /api/partner/v1/orders/{orderId}", h.getOrder, h.authMiddleware)
	router.Handle("POST /api/partner/v1/orders/{orderId}/accept", h.acceptOrder, h.authMiddleware)
	router.Handle("POST /api/partner/v1/orders/{orderId}/reject", h.rejectOrder, h.authMiddleware)
	router.Handle("PATCH /api/partner/v1/orders/{orderId}/status", h.changeOrderStatus, h.authMiddleware)
}

func (h *PartnerHandler) getOrders(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := infrahttp.EstablishmentID(w, r)
	if !ok {
		return
	}
	status, err := optionalOrderStatus(r)
	if err != nil {
		infrahttp.WriteBadRequest(w)
		return
	}
	orders, err := h.usecase.FindOrdersByEstablishmentID(r.Context(), establishmentID, status)
	if err != nil {
		h.writeUseCaseError(w, "poll orders", err)
		return
	}

	items := make([]Order, 0, len(orders))
	for _, order := range orders {
		items = append(items, domainOrderToDTO(order))
	}
	infrahttp.WriteJSON(w, http.StatusOK, PartnerOrderListResponse{Items: items})
}

func optionalOrderStatus(r *http.Request) (*int, error) {
	rawStatus := r.URL.Query().Get("status")
	if rawStatus == "" {
		return nil, nil
	}
	status, err := strconv.Atoi(rawStatus)
	if err != nil || status < int(domain.OrderCreated) || status > int(domain.OrderCanceled) {
		return nil, fmt.Errorf("invalid order status %q", rawStatus)
	}
	return &status, nil
}

func (h *PartnerHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := infrahttp.EstablishmentID(w, r)
	if !ok {
		return
	}
	order, err := h.usecase.FindOrderByOrderIdAndEstablishmentId(r.Context(), r.PathValue("orderId"), establishmentID)
	if err != nil {
		h.writeUseCaseError(w, "get partner order", err)
		return
	}
	writeOrder(w, order)
}

func (h *PartnerHandler) acceptOrder(w http.ResponseWriter, r *http.Request) {
	h.changeToStatus(w, r, domain.OrderAccepted)
}

func (h *PartnerHandler) rejectOrder(w http.ResponseWriter, r *http.Request) {
	var reason *string
	if r.Body != nil && r.ContentLength != 0 {
		request := new(RejectOrderRequest)
		if err := infrahttp.ParseJSON(r, request); err != nil {
			infrahttp.WriteBadRequest(w)
			return
		}
		if utf8.RuneCountInString(request.Reason) > 255 {
			infrahttp.WriteValidationError(w)
			return
		}
		reason = &request.Reason
	}
	establishmentID, ok := infrahttp.EstablishmentID(w, r)
	if !ok {
		return
	}
	order, err := h.usecase.RejectOrder(r.Context(), r.PathValue("orderId"), establishmentID, reason)
	if err != nil {
		h.writeUseCaseError(w, "reject order", err)
		return
	}
	writeOrder(w, order)
}

func (h *PartnerHandler) changeOrderStatus(w http.ResponseWriter, r *http.Request) {
	request := new(UpdateOrderStatusRequest)
	if err := infrahttp.ParseJSON(r, request); err != nil {
		infrahttp.WriteBadRequest(w)
		return
	}
	if request.Status < int(domain.OrderPreparing) || request.Status > int(domain.OrderCompleted) {
		infrahttp.WriteValidationError(w)
		return
	}
	h.changeToStatus(w, r, domain.OrderStatusID(request.Status))
}

func (h *PartnerHandler) changeToStatus(w http.ResponseWriter, r *http.Request, status domain.OrderStatusID) {
	establishmentID, ok := infrahttp.EstablishmentID(w, r)
	if !ok {
		return
	}
	order, err := h.usecase.ChangeOrderStatus(r.Context(), r.PathValue("orderId"), establishmentID, int(status))
	if err != nil {
		h.writeUseCaseError(w, "change order status", err)
		return
	}
	writeOrder(w, order)
}

func (h *PartnerHandler) writeUseCaseError(w http.ResponseWriter, operation string, err error) {
	h.logger.Error(operation, "error", err)
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		infrahttp.WriteNotFound(w)
	case errors.Is(err, domain.ErrInvalidOrderTransition):
		writeInvalidTransition(w)
	default:
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
	}
}

func writeOrder(w http.ResponseWriter, order *domain.Order) {
	if order == nil {
		infrahttp.WriteNotFound(w)
		return
	}
	infrahttp.WriteJSON(w, http.StatusOK, domainOrderToDTO(*order))
}

func writeInvalidTransition(w http.ResponseWriter) {
	infrahttp.WriteError(w, http.StatusConflict, "INVALID_ORDER_STATE_TRANSITION", "Requested order state transition is not allowed.")
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
