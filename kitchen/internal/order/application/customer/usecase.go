package customer

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type CustomerUseCase struct {
	OrderRepo         OrderRepository
	ProductRepo       ProductRepository
	EstablishmentRepo EstablishmentRepository
	Tx                Transactor
}

func NewCustomerUseCase(
	or OrderRepository, pr ProductRepository,
	er EstablishmentRepository, tx Transactor,
) *CustomerUseCase {
	return &CustomerUseCase{
		OrderRepo:         or,
		ProductRepo:       pr,
		EstablishmentRepo: er,
		Tx:                tx,
	}
}

type CreateOrderItem struct {
	ProductID string
	Quantity  int
}

func (cu *CustomerUseCase) CreateOrder(ctx context.Context, establishmentID, idempotencyKey string, orderItems []CreateOrderItem) (*domain.Order, error) {
	if len(orderItems) == 0 {
		return nil, domain.ErrEmptyOrder
	}

	requestedQuantities := make(map[string]int, len(orderItems))
	for _, item := range orderItems {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("%w: product %q", domain.ErrInvalidItemQuantity, item.ProductID)
		}
		requestedQuantities[item.ProductID] += item.Quantity
	}
	productIDs := make([]string, 0, len(requestedQuantities))
	for productID := range requestedQuantities {
		productIDs = append(productIDs, productID)
	}
	sort.Strings(productIDs)

	if cu.Tx == nil {
		return nil, errors.New("create order: transactor is not configured")
	}

	var createdOrder *domain.Order
	err := cu.Tx.WithTx(ctx, func(txCtx context.Context) error {
		existingOrder, findErr := cu.OrderRepo.FindOrderByIdempotencyKey(txCtx, idempotencyKey)
		if findErr != nil && !errors.Is(findErr, domain.ErrOrderNotFound) {
			return fmt.Errorf("find order by idempotency key: %w", findErr)
		}
		if existingOrder != nil {
			if !sameOrderRequest(*existingOrder, establishmentID, requestedQuantities) {
				return domain.ErrIdempotencyConflict
			}
			createdOrder = existingOrder
			return nil
		}

		establishment, selectErr := cu.EstablishmentRepo.FindEstablishmentById(txCtx, establishmentID)
		if selectErr != nil {
			return fmt.Errorf("select establishment %q: %w", establishmentID, selectErr)
		}
		if establishment == nil {
			return domain.ErrEstablishmentNotFound
		}
		if !establishment.Available {
			return domain.ErrEstablishmentUnavailable
		}

		products := make(map[string]*domain.Product, len(requestedQuantities))
		for _, productID := range productIDs {
			product, productErr := cu.ProductRepo.FindProductById(txCtx, productID)
			if productErr != nil {
				return fmt.Errorf("find product %q: %w", productID, productErr)
			}
			if validateErr := validateProduct(product, productID, establishmentID); validateErr != nil {
				return validateErr
			}
			products[productID] = product
		}

		for _, productID := range productIDs {
			quantity := requestedQuantities[productID]
			if products[productID].InventoryTypeID != domain.Finite {
				continue
			}
			product, lockErr := cu.ProductRepo.FindProductByIdForUpdate(txCtx, productID)
			if lockErr != nil {
				return fmt.Errorf("lock product %q: %w", productID, lockErr)
			}
			if validateErr := validateProduct(product, productID, establishmentID); validateErr != nil {
				return validateErr
			}
			if product.Quantity == nil || quantity > *product.Quantity {
				return fmt.Errorf("%w: product %s, requested %d", domain.ErrInsufficientStock, productID, quantity)
			}
			products[productID] = product
		}

		now := time.Now()
		placeholderID := uuid.NewString()
		order := domain.Order{
			ID: uuid.NewString(), IdempotencyKey: idempotencyKey,
			EstablishmentID: establishmentID, CustomerID: placeholderID, DeliveryID: placeholderID,
			StatusID: domain.OrderCreated, Items: make([]domain.OrderItem, 0, len(requestedQuantities)),
			CreatedAt: now, UpdatedAt: now,
		}
		for _, productID := range productIDs {
			quantity := requestedQuantities[productID]
			product := products[productID]
			total := product.Price * float32(quantity)
			order.TotalPrice += total
			order.Items = append(order.Items, domain.OrderItem{
				ID: uuid.NewString(), OrderID: order.ID, ProductID: product.ID,
				ExternalProductID: product.ExternalID, ProductName: product.Name,
				UnitPrice: product.Price, Quantity: quantity, TotalSum: total,
			})
		}

		for _, productID := range productIDs {
			quantity := requestedQuantities[productID]
			if products[productID].InventoryTypeID == domain.Finite {
				if decrementErr := cu.ProductRepo.DecrementProductQuantity(txCtx, productID, quantity); decrementErr != nil {
					return fmt.Errorf("decrement product %q stock: %w", productID, decrementErr)
				}
			}
		}

		inserted, insertErr := cu.OrderRepo.InsertOrder(txCtx, order)
		if insertErr != nil {
			return fmt.Errorf("insert order: %w", insertErr)
		}
		if inserted == nil {
			return errors.New("insert order: repository returned a nil order")
		}
		for _, item := range order.Items {
			if _, itemErr := cu.OrderRepo.InsertOrderItem(txCtx, item); itemErr != nil {
				return fmt.Errorf("insert order item for product %q: %w", item.ProductID, itemErr)
			}
		}
		inserted.Items = order.Items
		inserted.TotalPrice = order.TotalPrice
		createdOrder = inserted
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	return createdOrder, nil
}

func validateProduct(product *domain.Product, productID, establishmentID string) error {
	if product == nil {
		return fmt.Errorf("find product %q: product not found", productID)
	}
	if product.DeletedAt != nil {
		return fmt.Errorf("%w: %s", domain.ErrProductDeleted, productID)
	}
	if product.EstablishmentID != establishmentID {
		return fmt.Errorf("%w: %s", domain.ErrProductFromOtherEstablishment, productID)
	}
	if !product.Available {
		return fmt.Errorf("%w: %s", domain.ErrProductUnavailable, productID)
	}
	return nil
}

func (cu *CustomerUseCase) FindOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	order, err := cu.OrderRepo.FindOrderById(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("find order %q: %w", orderID, err)
	}

	return order, nil
}

// Find order + update order in one transaction
func (cu *CustomerUseCase) CancelOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	if cu.Tx == nil {
		return nil, errors.New("create order: transactor is not configured")
	}

	var cancelledOrder *domain.Order
	cu.Tx.WithTx(ctx, func(ctx context.Context) error {
		order, err := cu.OrderRepo.FindOrderByIdForUpdate(ctx, orderID)
		if err != nil {
			return fmt.Errorf("find order %q: %w", orderID, err)
		}
		if order == nil {
			return domain.ErrOrderNotFound
		}

		switch order.StatusID {
		case domain.OrderCanceled:
			cancelledOrder = order
			return nil
		case domain.OrderCreated:
			cancelledOrder, err = cu.OrderRepo.UpdateOrderStatus(ctx, orderID, int(domain.OrderCanceled))
			if err != nil {
				return fmt.Errorf("cancel order %q: %w", orderID, err)
			}

			return nil
		default:
			return fmt.Errorf("%w: current StatusID %d", domain.ErrOrderCannotBeCancelled, order.StatusID)
		}
	})
	return cancelledOrder, nil
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
