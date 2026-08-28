package adapterhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	infrahttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/http"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type PartnerOrderUseCase interface {
	FindNewOrdersByEstablishmentId(ctx context.Context, establishmentId string) ([]domain.Order, error)
	FindOrderByOrderIdAndEstablishmentId(ctx context.Context, orderId, establishmentId string) (*domain.Order, error)
	ChangeOrderStatus(ctx context.Context, orderId, establishmentId string, statusID int) (*domain.Order, error)
}

// PartnerHandler handles routes belonging to a partner establishment.
type PartnerHandler struct {
	logger  *slog.Logger
	usecase PartnerOrderUseCase
}

func NewPartnerHandler(logger *slog.Logger, usecase PartnerOrderUseCase) *PartnerHandler {
	return &PartnerHandler{logger: logger, usecase: usecase}
}

func (h *PartnerHandler) RegisterRoutes(router *http.ServeMux) {
	router.HandleFunc("GET /api/partner/v1/orders", h.pollOrders)
	router.HandleFunc("GET /api/partner/v1/orders/{orderId}", h.getOrder)
	router.HandleFunc("POST /api/partner/v1/orders/{orderId}/accept", h.acceptOrder)
	router.HandleFunc("POST /api/partner/v1/orders/{orderId}/reject", h.rejectOrder)
	router.HandleFunc("PATCH /api/partner/v1/orders/{orderId}/status", h.changeOrderStatus)
}

func (h *PartnerHandler) pollOrders(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := establishmentIDFromContext(w, r)
	if !ok {
		return
	}
	orders, err := h.usecase.FindNewOrdersByEstablishmentId(r.Context(), establishmentID)
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

func (h *PartnerHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := establishmentIDFromContext(w, r)
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
	if r.Body != nil && r.ContentLength != 0 {
		request := new(RejectOrderRequest)
		if err := infrahttp.ParseJSON(r, request); err != nil {
			writeBadRequest(w)
			return
		}
		if len(request.Reason) > 255 {
			writeValidationError(w)
			return
		}
	}
	h.changeToStatus(w, r, domain.OrderRejected)
}

func (h *PartnerHandler) changeOrderStatus(w http.ResponseWriter, r *http.Request) {
	request := new(UpdateOrderStatusRequest)
	if err := infrahttp.ParseJSON(r, request); err != nil {
		writeBadRequest(w)
		return
	}
	if request.Status < int(domain.OrderPreparing) || request.Status > int(domain.OrderCompleted) {
		writeValidationError(w)
		return
	}
	h.changeToStatus(w, r, domain.OrderStatusID(request.Status))
}

func (h *PartnerHandler) changeToStatus(w http.ResponseWriter, r *http.Request, status domain.OrderStatusID) {
	establishmentID, ok := establishmentIDFromContext(w, r)
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

func establishmentIDFromContext(w http.ResponseWriter, r *http.Request) (string, bool) {
	establishmentID, ok := r.Context().Value(infrahttp.EstablishmentIDContextKey).(string)
	if !ok || establishmentID == "" {
		infrahttp.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid partner API key.")
		return "", false
	}
	return establishmentID, true
}

func (h *PartnerHandler) writeUseCaseError(w http.ResponseWriter, operation string, err error) {
	h.logger.Error(operation, "error", err)
	switch {
	case errors.Is(err, application.ErrOrderNotFound):
		writeNotFound(w)
	case errors.Is(err, application.ErrInvalidOrderTransition):
		writeInvalidTransition(w)
	default:
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
	}
}

func writeOrder(w http.ResponseWriter, order *domain.Order) {
	if order == nil {
		writeNotFound(w)
		return
	}
	infrahttp.WriteJSON(w, http.StatusOK, domainOrderToDTO(*order))
}
