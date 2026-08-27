package application

import (
	"context"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/domain"
)

type EstablishmentRepository interface {
	SelectEstablishments(ctx context.Context) ([]domain.Establishment, error)
	UpdateEstablishmentAvailability(ctx context.Context, establishmentId string, aval bool) (*domain.Establishment, error)
}
