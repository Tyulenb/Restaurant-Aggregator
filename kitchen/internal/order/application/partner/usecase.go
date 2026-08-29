package partner

import (
	"context"
	"fmt"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type PartnerUseCase struct {
	OrderRepo OrderRepository
	Tx        Transactor
}

func NewPartnerUseCase(orderRepo OrderRepository, tx Transactor) *PartnerUseCase {
	return &PartnerUseCase{
		OrderRepo: orderRepo,
		Tx:        tx,
	}
}

func (pu *PartnerUseCase) FindNewOrdersByEstablishmentId(ctx context.Context, establishmentID string) ([]domain.Order, error) {
	orders, err := pu.OrderRepo.FindNewOrdersByEstablishmentId(ctx, establishmentID)
	if err != nil {
		return nil, fmt.Errorf("find new orders for establishment %q: %w", establishmentID, err)
	}

	return orders, nil
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
	if pu.Tx == nil {
		return nil, fmt.Errorf("change order status: transactor is not configured")
	}
	var updatedOrder *domain.Order
	err := pu.Tx.WithTx(ctx, func(ctx context.Context) error {
		order, err := pu.FindOrderByOrderIdAndEstablishmentId(ctx, orderID, establishmentID)
		if err != nil {
			return err
		}
		requestedStatus := domain.OrderStatusID(statusID)
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

		updatedOrder, err = pu.OrderRepo.UpdateOrderStatus(ctx, orderID, statusID)
		if err != nil {
			return fmt.Errorf("update order %q status: %w", orderID, err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("change order %q status: %w", orderID, err)
	}

	return updatedOrder, nil
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
