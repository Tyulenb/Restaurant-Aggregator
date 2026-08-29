package domain

import "time"

type Category struct {
	ID              string
	EstablishmentID string
	ExternalID      string
	Name            string
	Description     *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}
