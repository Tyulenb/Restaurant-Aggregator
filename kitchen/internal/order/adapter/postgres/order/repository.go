package order

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	transactor "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/postgres"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/adapter/postgres/order/sqlc"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type OrderRepository struct {
	queries *sqlc.Queries
}

func NewOrderRepository(db sqlc.DBTX) *OrderRepository {
	return &OrderRepository{queries: sqlc.New(db)}
}

func (r *OrderRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}

	return r.queries
}

func (r *OrderRepository) InsertOrder(ctx context.Context, order domain.Order) (*domain.Order, error) {
	params, err := insertOrderParams(order)
	if err != nil {
		return nil, fmt.Errorf("prepare order for insertion: %w", err)
	}

	row, err := r.getQueries(ctx).InsertOrder(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("insert order: %w", err)
	}

	insertedOrder, err := toDomain(row)
	if err != nil {
		return nil, fmt.Errorf("convert inserted order: %w", err)
	}

	return &insertedOrder, nil
}

func (r *OrderRepository) FindOrderById(ctx context.Context, orderID string) (*domain.Order, error) {
	id, err := parseUUID("order ID", orderID)
	if err != nil {
		return nil, err
	}

	row, err := r.getQueries(ctx).SelectOrderByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select order by ID: %w", err)
	}

	result, err := toDomain(row)
	if err != nil {
		return nil, fmt.Errorf("convert order selected by ID: %w", err)
	}

	return &result, nil
}

func (r *OrderRepository) FindOrderByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.Order, error) {
	key, err := parseUUID("idempotency key", idempotencyKey)
	if err != nil {
		return nil, err
	}

	row, err := r.getQueries(ctx).SelectOrderByIdempotencyKey(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select order by idempotency key: %w", err)
	}

	result, err := toDomain(row)
	if err != nil {
		return nil, fmt.Errorf("convert order selected by idempotency key: %w", err)
	}

	return &result, nil
}

func (r *OrderRepository) FindNewOrdersByEstablishmentId(
	ctx context.Context,
	establishmentID string,
) ([]domain.Order, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	rows, err := r.getQueries(ctx).SelectCreatedOrdersByEstablishmentID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("select new orders by establishment ID: %w", err)
	}

	orders := make([]domain.Order, 0, len(rows))
	for _, row := range rows {
		order, convertErr := toDomain(row)
		if convertErr != nil {
			return nil, fmt.Errorf("convert new order: %w", convertErr)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, orderID string, status int) (*domain.Order, error) {
	id, err := parseUUID("order ID", orderID)
	if err != nil {
		return nil, err
	}

	row, err := r.getQueries(ctx).UpdateOrderStatusByID(ctx, sqlc.UpdateOrderStatusByIDParams{
		ID:       id,
		StatusID: int32(status),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}

	updatedOrder, err := toDomain(row)
	if err != nil {
		return nil, fmt.Errorf("convert updated order: %w", err)
	}

	return &updatedOrder, nil
}

func insertOrderParams(order domain.Order) (sqlc.InsertOrderParams, error) {
	id, err := parseUUID("order ID", order.ID)
	if err != nil {
		return sqlc.InsertOrderParams{}, err
	}
	idempotencyKey, err := parseUUID("idempotency key", order.IdempotencyKey)
	if err != nil {
		return sqlc.InsertOrderParams{}, err
	}
	establishmentID, err := parseUUID("establishment ID", order.EstablishmentID)
	if err != nil {
		return sqlc.InsertOrderParams{}, err
	}
	customerID, err := parseUUID("customer ID", order.CustomerID)
	if err != nil {
		return sqlc.InsertOrderParams{}, err
	}
	deliveryID, err := parseUUID("delivery ID", order.DeliveryID)
	if err != nil {
		return sqlc.InsertOrderParams{}, err
	}

	var totalPrice pgtype.Numeric
	if err = totalPrice.Scan(strconv.FormatFloat(float64(order.TotalPrice), 'f', -1, 32)); err != nil {
		return sqlc.InsertOrderParams{}, fmt.Errorf("convert total price: %w", err)
	}

	rejectionReason := pgtype.Text{}
	if order.RejectionReason != nil {
		rejectionReason = pgtype.Text{String: *order.RejectionReason, Valid: true}
	}

	return sqlc.InsertOrderParams{
		ID:              id,
		IdempotencyKey:  idempotencyKey,
		EstablishmentID: establishmentID,
		CustomerID:      customerID,
		DeliveryID:      deliveryID,
		StatusID:        int32(order.StatusID),
		TotalPrice:      totalPrice,
		RejectionReason: rejectionReason,
		CreatedAt:       pgtype.Timestamptz{Time: order.CreatedAt, Valid: true},
		UpdatedAt:       pgtype.Timestamptz{Time: order.UpdatedAt, Valid: true},
	}, nil
}

func parseUUID(name, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse %s: %w", name, err)
	}

	return id, nil
}

func toDomain(row sqlc.Order) (domain.Order, error) {
	totalPrice, err := row.TotalPrice.Float64Value()
	if err != nil {
		return domain.Order{}, fmt.Errorf("convert total price: %w", err)
	}

	var rejectionReason *string
	if row.RejectionReason.Valid {
		rejectionReason = &row.RejectionReason.String
	}

	return domain.Order{
		ID:              row.ID.String(),
		IdempotencyKey:  row.IdempotencyKey.String(),
		EstablishmentID: row.EstablishmentID.String(),
		CustomerID:      row.CustomerID.String(),
		DeliveryID:      row.DeliveryID.String(),
		StatusID:        domain.OrderStatusID(row.StatusID),
		TotalPrice:      float32(totalPrice.Float64),
		RejectionReason: rejectionReason,
		Items:           make([]domain.OrderItem, 0),
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}, nil
}
