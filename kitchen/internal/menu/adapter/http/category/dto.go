package adapterhttp

import (
	"encoding/json"
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

func (r *UpdateCategoryRequest) UnmarshalJSON(data []byte) error {
	type request UpdateCategoryRequest
	var decoded request
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*r = UpdateCategoryRequest(decoded)
	_, r.DescriptionSet = fields["description"]

	return nil
}

// PartnerCategory is the partner representation of a category.
type PartnerCategory struct {
	ID          string     `json:"id"`
	ExternalID  string     `json:"external_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	CreatedAt   time.Time  `json:"crjeated_at"`
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
