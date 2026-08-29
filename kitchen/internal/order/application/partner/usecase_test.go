package partner_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application/partner"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application/partner/mocks"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

const (
	establishmentID = "establishment-1"
	orderID         = "order-1"
)

func TestPartnerUseCase_FindNewOrdersByEstablishmentId(t *testing.T) {
	t.Run("returns repository orders", func(t *testing.T) {
		expected := []domain.Order{
			{ID: orderID, EstablishmentID: establishmentID, StatusID: domain.OrderCreated},
			{ID: "order-2", EstablishmentID: establishmentID, StatusID: domain.OrderCreated},
		}
		orderRepo := mocks.NewMockOrderRepository(t)
		orderRepo.EXPECT().FindNewOrdersByEstablishmentId(mock.Anything, establishmentID).
			Return(expected, nil).Once()
		useCase := partner.NewPartnerUseCase(orderRepo, mocks.NewMockTransactor(t))

		orders, err := useCase.FindNewOrdersByEstablishmentId(t.Context(), establishmentID)

		require.NoError(t, err)
		assert.Equal(t, expected, orders)
	})

	t.Run("wraps repository error", func(t *testing.T) {
		repositoryErr := errors.New("database unavailable")
		orderRepo := mocks.NewMockOrderRepository(t)
		orderRepo.EXPECT().FindNewOrdersByEstablishmentId(mock.Anything, establishmentID).
			Return(nil, repositoryErr).Once()
		useCase := partner.NewPartnerUseCase(orderRepo, mocks.NewMockTransactor(t))

		orders, err := useCase.FindNewOrdersByEstablishmentId(t.Context(), establishmentID)

		require.ErrorIs(t, err, repositoryErr)
		assert.ErrorContains(t, err, `find new orders for establishment "`+establishmentID+`"`)
		assert.Nil(t, orders)
	})
}

func TestPartnerUseCase_FindOrderByOrderIdAndEstablishmentId(t *testing.T) {
	tests := []struct {
		name            string
		order           *domain.Order
		repositoryErr   error
		establishmentID string
		wantErr         error
	}{
		{
			name:            "returns establishment order",
			order:           &domain.Order{ID: orderID, EstablishmentID: establishmentID},
			establishmentID: establishmentID,
		},
		{
			name:            "repository error",
			repositoryErr:   errors.New("database unavailable"),
			establishmentID: establishmentID,
		},
		{
			name:            "missing order",
			establishmentID: establishmentID,
			wantErr:         domain.ErrOrderNotFound,
		},
		{
			name:            "order belongs to another establishment",
			order:           &domain.Order{ID: orderID, EstablishmentID: "establishment-2"},
			establishmentID: establishmentID,
			wantErr:         domain.ErrOrderNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := mocks.NewMockOrderRepository(t)
			orderRepo.EXPECT().FindOrderByIdForUpdate(mock.Anything, orderID).Return(tt.order, tt.repositoryErr).Once()
			useCase := partner.NewPartnerUseCase(orderRepo, mocks.NewMockTransactor(t))

			order, err := useCase.FindOrderByOrderIdAndEstablishmentId(
				t.Context(), orderID, tt.establishmentID,
			)

			if tt.repositoryErr != nil {
				require.ErrorIs(t, err, tt.repositoryErr)
				assert.ErrorContains(t, err, `find order "`+orderID+`"`)
				assert.Nil(t, order)
				return
			}
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, order)
				return
			}
			require.NoError(t, err)
			assert.Same(t, tt.order, order)
		})
	}
}

func TestPartnerUseCase_ChangeOrderStatus_AllowsLifecycleTransitions(t *testing.T) {
	tests := []struct {
		name string
		from domain.OrderStatusID
		to   domain.OrderStatusID
	}{
		{name: "created to accepted", from: domain.OrderCreated, to: domain.OrderAccepted},
		{name: "created to rejected", from: domain.OrderCreated, to: domain.OrderRejected},
		{name: "accepted to preparing", from: domain.OrderAccepted, to: domain.OrderPreparing},
		{name: "preparing to ready", from: domain.OrderPreparing, to: domain.OrderReady},
		{name: "ready to completed", from: domain.OrderReady, to: domain.OrderCompleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := &domain.Order{ID: orderID, EstablishmentID: establishmentID, StatusID: tt.from}
			updated := &domain.Order{ID: orderID, EstablishmentID: establishmentID, StatusID: tt.to}
			orderRepo := mocks.NewMockOrderRepository(t)
			orderRepo.EXPECT().FindOrderByIdForUpdate(mock.Anything, orderID).Return(current, nil).Once()
			orderRepo.EXPECT().UpdateOrderStatus(mock.Anything, orderID, int(tt.to)).Return(updated, nil).Once()
			useCase := partner.NewPartnerUseCase(orderRepo, executingTransactor(t))

			order, err := useCase.ChangeOrderStatus(t.Context(), orderID, establishmentID, int(tt.to))

			require.NoError(t, err)
			assert.Same(t, updated, order)
		})
	}
}

func TestPartnerUseCase_ChangeOrderStatus_IsIdempotentForPartnerStatuses(t *testing.T) {
	statuses := []domain.OrderStatusID{
		domain.OrderAccepted,
		domain.OrderPreparing,
		domain.OrderReady,
		domain.OrderCompleted,
		domain.OrderRejected,
	}

	for _, status := range statuses {
		t.Run(statusName(status), func(t *testing.T) {
			current := &domain.Order{ID: orderID, EstablishmentID: establishmentID, StatusID: status}
			orderRepo := mocks.NewMockOrderRepository(t)
			orderRepo.EXPECT().FindOrderByIdForUpdate(mock.Anything, orderID).Return(current, nil).Once()
			useCase := partner.NewPartnerUseCase(orderRepo, executingTransactor(t))

			order, err := useCase.ChangeOrderStatus(t.Context(), orderID, establishmentID, int(status))

			require.NoError(t, err)
			assert.Same(t, current, order)
		})
	}
}

func TestPartnerUseCase_ChangeOrderStatus_RejectsUnsupportedTargetStatus(t *testing.T) {
	statuses := []domain.OrderStatusID{
		domain.OrderCreated,
		domain.OrderCanceled,
		domain.OrderStatusID(99),
	}

	for _, status := range statuses {
		t.Run(statusName(status), func(t *testing.T) {
			current := &domain.Order{ID: orderID, EstablishmentID: establishmentID, StatusID: domain.OrderCreated}
			orderRepo := mocks.NewMockOrderRepository(t)
			orderRepo.EXPECT().FindOrderByIdForUpdate(mock.Anything, orderID).Return(current, nil).Once()
			useCase := partner.NewPartnerUseCase(orderRepo, executingTransactor(t))

			order, err := useCase.ChangeOrderStatus(t.Context(), orderID, establishmentID, int(status))

			require.ErrorIs(t, err, domain.ErrInvalidOrderTransition)
			assert.ErrorContains(t, err, "unsupported requested status")
			assert.Nil(t, order)
		})
	}
}

func TestPartnerUseCase_ChangeOrderStatus_RejectsInvalidLifecycleTransitions(t *testing.T) {
	tests := []struct {
		name string
		from domain.OrderStatusID
		to   domain.OrderStatusID
	}{
		{name: "created skips accepted", from: domain.OrderCreated, to: domain.OrderPreparing},
		{name: "accepted is rejected", from: domain.OrderAccepted, to: domain.OrderRejected},
		{name: "accepted skips preparing", from: domain.OrderAccepted, to: domain.OrderReady},
		{name: "preparing moves backwards", from: domain.OrderPreparing, to: domain.OrderAccepted},
		{name: "ready moves backwards", from: domain.OrderReady, to: domain.OrderPreparing},
		{name: "completed changes status", from: domain.OrderCompleted, to: domain.OrderRejected},
		{name: "rejected changes status", from: domain.OrderRejected, to: domain.OrderAccepted},
		{name: "cancelled changes status", from: domain.OrderCanceled, to: domain.OrderAccepted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := &domain.Order{ID: orderID, EstablishmentID: establishmentID, StatusID: tt.from}
			orderRepo := mocks.NewMockOrderRepository(t)
			orderRepo.EXPECT().FindOrderByIdForUpdate(mock.Anything, orderID).Return(current, nil).Once()
			useCase := partner.NewPartnerUseCase(orderRepo, executingTransactor(t))

			order, err := useCase.ChangeOrderStatus(t.Context(), orderID, establishmentID, int(tt.to))

			require.ErrorIs(t, err, domain.ErrInvalidOrderTransition)
			assert.ErrorContains(t, err, "current status")
			assert.Nil(t, order)
		})
	}
}

func TestPartnerUseCase_ChangeOrderStatus_PropagatesLookupAndUpdateErrors(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	t.Run("lookup", func(t *testing.T) {
		orderRepo := mocks.NewMockOrderRepository(t)
		orderRepo.EXPECT().FindOrderByIdForUpdate(mock.Anything, orderID).Return(nil, repositoryErr).Once()
		useCase := partner.NewPartnerUseCase(orderRepo, executingTransactor(t))

		order, err := useCase.ChangeOrderStatus(
			t.Context(), orderID, establishmentID, int(domain.OrderAccepted),
		)

		require.ErrorIs(t, err, repositoryErr)
		assert.Nil(t, order)
	})

	t.Run("order belongs to another establishment", func(t *testing.T) {
		foreign := &domain.Order{ID: orderID, EstablishmentID: "establishment-2", StatusID: domain.OrderCreated}
		orderRepo := mocks.NewMockOrderRepository(t)
		orderRepo.EXPECT().FindOrderByIdForUpdate(mock.Anything, orderID).Return(foreign, nil).Once()
		useCase := partner.NewPartnerUseCase(orderRepo, executingTransactor(t))

		order, err := useCase.ChangeOrderStatus(
			t.Context(), orderID, establishmentID, int(domain.OrderAccepted),
		)

		require.ErrorIs(t, err, domain.ErrOrderNotFound)
		assert.Nil(t, order)
	})

	t.Run("update", func(t *testing.T) {
		current := &domain.Order{ID: orderID, EstablishmentID: establishmentID, StatusID: domain.OrderCreated}
		orderRepo := mocks.NewMockOrderRepository(t)
		orderRepo.EXPECT().FindOrderByIdForUpdate(mock.Anything, orderID).Return(current, nil).Once()
		orderRepo.EXPECT().UpdateOrderStatus(mock.Anything, orderID, int(domain.OrderAccepted)).
			Return(nil, repositoryErr).Once()
		useCase := partner.NewPartnerUseCase(orderRepo, executingTransactor(t))

		order, err := useCase.ChangeOrderStatus(
			t.Context(), orderID, establishmentID, int(domain.OrderAccepted),
		)

		require.ErrorIs(t, err, repositoryErr)
		assert.ErrorContains(t, err, `update order "`+orderID+`" status`)
		assert.Nil(t, order)
	})
}

func TestPartnerUseCase_ChangeOrderStatus_RequiresAndPropagatesTransaction(t *testing.T) {
	t.Run("missing transactor", func(t *testing.T) {
		useCase := partner.NewPartnerUseCase(mocks.NewMockOrderRepository(t), nil)

		order, err := useCase.ChangeOrderStatus(
			t.Context(), orderID, establishmentID, int(domain.OrderAccepted),
		)

		require.ErrorContains(t, err, "transactor is not configured")
		assert.Nil(t, order)
	})

	t.Run("transaction error", func(t *testing.T) {
		transactionErr := errors.New("begin transaction")
		tx := mocks.NewMockTransactor(t)
		tx.EXPECT().WithTx(mock.Anything, mock.Anything).Return(transactionErr).Once()
		useCase := partner.NewPartnerUseCase(mocks.NewMockOrderRepository(t), tx)

		order, err := useCase.ChangeOrderStatus(
			t.Context(), orderID, establishmentID, int(domain.OrderAccepted),
		)

		require.ErrorIs(t, err, transactionErr)
		assert.ErrorContains(t, err, `change order "`+orderID+`" status`)
		assert.Nil(t, order)
	})
}

func executingTransactor(t *testing.T) *mocks.MockTransactor {
	t.Helper()
	tx := mocks.NewMockTransactor(t)
	tx.EXPECT().WithTx(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, callback func(context.Context) error) error {
			return callback(ctx)
		}).Once()
	return tx
}

func statusName(status domain.OrderStatusID) string {
	switch status {
	case domain.OrderCreated:
		return "created"
	case domain.OrderAccepted:
		return "accepted"
	case domain.OrderPreparing:
		return "preparing"
	case domain.OrderReady:
		return "ready"
	case domain.OrderCompleted:
		return "completed"
	case domain.OrderRejected:
		return "rejected"
	case domain.OrderCanceled:
		return "cancelled"
	default:
		return "unknown"
	}
}
