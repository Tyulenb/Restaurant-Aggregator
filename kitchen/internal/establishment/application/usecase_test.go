package application_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/application"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/application/mocks"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/domain"
)

const establishmentID = "establishment-1"

func TestUseCase_FindEstablishments_ReturnsRepositoryEstablishments(t *testing.T) {
	ctx := t.Context()
	description := "Bakery and coffee shop"
	createdAt := time.Now().Add(-time.Hour)
	updatedAt := time.Now()
	expected := []domain.Establishment{
		{
			ID:          establishmentID,
			Name:        "Bakery",
			Description: &description,
			Available:   true,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		},
		{
			ID:        "establishment-2",
			Name:      "Closed cafe",
			Available: false,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
	}
	repository := mocks.NewMockEstablishmentRepository(t)
	repository.EXPECT().SelectEstablishments(ctx).Return(expected, nil).Once()
	useCase := application.NewUseCase(repository)

	establishments, err := useCase.FindEstablishments(ctx)

	require.NoError(t, err)
	assert.Equal(t, expected, establishments)
}

func TestUseCase_FindEstablishments_ReturnsEmptyRepositoryResult(t *testing.T) {
	ctx := t.Context()
	repository := mocks.NewMockEstablishmentRepository(t)
	repository.EXPECT().SelectEstablishments(ctx).Return([]domain.Establishment{}, nil).Once()
	useCase := application.NewUseCase(repository)

	establishments, err := useCase.FindEstablishments(ctx)

	require.NoError(t, err)
	assert.Empty(t, establishments)
	assert.NotNil(t, establishments)
}

func TestUseCase_FindEstablishments_PropagatesRepositoryError(t *testing.T) {
	ctx := t.Context()
	repositoryErr := errors.New("database unavailable")
	repository := mocks.NewMockEstablishmentRepository(t)
	repository.EXPECT().SelectEstablishments(ctx).Return(nil, repositoryErr).Once()
	useCase := application.NewUseCase(repository)

	establishments, err := useCase.FindEstablishments(ctx)

	require.ErrorIs(t, err, repositoryErr)
	assert.Nil(t, establishments)
}

func TestUseCase_ChangeAvailability_UpdatesExplicitAvailability(t *testing.T) {
	tests := []struct {
		name      string
		available bool
	}{
		{name: "resume accepting orders", available: true},
		{name: "pause accepting orders", available: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			expected := &domain.Establishment{
				ID:        establishmentID,
				Name:      "Bakery",
				Available: tt.available,
			}
			repository := mocks.NewMockEstablishmentRepository(t)
			repository.EXPECT().UpdateEstablishmentAvailability(ctx, establishmentID, tt.available).
				Return(expected, nil).Once()
			useCase := application.NewUseCase(repository)

			establishment, err := useCase.ChangeAvailability(ctx, establishmentID, tt.available)

			require.NoError(t, err)
			assert.Same(t, expected, establishment)
			assert.Equal(t, tt.available, establishment.Available)
		})
	}
}

func TestUseCase_ChangeAvailability_PropagatesRepositoryError(t *testing.T) {
	ctx := t.Context()
	repositoryErr := errors.New("database unavailable")
	repository := mocks.NewMockEstablishmentRepository(t)
	repository.EXPECT().UpdateEstablishmentAvailability(ctx, establishmentID, true).
		Return(nil, repositoryErr).Once()
	useCase := application.NewUseCase(repository)

	establishment, err := useCase.ChangeAvailability(ctx, establishmentID, true)

	require.ErrorIs(t, err, repositoryErr)
	assert.Nil(t, establishment)
}
