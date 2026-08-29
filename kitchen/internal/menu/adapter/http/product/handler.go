package adapterhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	infrahttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/http"
	application "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/product"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
	orderdomain "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type ProductUseCase interface {
	ListPartnerProducts(ctx context.Context, establishmentID, categoryExternalID string, includeDeleted bool) ([]orderdomain.Product, error)
	CreatePartnerProduct(ctx context.Context, establishmentID string, cmd application.CommandCreateProduct) (*orderdomain.Product, error)
	FindProductByExternalId(ctx context.Context, establishmentID, externalID string) (*orderdomain.Product, error)
	UpdateProductByExternalId(ctx context.Context, establishmentID, externalID string, cmd application.CommandUpdateProduct) (*orderdomain.Product, error)
	DeleteProductByExternalId(ctx context.Context, establishmentID, externalID string) (*orderdomain.Product, error)
	AdjustProductStock(ctx context.Context, establishmentID, externalID string, delta int) (*orderdomain.Product, error)
}

type ProductHandler struct {
	logger  *slog.Logger
	usecase ProductUseCase
}

func NewProductHandler(logger *slog.Logger, usecase ProductUseCase) *ProductHandler {
	return &ProductHandler{logger: logger, usecase: usecase}
}

func (h *ProductHandler) RegisterRoutes(router *http.ServeMux) {
	router.HandleFunc("GET /api/partner/v1/products", h.getPartnerProducts)
	router.HandleFunc("POST /api/partner/v1/products", h.postPartnerProduct)
	router.HandleFunc("GET /api/partner/v1/products/{externalProductId}", h.getPartnerProduct)
	router.HandleFunc("PATCH /api/partner/v1/products/{externalProductId}", h.patchPartnerProduct)
	router.HandleFunc("DELETE /api/partner/v1/products/{externalProductId}", h.deletePartnerProduct)
	router.HandleFunc("POST /api/partner/v1/products/{externalProductId}/stock-adjustments", h.adjustStocks)
}

func (h *ProductHandler) getPartnerProducts(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := infrahttp.EstablishmentID(w, r)
	if !ok {
		return
	}
	includeDeleted, err := infrahttp.OptionalBoolQuery(r, "include_deleted")
	if err != nil {
		writeProductBadRequest(w)
		return
	}
	categoryExternalID := r.URL.Query().Get("category_external_id")
	products, err := h.usecase.ListPartnerProducts(r.Context(), establishmentID, categoryExternalID, includeDeleted)
	if err != nil {
		h.writeUseCaseError(w, "list products", err)
		return
	}
	items := make([]PartnerProduct, 0, len(products))
	for _, product := range products {
		items = append(items, productToDTO(product))
	}
	infrahttp.WriteJSON(w, http.StatusOK, PartnerProductListResponse{Items: items})
}

func (h *ProductHandler) postPartnerProduct(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := infrahttp.EstablishmentID(w, r)
	if !ok {
		return
	}
	request := new(CreateProductRequest)
	if err := infrahttp.ParseJSON(r, request); err != nil {
		writeProductBadRequest(w)
		return
	}
	product, err := h.usecase.CreatePartnerProduct(r.Context(), establishmentID, application.CommandCreateProduct{
		ExternalID: request.ExternalID, CategoryExternalID: request.CategoryID,
		Name: request.Name, Description: request.Description, Price: request.Price,
		Available: request.Available, InventoryType: request.InventoryType, Quantity: request.Quantity,
	})
	if err != nil {
		h.writeUseCaseError(w, "create product", err)
		return
	}
	if product == nil {
		h.writeUseCaseError(w, "create product", errors.New("use case returned a nil product"))
		return
	}
	infrahttp.WriteJSON(w, http.StatusCreated, productToDTO(*product))
}

func (h *ProductHandler) getPartnerProduct(w http.ResponseWriter, r *http.Request) {
	establishmentID, externalID, ok := productRequestIDs(w, r)
	if !ok {
		return
	}
	product, err := h.usecase.FindProductByExternalId(r.Context(), establishmentID, externalID)
	if err != nil {
		h.writeUseCaseError(w, "get product", err)
		return
	}
	writeProduct(w, product)
}

func (h *ProductHandler) patchPartnerProduct(w http.ResponseWriter, r *http.Request) {
	establishmentID, externalID, ok := productRequestIDs(w, r)
	if !ok {
		return
	}
	request := new(UpdateProductRequest)
	if err := infrahttp.ParseJSON(r, request); err != nil {
		writeProductBadRequest(w)
		return
	}
	product, err := h.usecase.UpdateProductByExternalId(r.Context(), establishmentID, externalID, application.CommandUpdateProduct{
		CategoryExternalID: request.CategoryID, Name: request.Name,
		Description: request.Description, DescriptionSet: request.DescriptionSet, Price: request.Price,
		Available: request.Available,
	})
	if err != nil {
		h.writeUseCaseError(w, "update product", err)
		return
	}
	writeProduct(w, product)
}

func (h *ProductHandler) deletePartnerProduct(w http.ResponseWriter, r *http.Request) {
	establishmentID, externalID, ok := productRequestIDs(w, r)
	if !ok {
		return
	}
	product, err := h.usecase.DeleteProductByExternalId(r.Context(), establishmentID, externalID)
	if err != nil {
		h.writeUseCaseError(w, "delete product", err)
		return
	}
	if product == nil {
		writeProductNotFound(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) adjustStocks(w http.ResponseWriter, r *http.Request) {
	establishmentID, externalID, ok := productRequestIDs(w, r)
	if !ok {
		return
	}
	request := new(StockAdjustmentRequest)
	if err := infrahttp.ParseJSON(r, request); err != nil {
		writeProductBadRequest(w)
		return
	}
	product, err := h.usecase.AdjustProductStock(
		r.Context(), establishmentID, externalID, request.NewQuantity,
	)
	if err != nil {
		h.writeUseCaseError(w, "adjust product stock", err)
		return
	}
	writeProduct(w, product)
}

func productRequestIDs(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	establishmentID, ok := infrahttp.EstablishmentID(w, r)
	if !ok {
		return "", "", false
	}
	externalID := r.PathValue("externalProductId")
	return establishmentID, externalID, true
}

func (h *ProductHandler) writeUseCaseError(w http.ResponseWriter, operation string, err error) {
	h.logger.Error(operation, "error", err)
	switch {
	case errors.Is(err, domain.ErrProductNotFound), errors.Is(err, domain.ErrCategoryNotFound):
		writeProductNotFound(w)
	case errors.Is(err, domain.ErrProductConflict):
		infrahttp.WriteError(w, http.StatusConflict, "PRODUCT_ALREADY_EXISTS", "A product with this external ID already exists.")
	case errors.Is(err, domain.ErrProductValidation):
		infrahttp.WriteValidationError(w)
	case errors.Is(err, domain.ErrStockAdjustmentConflict):
		infrahttp.WriteError(w, http.StatusConflict, "STOCK_ADJUSTMENT_CONFLICT", "Stock adjustment cannot be applied.")
	case errors.Is(err, domain.ErrProductIdempotencyConflict):
		infrahttp.WriteError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Idempotency key was already used for another request.")
	default:
		infrahttp.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
	}
}

func writeProduct(w http.ResponseWriter, product *orderdomain.Product) {
	if product == nil {
		writeProductNotFound(w)
		return
	}
	infrahttp.WriteJSON(w, http.StatusOK, productToDTO(*product))
}

func productToDTO(product orderdomain.Product) PartnerProduct {
	return PartnerProduct{
		ID: product.ID, ExternalID: product.ExternalID, CategoryID: product.CategoryID,
		Name: product.Name, Description: product.Description, Price: product.Price,
		Available: product.Available, InventoryType: inventoryTypeName(product.InventoryTypeID),
		Quantity: product.Quantity, CreatedAt: product.CreatedAt, UpdatedAt: product.UpdatedAt,
		DeletedAt: product.DeletedAt,
	}
}

func inventoryTypeName(inventoryType orderdomain.InventoryType) string {
	if inventoryType == orderdomain.Finite {
		return "finite"
	}
	return "untracked"
}

func writeProductBadRequest(w http.ResponseWriter) {
	infrahttp.WriteBadRequest(w)
}

func writeProductNotFound(w http.ResponseWriter) {
	infrahttp.WriteNotFound(w)
}
