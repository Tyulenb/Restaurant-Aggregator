package domain

import (
	"time"

	"github.com/google/uuid"
)

type Establishment struct {
	ID          string
	Name        string
	Description *string
	Available   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewEstablishment(name string, description *string) *Establishment {
	return &Establishment{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		Available:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
