package application

import (
	"context"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/domain"
)

type Service struct {
	repository EstablishmentRepository
}

func NewService(repository EstablishmentRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) FindEstablishments(ctx context.Context) ([]domain.Establishment, error) {
	return s.repository.SelectEstablishments(ctx)
}

func (s *Service) ChangeAvailability(ctx context.Context, establishmentID string, available bool) (*domain.Establishment, error) {
	return s.repository.UpdateEstablishmentAvailability(ctx, establishmentID, available)
}
