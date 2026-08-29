package adapterhttp

import "time"

type Establishment struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Available   bool    `json:"available"`
}

type PartnerEstablishment struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Available   bool      `json:"available"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ChangeAvailabilityRequest struct {
	Available *bool `json:"available"`
}

type EstablishmentListResponse struct {
	Items []Establishment `json:"items"`
}
