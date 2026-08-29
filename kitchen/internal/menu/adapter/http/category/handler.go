package adapterhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	infrahttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/http"
	application "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/category"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

type CategoryUseCase interface {
	GetCategories(ctx context.Context, establishmentID string, includeDeleted bool) ([]domain.Category, error)
	CreateCategory(ctx context.Context, establishmentID, externalID, name string, description *string) (*domain.Category, error)

	GetCategoryByExternalID(ctx context.Context, establishmentID, externalID string) (*domain.Category, error)
	UpdateCategoryByExternalID(
		ctx context.Context, establishmentID, externalID string, cmd application.CommandUpdateCategory,
	) (*domain.Category, error)
	DeleteCategoryByExternalID(ctx context.Context, establishmentID, externalID string) (*domain.Category, error)
}

type CategoryHandler struct {
	logger         *slog.Logger
	usecase        CategoryUseCase
	authMiddleware infrahttp.Middleware
}

func NewCategoryHandler(logger *slog.Logger, usecase CategoryUseCase, authMiddleware infrahttp.Middleware) *CategoryHandler {
	return &CategoryHandler{logger: logger, usecase: usecase, authMiddleware: authMiddleware}
}

func (h *CategoryHandler) RegisterRoutes(router *infrahttp.Router) {
	router.Handle("GET /api/partner/v1/categories", h.getEstablishmentCategories, h.authMiddleware)
	router.Handle("POST /api/partner/v1/categories", h.createCategory, h.authMiddleware)
	router.Handle("GET /api/partner/v1/categories/{externalCategoryId}", h.getCategory, h.authMiddleware)
	router.Handle("PATCH /api/partner/v1/categories/{externalCategoryId}", h.patchCategory, h.authMiddleware)
	router.Handle("DELETE /api/partner/v1/categories/{externalCategoryId}", h.deleteCategory, h.authMiddleware)
}

func (h *CategoryHandler) getEstablishmentCategories(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := infrahttp.EstablishmentID(w, r)
	if !ok {
		return
	}
	includeDeleted, err := infrahttp.OptionalBoolQuery(r, "include_deleted")
	if err != nil {
		writeCategoryBadRequest(w)
		return
	}
	categories, err := h.usecase.GetCategories(r.Context(), establishmentID, includeDeleted)
	if err != nil {
		h.writeUseCaseError(w, "list categories", err)
		return
	}

	items := make([]PartnerCategory, 0, len(categories))
	for _, category := range categories {
		items = append(items, categoryToDTO(category))
	}
	infrahttp.WriteJSON(w, http.StatusOK, PartnerCategoryListResponse{Items: items})
}

func (h *CategoryHandler) createCategory(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := infrahttp.EstablishmentID(w, r)
	if !ok {
		return
	}
	request := new(CreateCategoryRequest)
	if err := infrahttp.ParseJSON(r, request); err != nil {
		writeCategoryBadRequest(w)
		return
	}
	category, err := h.usecase.CreateCategory(
		r.Context(), establishmentID, request.ExternalID, request.Name, request.Description,
	)
	if err != nil {
		h.writeUseCaseError(w, "create category", err)
		return
	}
	if category == nil {
		h.writeUseCaseError(w, "create category", errors.New("use case returned a nil category"))
		return
	}
	infrahttp.WriteJSON(w, http.StatusCreated, categoryToDTO(*category))
}

func (h *CategoryHandler) getCategory(w http.ResponseWriter, r *http.Request) {
	establishmentID, externalID, ok := categoryRequestIDs(w, r)
	if !ok {
		return
	}
	category, err := h.usecase.GetCategoryByExternalID(r.Context(), establishmentID, externalID)
	if err != nil {
		h.writeUseCaseError(w, "get category", err)
		return
	}
	writeCategory(w, category)
}

func (h *CategoryHandler) patchCategory(w http.ResponseWriter, r *http.Request) {
	establishmentID, externalID, ok := categoryRequestIDs(w, r)
	if !ok {
		return
	}
	request := new(UpdateCategoryRequest)
	if err := infrahttp.ParseJSON(r, request); err != nil {
		writeCategoryBadRequest(w)
		return
	}
	category, err := h.usecase.UpdateCategoryByExternalID(
		r.Context(), establishmentID, externalID, application.CommandUpdateCategory{
			Name:           request.Name,
			Description:    request.Description,
			DescriptionSet: request.DescriptionSet,
		},
	)
	if err != nil {
		h.writeUseCaseError(w, "update category", err)
		return
	}
	writeCategory(w, category)
}

func (h *CategoryHandler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	establishmentID, externalID, ok := categoryRequestIDs(w, r)
	if !ok {
		return
	}
	category, err := h.usecase.DeleteCategoryByExternalID(r.Context(), establishmentID, externalID)
	if err != nil {
		h.writeUseCaseError(w, "delete category", err)
		return
	}
	if category == nil {
		writeCategoryNotFound(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func categoryRequestIDs(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	establishmentID, ok := infrahttp.EstablishmentID(w, r)
	if !ok {
		return "", "", false
	}
	externalID := r.PathValue("externalCategoryId")
	return establishmentID, externalID, true
}

func (h *CategoryHandler) writeUseCaseError(w http.ResponseWriter, operation string, err error) {
	h.logger.Error(operation, "error", err)
	switch {
	case errors.Is(err, domain.ErrCategoryNotFound):
		writeCategoryNotFound(w)
	case errors.Is(err, domain.ErrCategoryConflict):
		infrahttp.WriteError(w, http.StatusConflict, "CATEGORY_ALREADY_EXISTS", "A category with this external ID already exists.")
	case errors.Is(err, domain.ErrCategoryValidation):
		writeCategoryValidationError(w)
	default:
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
	}
}

func writeCategory(w http.ResponseWriter, category *domain.Category) {
	if category == nil {
		writeCategoryNotFound(w)
		return
	}
	infrahttp.WriteJSON(w, http.StatusOK, categoryToDTO(*category))
}

func categoryToDTO(category domain.Category) PartnerCategory {
	return PartnerCategory{
		ID: category.ID, ExternalID: category.ExternalID, Name: category.Name,
		Description: category.Description, CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt, DeletedAt: category.DeletedAt,
	}
}

func writeCategoryBadRequest(w http.ResponseWriter) {
	infrahttp.WriteBadRequest(w)
}

func writeCategoryValidationError(w http.ResponseWriter) {
	infrahttp.WriteValidationError(w)
}

func writeCategoryNotFound(w http.ResponseWriter) {
	infrahttp.WriteNotFound(w)
}
