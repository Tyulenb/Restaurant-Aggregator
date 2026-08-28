package domain

import "time"

type Product struct {
	ID              string
	ExternalID      string
	EstablishmentID string
	CategoryID      string
	Name            string
	Description     *string
	Available       bool
	InventoryTypeID int
	Quantity        *int
	Price           float32
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}
