package adapterhttp

import (
	"log/slog"
	"net/http"
)

// Handle routes belong to customer.
type PartnerHandler struct {
	logger *slog.Logger
}

func NewPartnerHandler(logger *slog.Logger) *CustomerHandler {
	return &CustomerHandler{
		logger: logger,
	}
}

func (h *PartnerHandler) RegisterRoutes(router *http.ServeMux) {
	router.HandleFunc("GET /api/partner/v1/orders", h.pollOrders)
	router.HandleFunc("GET /api/partner/v1/orders/{orderId}", h.getOrder)
	router.HandleFunc("POST /api/partner/v1/orders/{orderId}/accept", h.acceptOrder)
	router.HandleFunc("POST /api/partner/v1/orders/{orderId}/reject", h.rejectOrder)
	router.HandleFunc("POST /api/partner/v1/orders/{orderId}/status", h.changeOrderStatus)
}

func (h *PartnerHandler) pollOrders(w http.ResponseWriter, r *http.Request) {}

func (h *PartnerHandler) getOrder(w http.ResponseWriter, r *http.Request) {}

func (h *PartnerHandler) acceptOrder(w http.ResponseWriter, r *http.Request) {}

func (h *PartnerHandler) rejectOrder(w http.ResponseWriter, r *http.Request) {}

func (h *PartnerHandler) changeOrderStatus(w http.ResponseWriter, r *http.Request) {}
