package domain

import "time"

type Menu struct {
	Establishment Establishment
	Categories    []MenuCategory
}

type Establishment struct {
	ID          string
	Name        string
	Description *string
	Available   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type MenuCategory struct {
	ID          string
	ExternalID  string
	Name        string
	Description *string
	Products    []Product
}
