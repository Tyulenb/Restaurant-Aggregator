package adapterhttp

import (
	"time"
)

// CreateCategoryRequest is the payload for creating a category.
type CreateCategoryRequest struct {
	ExternalID  string  `json:"external_id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// UpdateCategoryRequest contains the category fields that may be changed.
type UpdateCategoryRequest struct {
	Name           *string `json:"name"`
	Description    *string `json:"description"`
	DescriptionSet bool    `json:"-"`
}

// PartnerCategory is the partner representation of a category.
type PartnerCategory struct {
	ID          string     `json:"id"`
	ExternalID  string     `json:"external_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

// PartnerCategoryListResponse is the response returned when listing categories.
type PartnerCategoryListResponse struct {
	Items []PartnerCategory `json:"items"`
}

// CategoryListQuery contains filters for listing partner categories.
type CategoryListQuery struct {
	IncludeDeleted bool `json:"include_deleted"`
}
