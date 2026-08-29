package domain

import "time"

type InventoryType int

const (
	Finite    InventoryType = 1
	Unlimited InventoryType = 2
)

type Product struct {
	ID              string
	ExternalID      string
	EstablishmentID string
	CategoryID      string
	Name            string
	Description     *string
	Available       bool
	InventoryTypeID InventoryType
	Quantity        *int
	Price           float32
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}
