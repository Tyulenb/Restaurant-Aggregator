package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

var (
	ErrOrderNotFound                 = errors.New("order not found")
	ErrEmptyOrder                    = errors.New("order must contain at least one item")
	ErrInvalidItemQuantity           = errors.New("order item quantity must be positive")
	ErrProductDeleted                = errors.New("product is deleted")
	ErrProductFromOtherEstablishment = errors.New("product belongs to another establishment")
	ErrProductUnavailable            = errors.New("product is unavailable")
	ErrInsufficientStock             = errors.New("insufficient product stock")
	ErrIdempotencyConflict           = errors.New("idempotency key was already used for another request")
	ErrOrderCannotBeCancelled        = errors.New("order can be cancelled only before it is accepted")
)

type CustomerUseCase struct {
	OrderRepo   OrderRepository
	ProductRepo ProductRepository
}

func NewCustomerUseCase(or OrderRepository, pr ProductRepository) *CustomerUseCase {
	return &CustomerUseCase{
		OrderRepo:   or,
		ProductRepo: pr,
	}
}

func (cu *CustomerUseCase) CreateOrder(ctx context.Context, establishmentID, idempotencyKey string, orderItems []domain.CreateOrderItem) (*domain.Order, error) {
	if len(orderItems) == 0 {
		return nil, ErrEmptyOrder
	}

	requestedQuantities := make(map[string]int, len(orderItems))
	for _, item := range orderItems {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("%w: product %q", ErrInvalidItemQuantity, item.ProductID)
		}
		requestedQuantities[item.ProductID] += item.Quantity
	}

	existingOrder, err := cu.OrderRepo.FindOrderByIdempotencyKey(ctx, idempotencyKey)
	if err != nil && !errors.Is(err, ErrOrderNotFound) {
		return nil, fmt.Errorf("find order by idempotency key: %w", err)
	}
	if existingOrder != nil {
		if sameOrderRequest(*existingOrder, establishmentID, requestedQuantities) {
			return existingOrder, nil
		}

		return nil, ErrIdempotencyConflict
	}

	products := make(map[string]*domain.Product, len(requestedQuantities))
	for productID, requestedQuantity := range requestedQuantities {
		product, findErr := cu.ProductRepo.FindProductById(ctx, productID)
		if findErr != nil {
			return nil, fmt.Errorf("find product %q: %w", productID, findErr)
		}
		if product == nil {
			return nil, fmt.Errorf("find product %q: product not found", productID)
		}
		if product.DeletedAt != nil {
			return nil, fmt.Errorf("%w: %s", ErrProductDeleted, productID)
		}
		if product.EstablishmentID != establishmentID {
			return nil, fmt.Errorf("%w: %s", ErrProductFromOtherEstablishment, productID)
		}
		if !product.Available {
			return nil, fmt.Errorf("%w: %s", ErrProductUnavailable, productID)
		}
		if product.InventoryTypeID == domain.Finite &&
			(product.Quantity == nil || requestedQuantity > *product.Quantity) {
			return nil, fmt.Errorf("%w: product %s, requested %d", ErrInsufficientStock, productID, requestedQuantity)
		}

		products[productID] = product
	}

	now := time.Now()
	placeholderID := uuid.NewString()
	order := domain.Order{
		ID:              uuid.NewString(),
		IdempotencyKey:  idempotencyKey,
		EstablishmentID: establishmentID,
		CustomerID:      placeholderID,
		DeliveryID:      placeholderID,
		StatusID:        domain.OrderCreated,
		Items:           make([]domain.OrderItem, 0, len(requestedQuantities)),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	for productID, requestedQuantity := range requestedQuantities {
		product := products[productID]
		totalSum := product.Price * float32(requestedQuantity)
		order.TotalPrice += totalSum
		order.Items = append(order.Items, domain.OrderItem{
			ID:                uuid.NewString(),
			ProductID:         product.ID,
			ExternalProductID: product.ExternalID,
			ProductName:       product.Name,
			UnitPrice:         product.Price,
			Quantity:          requestedQuantity,
			TotalSum:          totalSum,
		})
	}

	createdOrder, err := cu.OrderRepo.InsertOrder(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("insert order: %w", err)
	}

	return createdOrder, nil
}

func (cu *CustomerUseCase) FindOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	order, err := cu.OrderRepo.FindOrderById(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("find order %q: %w", orderID, err)
	}

	return order, nil
}

func (cu *CustomerUseCase) CancelOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	order, err := cu.OrderRepo.FindOrderById(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("find order %q: %w", orderID, err)
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}

	switch order.StatusID {
	case domain.OrderCanceled:
		return order, nil
	case domain.OrderCreated:
		cancelledOrder, updateErr := cu.OrderRepo.UpdateOrderStatus(ctx, orderID, int(domain.OrderCanceled))
		if updateErr != nil {
			return nil, fmt.Errorf("cancel order %q: %w", orderID, updateErr)
		}

		return cancelledOrder, nil
	default:
		return nil, fmt.Errorf("%w: current StatusID %d", ErrOrderCannotBeCancelled, order.StatusID)
	}
}

func sameOrderRequest(order domain.Order, establishmentID string, requestedQuantities map[string]int) bool {
	if order.EstablishmentID != establishmentID {
		return false
	}

	storedQuantities := make(map[string]int, len(order.Items))
	for _, item := range order.Items {
		storedQuantities[item.ProductID] += item.Quantity
	}
	if len(storedQuantities) != len(requestedQuantities) {
		return false
	}
	for productID, quantity := range requestedQuantities {
		if storedQuantities[productID] != quantity {
			return false
		}
	}

	return true
}
