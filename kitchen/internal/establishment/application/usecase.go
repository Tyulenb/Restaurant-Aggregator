package application

import (
	"context"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/domain"
)

type UseCase struct {
	repository EstablishmentRepository
}

func NewUseCase(repository EstablishmentRepository) *UseCase {
	return &UseCase{repository: repository}
}

func (s *UseCase) FindEstablishments(ctx context.Context, available *bool) ([]domain.Establishment, error) {
	return s.repository.SelectEstablishments(ctx, available)
}

func (s *UseCase) ChangeAvailability(ctx context.Context, establishmentID string, available bool) (*domain.Establishment, error) {
	return s.repository.UpdateEstablishmentAvailability(ctx, establishmentID, available)
}

func (s *UseCase) FindPartnerEstablishment(ctx context.Context, establishmentId string) (*domain.Establishment, error) {
	return s.repository.SelectEstablishmentById(ctx, establishmentId)
}
