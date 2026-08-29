package category

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	transactor "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/postgres"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/adapter/postgres/category/sqlc"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/category"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/establishment"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

var _ establishment.CategoryRepository = (*CategoryRepository)(nil)
var _ category.CategoryRepository = (*CategoryRepository)(nil)

const uniqueViolationCode = "23505"

type CategoryRepository struct {
	queries *sqlc.Queries
}

func NewCategoryRepository(db sqlc.DBTX) *CategoryRepository {
	return &CategoryRepository{queries: sqlc.New(db)}
}

func (r *CategoryRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}

	return r.queries
}

func (r *CategoryRepository) SelectCategoriesWithDeleted(
	ctx context.Context,
	establishmentID string,
) ([]domain.Category, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	rows, err := r.getQueries(ctx).SelectCategoriesWithDeleted(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("select categories with deleted: %w", err)
	}

	return categoriesToDomain(rows), nil
}

func (r *CategoryRepository) SelectCategories(
	ctx context.Context,
	establishmentID string,
) ([]domain.Category, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	rows, err := r.getQueries(ctx).SelectCategories(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("select categories: %w", err)
	}

	return categoriesToDomain(rows), nil
}

func (r *CategoryRepository) SelectCategoryByExternalId(
	ctx context.Context,
	establishmentID, externalID string,
) (*domain.Category, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	row, err := r.getQueries(ctx).SelectCategoryByExternalID(ctx, sqlc.SelectCategoryByExternalIDParams{
		EstablishmentID: id,
		ExternalID:      externalID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select category by external ID: %w", err)
	}

	category := toDomain(row)
	return &category, nil
}

func (r *CategoryRepository) InsertCategory(ctx context.Context, category domain.Category) (*domain.Category, error) {
	params, err := insertCategoryParams(category)
	if err != nil {
		return nil, fmt.Errorf("prepare category for insertion: %w", err)
	}

	row, err := r.getQueries(ctx).InsertCategory(ctx, params)
	if isUniqueViolation(err) {
		return nil, fmt.Errorf("insert category: %w", domain.ErrCategoryConflict)
	}
	if err != nil {
		return nil, fmt.Errorf("insert category: %w", err)
	}

	insertedCategory := toDomain(row)
	return &insertedCategory, nil
}

func (r *CategoryRepository) UpdateCategoryNameByExternalId(
	ctx context.Context,
	establishmentID, externalID, name string,
) (*domain.Category, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	row, err := r.getQueries(ctx).UpdateCategoryNameByExternalID(ctx, sqlc.UpdateCategoryNameByExternalIDParams{
		Name:            name,
		EstablishmentID: id,
		ExternalID:      externalID,
	})
	return updatedCategory(row, err, "update category name")
}

func (r *CategoryRepository) UpdateCategoryDescriptionByExternalId(
	ctx context.Context,
	establishmentID, externalID string,
	description *string,
) (*domain.Category, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	row, err := r.getQueries(ctx).UpdateCategoryDescriptionByExternalID(
		ctx,
		sqlc.UpdateCategoryDescriptionByExternalIDParams{
			Description:     nullableText(description),
			EstablishmentID: id,
			ExternalID:      externalID,
		},
	)
	return updatedCategory(row, err, "update category description")
}

func (r *CategoryRepository) UpdateCategoryNameAndDescriptionByExternalId(
	ctx context.Context,
	establishmentID, externalID, name string,
	description *string,
) (*domain.Category, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	row, err := r.getQueries(ctx).UpdateCategoryNameAndDescriptionByExternalID(
		ctx,
		sqlc.UpdateCategoryNameAndDescriptionByExternalIDParams{
			Name:            name,
			Description:     nullableText(description),
			EstablishmentID: id,
			ExternalID:      externalID,
		},
	)
	return updatedCategory(row, err, "update category name and description")
}

func (r *CategoryRepository) DeleteCategoryByExternalId(
	ctx context.Context,
	establishmentID, externalID string,
) (*domain.Category, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	row, err := r.getQueries(ctx).DeleteCategoryByExternalID(ctx, sqlc.DeleteCategoryByExternalIDParams{
		EstablishmentID: id,
		ExternalID:      externalID,
	})
	return updatedCategory(row, err, "delete category")
}

func insertCategoryParams(category domain.Category) (sqlc.InsertCategoryParams, error) {
	id, err := parseUUID("category ID", category.ID)
	if err != nil {
		return sqlc.InsertCategoryParams{}, err
	}
	establishmentID, err := parseUUID("establishment ID", category.EstablishmentID)
	if err != nil {
		return sqlc.InsertCategoryParams{}, err
	}

	params := sqlc.InsertCategoryParams{
		ID:              id,
		EstablishmentID: establishmentID,
		ExternalID:      category.ExternalID,
		Name:            category.Name,
		CreatedAt:       pgtype.Timestamptz{Time: category.CreatedAt, Valid: true},
		UpdatedAt:       pgtype.Timestamptz{Time: category.UpdatedAt, Valid: true},
	}
	if category.Description != nil {
		params.Description = pgtype.Text{String: *category.Description, Valid: true}
	}
	if category.DeletedAt != nil {
		params.DeletedAt = pgtype.Timestamptz{Time: *category.DeletedAt, Valid: true}
	}

	return params, nil
}

func updatedCategory(row sqlc.Category, err error, operation string) (*domain.Category, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	category := toDomain(row)
	return &category, nil
}

func categoriesToDomain(rows []sqlc.Category) []domain.Category {
	categories := make([]domain.Category, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, toDomain(row))
	}

	return categories
}

func parseUUID(name, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse %s: %w", name, err)
	}

	return id, nil
}

func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == uniqueViolationCode
}

func nullableText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *value, Valid: true}
}

func toDomain(row sqlc.Category) domain.Category {
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}

	var deletedAt *time.Time
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}

	return domain.Category{
		ID:              row.ID.String(),
		EstablishmentID: row.EstablishmentID.String(),
		ExternalID:      row.ExternalID,
		Name:            row.Name,
		Description:     description,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
		DeletedAt:       deletedAt,
	}
}
