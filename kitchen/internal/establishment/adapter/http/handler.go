package adapterhttp

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/domain"
	infrahttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/http"
)

type Service interface {
	FindEstablishments(ctx context.Context) ([]domain.Establishment, error)
	ChangeAvailability(ctx context.Context, establishmentId string, availability bool) (*domain.Establishment, error)
}

type EstablishmentHandler struct {
	logger  *slog.Logger
	service Service
}

func NewCustomerHandler(logger *slog.Logger, service Service) *EstablishmentHandler {
	return &EstablishmentHandler{
		logger:  logger,
		service: service,
	}
}

func (c *EstablishmentHandler) RegisterRoutes(router *http.ServeMux) {
	router.HandleFunc("GET /api/v1/establishments", c.getEstablishments)
	router.HandleFunc("PATCH /api/partner/v1/establishment/availability", c.changeAvailability)
}

func (c *EstablishmentHandler) getEstablishments(w http.ResponseWriter, r *http.Request) {
	domainEstablishments, err := c.service.FindEstablishments(r.Context())
	if err != nil {
		c.logger.Error("list establishments", "error", err)
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
		return
	}

	dtoEstablishments := make([]Establishment, 0, len(domainEstablishments))
	for _, est := range domainEstablishments {
		dtoEstablishments = append(dtoEstablishments, domainToDTO(est))
	}
	infrahttp.WriteJSON(w, http.StatusOK, dtoEstablishments)
}

func (c *EstablishmentHandler) changeAvailability(w http.ResponseWriter, r *http.Request) {
	availReq := new(ChangeAvailabilityRequest)
	err := infrahttp.ParseJSON(r, availReq)
	if err != nil {
		c.logger.Error("change availability", "error", err)
		infrahttp.WriteError(w, http.StatusUnprocessableEntity, "Unprocessable Entity", "Unprocessable input data.")
		return
	}
	estableId, ok := r.Context().Value("token").(string)
	if !ok {
		c.logger.Error("change availability", "error", "type assertion error")
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
		return
	}
	establishment, err := c.service.ChangeAvailability(r.Context(), estableId, availReq.Available)
	if err != nil {
		c.logger.Error("change availability", "error", err)
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
		return
	}
	infrahttp.WriteJSON(w, http.StatusOK, domainToDTO(*establishment))
}

func domainToDTO(e domain.Establishment) Establishment {
	return Establishment{
		ID:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		Available:   e.Available,
	}
}
