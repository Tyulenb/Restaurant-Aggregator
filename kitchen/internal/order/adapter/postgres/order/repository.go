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
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application/customer"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application/partner"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

var _ customer.OrderRepository = (*OrderRepository)(nil)
var _ partner.OrderRepository = (*OrderRepository)(nil)

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

func (r *OrderRepository) InsertOrderItem(ctx context.Context, item domain.OrderItem) (*domain.OrderItem, error) {
	id, err := uuid.Parse(item.ID)
	if err != nil {
		return nil, fmt.Errorf("parse order item ID: %w", err)
	}
	orderID, err := uuid.Parse(item.OrderID)
	if err != nil {
		return nil, fmt.Errorf("parse order ID: %w", err)
	}
	productID, err := uuid.Parse(item.ProductID)
	if err != nil {
		return nil, fmt.Errorf("parse product ID: %w", err)
	}
	unitPrice, err := numericFromFloat32(item.UnitPrice)
	if err != nil {
		return nil, fmt.Errorf("convert unit price: %w", err)
	}
	totalSum, err := numericFromFloat32(item.TotalSum)
	if err != nil {
		return nil, fmt.Errorf("convert total sum: %w", err)
	}
	row, err := r.getQueries(ctx).InsertOrderItem(ctx, sqlc.InsertOrderItemParams{
		ID: id, OrderID: orderID, ProductID: productID,
		ExternalProductID: item.ExternalProductID, ProductName: item.ProductName,
		UnitPrice: unitPrice, Quantity: int32(item.Quantity), TotalSum: totalSum,
	})
	if err != nil {
		return nil, fmt.Errorf("insert order item: %w", err)
	}
	item.ID = row.ID.String()
	return &item, nil
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

	if err := r.loadOrderItems(ctx, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *OrderRepository) FindOrderByIdForUpdate(ctx context.Context, orderID string) (*domain.Order, error) {
	id, err := parseUUID("order ID", orderID)
	if err != nil {
		return nil, err
	}

	row, err := r.getQueries(ctx).SelectOrderByIDForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select order by ID for update: %w", err)
	}

	result, err := toDomain(row)
	if err != nil {
		return nil, fmt.Errorf("convert order selected by ID for update: %w", err)
	}

	if err := r.loadOrderItems(ctx, &result); err != nil {
		return nil, err
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

	if err := r.loadOrderItems(ctx, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *OrderRepository) loadOrderItems(ctx context.Context, order *domain.Order) error {
	orderID, err := uuid.Parse(order.ID)
	if err != nil {
		return fmt.Errorf("parse order ID for items: %w", err)
	}
	rows, err := r.getQueries(ctx).SelectOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("select order items: %w", err)
	}
	order.Items = make([]domain.OrderItem, 0, len(rows))
	for _, row := range rows {
		unitPrice, priceErr := row.UnitPrice.Float64Value()
		if priceErr != nil {
			return fmt.Errorf("convert order item unit price: %w", priceErr)
		}
		totalSum, sumErr := row.TotalSum.Float64Value()
		if sumErr != nil {
			return fmt.Errorf("convert order item total sum: %w", sumErr)
		}
		order.Items = append(order.Items, domain.OrderItem{
			ID: row.ID.String(), OrderID: row.OrderID.String(), ProductID: row.ProductID.String(),
			ExternalProductID: row.ExternalProductID, ProductName: row.ProductName,
			UnitPrice: float32(unitPrice.Float64), Quantity: int(row.Quantity), TotalSum: float32(totalSum.Float64),
		})
	}
	return nil
}

func (r *OrderRepository) FindOrdersByEstablishmentID(
	ctx context.Context,
	establishmentID string,
	status *int,
) ([]domain.Order, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	var rows []sqlc.Order
	if status == nil {
		rows, err = r.getQueries(ctx).SelectOrdersByEstablishmentID(ctx, id)
	} else {
		rows, err = r.getQueries(ctx).SelectOrdersByEstablishmentIDAndStatus(
			ctx,
			sqlc.SelectOrdersByEstablishmentIDAndStatusParams{
				EstablishmentID: id,
				StatusID:        int32(*status),
			},
		)
	}
	if err != nil {
		return nil, fmt.Errorf("select orders by establishment ID: %w", err)
	}

	orders := make([]domain.Order, 0, len(rows))
	for _, row := range rows {
		order, convertErr := toDomain(row)
		if convertErr != nil {
			return nil, fmt.Errorf("convert order: %w", convertErr)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (r *OrderRepository) RejectOrder(
	ctx context.Context, orderID string, reason *string,
) (*domain.Order, error) {
	id, err := parseUUID("order ID", orderID)
	if err != nil {
		return nil, err
	}

	var rejectionReason pgtype.Text
	if reason != nil {
		rejectionReason = pgtype.Text{String: *reason, Valid: true}
	}
	row, err := r.getQueries(ctx).RejectOrderByID(ctx, sqlc.RejectOrderByIDParams{
		RejectionReason: rejectionReason,
		ID:              id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reject order: %w", err)
	}
	updatedOrder, err := toDomain(row)
	if err != nil {
		return nil, fmt.Errorf("convert rejected order: %w", err)
	}
	return &updatedOrder, nil
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

func numericFromFloat32(value float32) (pgtype.Numeric, error) {
	var numeric pgtype.Numeric
	if err := numeric.Scan(strconv.FormatFloat(float64(value), 'f', -1, 32)); err != nil {
		return pgtype.Numeric{}, err
	}
	return numeric, nil
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
