package adapterhttp

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/domain"
	infrahttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/http"
)

type Service interface {
	FindEstablishments(ctx context.Context, available *bool) ([]domain.Establishment, error)
	ChangeAvailability(ctx context.Context, establishmentId string, availability bool) (*domain.Establishment, error)
}

type PartnerEstablishmentService interface {
	FindPartnerEstablishment(ctx context.Context, establishmentID string) (*domain.Establishment, error)
}

type EstablishmentHandler struct {
	logger         *slog.Logger
	service        Service
	authMiddleware infrahttp.Middleware
}

func NewCustomerHandler(logger *slog.Logger, service Service, authMiddleware infrahttp.Middleware) *EstablishmentHandler {
	return &EstablishmentHandler{
		logger:         logger,
		service:        service,
		authMiddleware: authMiddleware,
	}
}

func (c *EstablishmentHandler) RegisterRoutes(router *infrahttp.Router) {
	router.Handle("GET /api/v1/establishments", c.getEstablishments)
	router.Handle("PATCH /api/partner/v1/establishment/availability", c.changeAvailability, c.authMiddleware)
	router.Handle("GET /api/partner/v1/establishment", c.getPartnerEstablishment, c.authMiddleware)
}

func (c *EstablishmentHandler) getEstablishments(w http.ResponseWriter, r *http.Request) {
	available, err := infrahttp.OptionalBoolQueryValue(r, "available")
	if err != nil {
		infrahttp.WriteBadRequest(w)
		return
	}
	domainEstablishments, err := c.service.FindEstablishments(r.Context(), available)
	if err != nil {
		c.logger.Error("list establishments", "error", err)
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
		return
	}

	dtoEstablishments := make([]Establishment, 0, len(domainEstablishments))
	for _, est := range domainEstablishments {
		dtoEstablishments = append(dtoEstablishments, domainToDTO(est))
	}
	infrahttp.WriteJSON(w, http.StatusOK, EstablishmentListResponse{Items: dtoEstablishments})
}

func (c *EstablishmentHandler) changeAvailability(w http.ResponseWriter, r *http.Request) {
	availReq := new(ChangeAvailabilityRequest)
	err := infrahttp.ParseJSON(r, availReq)
	if err != nil {
		c.logger.Error("change availability", "error", err)
		infrahttp.WriteError(w, http.StatusUnprocessableEntity, "Unprocessable Entity", "Unprocessable input data.")
		return
	}
	if availReq.Available == nil {
		c.logger.Error("change availability", "error", "available is nil")
		infrahttp.WriteValidationError(w)
		return
	}
	estableId, ok := infrahttp.EstablishmentID(w, r)
	if !ok {
		return
	}
	establishment, err := c.service.ChangeAvailability(r.Context(), estableId, *availReq.Available)
	if err != nil {
		c.logger.Error("change availability", "error", err)
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
		return
	}
	if establishment == nil {
		c.logger.Error("change availability", "error", "service returned a nil establishment")
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
		return
	}
	infrahttp.WriteJSON(w, http.StatusOK, domainToPartnerDTO(*establishment))
}

func (c *EstablishmentHandler) getPartnerEstablishment(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := infrahttp.EstablishmentID(w, r)
	if !ok {
		return
	}
	service, ok := c.service.(PartnerEstablishmentService)
	if !ok {
		c.logger.Error("get partner establishment", "error", "service does not support partner establishment lookup")
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
		return
	}
	establishment, err := service.FindPartnerEstablishment(r.Context(), establishmentID)
	if err != nil {
		c.logger.Error("get partner establishment", "error", err)
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
		return
	}
	if establishment == nil {
		c.logger.Error("get partner establishment", "error", "service returned a nil establishment")
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
		return
	}

	infrahttp.WriteJSON(w, http.StatusOK, domainToPartnerDTO(*establishment))
}

func domainToDTO(e domain.Establishment) Establishment {
	return Establishment{
		ID:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		Available:   e.Available,
	}
}

func domainToPartnerDTO(e domain.Establishment) PartnerEstablishment {
	return PartnerEstablishment{
		ID:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		Available:   e.Available,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}
