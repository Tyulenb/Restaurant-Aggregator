package adapterhttp

import (
	"encoding/json"
	"time"
)

// CreateProductRequest is the payload for creating a product.
type CreateProductRequest struct {
	ExternalID    string  `json:"external_id"`
	CategoryID    string  `json:"category_id"`
	Name          string  `json:"name"`
	Description   *string `json:"description"`
	Price         float32 `json:"price"`
	Available     bool    `json:"available"`
	InventoryType string  `json:"inventory_type"`
	Quantity      *int    `json:"quantity"`
}

// UpdateProductRequest contains product metadata that may be changed.
type UpdateProductRequest struct {
	CategoryID     *string  `json:"category_id"`
	Name           *string  `json:"name"`
	Description    *string  `json:"description"`
	DescriptionSet bool     `json:"-"`
	Price          *float32 `json:"price"`
	Available      *bool    `json:"available"`
}

func (r *UpdateProductRequest) UnmarshalJSON(data []byte) error {
	type request UpdateProductRequest
	var decoded request
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*r = UpdateProductRequest(decoded)
	_, r.DescriptionSet = fields["description"]

	return nil
}

// PartnerProduct is the partner representation of a product.
type PartnerProduct struct {
	ID            string     `json:"id"`
	ExternalID    string     `json:"external_id"`
	CategoryID    string     `json:"category_id"`
	Name          string     `json:"name"`
	Description   *string    `json:"description"`
	Price         float32    `json:"price"`
	Available     bool       `json:"available"`
	InventoryType string     `json:"inventory_type"`
	Quantity      *int       `json:"quantity"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at"`
}

// PartnerProductListResponse is the response returned when listing products.
type PartnerProductListResponse struct {
	Items []PartnerProduct `json:"items"`
}

// ProductListQuery contains filters for listing partner products.
type ProductListQuery struct {
	CategoryExternalID string `json:"category_external_id"`
	IncludeDeleted     bool   `json:"include_deleted"`
}

// StockAdjustmentRequest is the payload for changing finite product stock.
type StockAdjustmentRequest struct {
	Delta int `json:"delta"`
}
