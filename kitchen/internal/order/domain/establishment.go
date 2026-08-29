package domain

import "time"

type Establishment struct {
	ID          string
	Name        string
	Description *string
	Available   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
