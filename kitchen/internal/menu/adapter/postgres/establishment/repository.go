package establishment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	transactor "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/postgres"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/adapter/postgres/establishment/sqlc"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/establishment"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

var _ establishment.EstablishmentRepository = (*EstablishmentRepository)(nil)

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

func (r *EstablishmentRepository) SelectEstablishmentById(
	ctx context.Context,
	establishmentID string,
) (*domain.Establishment, error) {
	id, err := uuid.Parse(establishmentID)
	if err != nil {
		return nil, fmt.Errorf("parse establishment ID: %w", err)
	}

	row, err := r.getQueries(ctx).SelectEstablishmentByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select establishment by ID: %w", err)
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
