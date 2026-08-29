package establishment

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	infrahttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/http"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

type EstablishmentUseCase interface {
	GetMenu(ctx context.Context, establishmentId string) (domain.Menu, error)
}

type EstablishmentHandler struct {
	logger  *slog.Logger
	usecase EstablishmentUseCase
}

func NewEstablishmentHandler(logger *slog.Logger, usecase EstablishmentUseCase) *EstablishmentHandler {
	return &EstablishmentHandler{logger: logger, usecase: usecase}
}

func (h *EstablishmentHandler) RegisterRoutes(router *infrahttp.Router) {
	router.Handle("GET /api/v1/establishments/{establishmentId}/menu", h.getMenu)
}

func (h *EstablishmentHandler) getMenu(w http.ResponseWriter, r *http.Request) {
	menu, err := h.usecase.GetMenu(r.Context(), r.PathValue("establishmentId"))
	if errors.Is(err, domain.ErrEstablishmentNotFound) {
		infrahttp.WriteNotFound(w)
		return
	}
	if err != nil {
		h.logger.Error("get establishment menu", "error", err)
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
		return
	}
	infrahttp.WriteJSON(w, http.StatusOK, menuToDTO(menu))
}

func menuToDTO(menu domain.Menu) MenuResponse {
	categories := make([]MenuCategory, 0, len(menu.Categories))
	for _, category := range menu.Categories {
		products := make([]MenuProduct, 0, len(category.Products))
		for _, product := range category.Products {
			inventoryType := "untracked"
			if product.InventoryTypeID == domain.Finite {
				inventoryType = "finite"
			}
			products = append(products, MenuProduct{
				ID: product.ID, ExternalID: product.ExternalID, Name: product.Name,
				Description: product.Description, Price: product.Price, Available: product.Available,
				InventoryType: inventoryType, Quantity: product.Quantity,
			})
		}
		categories = append(categories, MenuCategory{
			ID: category.ID, ExternalID: category.ExternalID, Name: category.Name,
			Description: category.Description, Products: products,
		})
	}
	return MenuResponse{
		Establishment: Establishment{
			ID: menu.Establishment.ID, Name: menu.Establishment.Name,
			Description: menu.Establishment.Description, Available: menu.Establishment.Available,
		},
		Categories: categories,
	}
}
