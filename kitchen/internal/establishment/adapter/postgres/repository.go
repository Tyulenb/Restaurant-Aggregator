package adapterpostgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/adapter/postgres/sqlc"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/application"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/domain"
	transactor "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/postgres"
)

var _ application.EstablishmentRepository = (*EstablishmentRepository)(nil)

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

func (e *EstablishmentRepository) SelectEstablishments(ctx context.Context, available *bool) ([]domain.Establishment, error) {
	var (
		rows []sqlc.Establishment
		err  error
	)
	if available == nil {
		rows, err = e.getQueries(ctx).GetEstablishments(ctx)
	} else {
		rows, err = e.getQueries(ctx).GetEstablishmentsByAvailability(ctx, *available)
	}
	if err != nil {
		return nil, fmt.Errorf("get establishments: %w", err)
	}

	establishments := make([]domain.Establishment, 0, len(rows))
	for _, row := range rows {
		establishments = append(establishments, toDomain(row))
	}
	return establishments, nil
}

func (e *EstablishmentRepository) SelectEstablishmentById(
	ctx context.Context,
	establishmentID string,
) (*domain.Establishment, error) {
	id, err := uuid.Parse(establishmentID)
	if err != nil {
		return nil, fmt.Errorf("parse establishment ID: %w", err)
	}
	row, err := e.getQueries(ctx).GetEstablishmentByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get establishment by ID: %w", err)
	}
	return &domain.Establishment{
		ID: row.ID.String(), Name: row.Name, Description: nullableString(row.Description),
		Available: row.Available, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (e *EstablishmentRepository) SelectPartnerEstablishmentById(
	ctx context.Context,
	establishmentID string,
) (*domain.Establishment, error) {
	id, err := uuid.Parse(establishmentID)
	if err != nil {
		return nil, fmt.Errorf("parse establishment ID: %w", err)
	}
	row, err := e.getQueries(ctx).GetEstablishmentByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get partner establishment by ID: %w", err)
	}

	establishment := toDomain(row)
	return &establishment, nil
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
	return domain.Establishment{
		ID:          row.ID.String(),
		Name:        row.Name,
		Description: nullableString(row.Description),
		Available:   row.Available,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}

func nullableString(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}
