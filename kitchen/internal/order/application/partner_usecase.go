package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

var ErrInvalidOrderTransition = errors.New("requested order state transition is not allowed")

type PartnerUseCase struct {
	OrderRepo OrderRepository
}

func NewPartnerUseCase(orderRepo OrderRepository) *PartnerUseCase {
	return &PartnerUseCase{OrderRepo: orderRepo}
}

func (pu *PartnerUseCase) FindNewOrdersByEstablishmentId(ctx context.Context, establishmentID string) ([]domain.Order, error) {
	orders, err := pu.OrderRepo.FindNewOrdersByEstablishmentId(ctx, establishmentID)
	if err != nil {
		return nil, fmt.Errorf("find new orders for establishment %q: %w", establishmentID, err)
	}

	return orders, nil
}

func (pu *PartnerUseCase) FindOrderByOrderIdAndEstablishmentId(ctx context.Context, orderID, establishmentID string) (*domain.Order, error) {
	order, err := pu.OrderRepo.FindOrderById(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("find order %q: %w", orderID, err)
	}
	if order == nil || order.EstablishmentID != establishmentID {
		return nil, ErrOrderNotFound
	}

	return order, nil
}

func (pu *PartnerUseCase) ChangeOrderStatus(ctx context.Context, orderID, establishmentID string, statusID int) (*domain.Order, error) {
	order, err := pu.FindOrderByOrderIdAndEstablishmentId(ctx, orderID, establishmentID)
	if err != nil {
		return nil, err
	}

	requestedStatus := domain.OrderStatusID(statusID)
	if !isPartnerTargetStatus(requestedStatus) {
		return nil, fmt.Errorf("%w: unsupported requested status %d", ErrInvalidOrderTransition, requestedStatus)
	}
	if order.StatusID == requestedStatus {
		return order, nil
	}
	if !isAllowedPartnerTransition(order.StatusID, requestedStatus) {
		return nil, fmt.Errorf(
			"%w: current status %d, requested status %d",
			ErrInvalidOrderTransition,
			order.StatusID,
			requestedStatus,
		)
	}

	updatedOrder, err := pu.OrderRepo.UpdateOrderStatus(ctx, orderID, statusID)
	if err != nil {
		return nil, fmt.Errorf("update order %q status: %w", orderID, err)
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
