package partner

import (
	"context"
	"fmt"
	"sort"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type PartnerUseCase struct {
	OrderRepo OrderRepository
	PrRepo    ProductRepository
	Tx        Transactor
}

func NewPartnerUseCase(orderRepo OrderRepository, tx Transactor, PrRepo ProductRepository) *PartnerUseCase {
	return &PartnerUseCase{
		OrderRepo: orderRepo,
		PrRepo:    PrRepo,
		Tx:        tx,
	}
}

func (pu *PartnerUseCase) FindOrdersByEstablishmentID(ctx context.Context, establishmentID string, status *int) ([]domain.Order, error) {
	orders, err := pu.OrderRepo.FindOrdersByEstablishmentID(ctx, establishmentID, status)
	if err != nil {
		return nil, fmt.Errorf("find orders for establishment %q: %w", establishmentID, err)
	}

	return orders, nil
}

func (pu *PartnerUseCase) RejectOrder(
	ctx context.Context, orderID, establishmentID string, reason *string,
) (*domain.Order, error) {
	return pu.changeOrderStatus(ctx, orderID, establishmentID, domain.OrderRejected, reason)
}

func (pu *PartnerUseCase) FindOrderByOrderIdAndEstablishmentId(ctx context.Context, orderID, establishmentID string) (*domain.Order, error) {
	order, err := pu.OrderRepo.FindOrderByIdForUpdate(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("find order %q: %w", orderID, err)
	}
	if order == nil || order.EstablishmentID != establishmentID {
		return nil, domain.ErrOrderNotFound
	}

	return order, nil
}

// Find order + update order status in one transaction
func (pu *PartnerUseCase) ChangeOrderStatus(ctx context.Context, orderID, establishmentID string, statusID int) (*domain.Order, error) {
	return pu.changeOrderStatus(ctx, orderID, establishmentID, domain.OrderStatusID(statusID), nil)
}

func (pu *PartnerUseCase) changeOrderStatus(
	ctx context.Context, orderID, establishmentID string, requestedStatus domain.OrderStatusID, rejectionReason *string,
) (*domain.Order, error) {
	if pu.Tx == nil {
		return nil, fmt.Errorf("change order status: transactor is not configured")
	}
	var updatedOrder *domain.Order
	err := pu.Tx.WithTx(ctx, func(ctx context.Context) error {
		order, err := pu.FindOrderByOrderIdAndEstablishmentId(ctx, orderID, establishmentID)
		if err != nil {
			return err
		}
		if !isPartnerTargetStatus(requestedStatus) {
			return fmt.Errorf("%w: unsupported requested status %d", domain.ErrInvalidOrderTransition, requestedStatus)
		}
		if order.StatusID == requestedStatus {
			updatedOrder = order
			return nil
		}
		if !isAllowedPartnerTransition(order.StatusID, requestedStatus) {
			return fmt.Errorf(
				"%w: current status %d, requested status %d",
				domain.ErrInvalidOrderTransition,
				order.StatusID,
				requestedStatus,
			)
		}

		if requestedStatus == domain.OrderRejected {
			updatedOrder, err = pu.OrderRepo.RejectOrder(ctx, orderID, rejectionReason)
		} else {
			updatedOrder, err = pu.OrderRepo.UpdateOrderStatus(ctx, orderID, int(requestedStatus))
		}
		if err != nil {
			return fmt.Errorf("update order %q status: %w", orderID, err)
		}
		if updatedOrder == nil {
			return fmt.Errorf("update order %q status: repository returned a nil order", orderID)
		}
		if requestedStatus == domain.OrderRejected {
			if restoreErr := pu.restoreFiniteStock(ctx, order.Items); restoreErr != nil {
				return fmt.Errorf("update order %q status: %w", orderID, restoreErr)
			}
		}
		updatedOrder.Items = order.Items
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("change order %q status: %w", orderID, err)
	}

	return updatedOrder, nil
}

func (pu *PartnerUseCase) restoreFiniteStock(ctx context.Context, items []domain.OrderItem) error {
	quantities := make(map[string]int, len(items))
	for _, item := range items {
		quantities[item.ProductID] += item.Quantity
	}
	productIDs := make([]string, 0, len(quantities))
	for productID := range quantities {
		productIDs = append(productIDs, productID)
	}
	sort.Strings(productIDs)

	for _, productID := range productIDs {
		product, err := pu.PrRepo.FindProductByIdForUpdate(ctx, productID)
		if err != nil {
			return fmt.Errorf("lock product %q for stock restoration: %w", productID, err)
		}
		if product == nil {
			return fmt.Errorf("lock product %q for stock restoration: product not found", productID)
		}
		if product.InventoryTypeID != domain.Finite {
			continue
		}
		if err := pu.PrRepo.IncrementProductQuantity(ctx, productID, quantities[productID]); err != nil {
			return fmt.Errorf("restore product %q stock: %w", productID, err)
		}
	}
	return nil
}

func isPartnerTargetStatus(status domain.OrderStatusID) bool {
	switch status {
	case domain.OrderAccepted,
		domain.OrderPreparing,
		domain.OrderReady,
		domain.OrderCompleted,
		domain.OrderRejected:
		return true
	default:
		return false
	}
}

func isAllowedPartnerTransition(currentStatus, requestedStatus domain.OrderStatusID) bool {
	switch currentStatus {
	case domain.OrderCreated:
		return requestedStatus == domain.OrderAccepted || requestedStatus == domain.OrderRejected
	case domain.OrderAccepted:
		return requestedStatus == domain.OrderPreparing
	case domain.OrderPreparing:
		return requestedStatus == domain.OrderReady
	case domain.OrderReady:
		return requestedStatus == domain.OrderCompleted
	default:
		return false
	}
}
