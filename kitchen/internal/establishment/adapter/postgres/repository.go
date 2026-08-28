package adapterpostgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/adapter/postgres/sqlc"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/domain"
	transactor "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/postgres"
)

type EstablishmentRepository struct {
	queries *sqlc.Queries
}

func NewEstablishmentRepository(db sqlc.DBTX) *EstablishmentRepository {
	return &EstablishmentRepository{queries: sqlc.New(db)}
}

func (r *EstablishmentRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (e *EstablishmentRepository) SelectEstablishments(ctx context.Context) ([]domain.Establishment, error) {
	rows, err := e.getQueries(ctx).GetAvailableEstablishments(ctx)
	if err != nil {
		return nil, fmt.Errorf("get available establishments: %w", err)
	}

	establishments := make([]domain.Establishment, 0, len(rows))
	for _, row := range rows {
		establishments = append(establishments, toDomain(row))
	}
	return establishments, nil
}

func (e *EstablishmentRepository) UpdateEstablishmentAvailability(
	ctx context.Context,
	establishmentID string,
	available bool,
) (*domain.Establishment, error) {
	id, err := uuid.Parse(establishmentID)
	if err != nil {
		return nil, fmt.Errorf("parse establishment ID: %w", err)
	}

	row, err := e.getQueries(ctx).UpdateEstablishmentAvailability(ctx, sqlc.UpdateEstablishmentAvailabilityParams{
		ID:        id,
		Available: available,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("update establishment availability: %w", pgx.ErrNoRows)
		}
		return nil, fmt.Errorf("update establishment availability: %w", err)
	}

	establishment := toDomain(row)
	return &establishment, nil
}

func toDomain(row sqlc.Establishment) domain.Establishment {
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}

	return domain.Establishment{
		ID:          row.ID.String(),
		Name:        row.Name,
		Description: description,
		Available:   row.Available,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}
