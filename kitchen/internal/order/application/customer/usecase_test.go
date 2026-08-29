package customer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application/customer"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application/customer/mocks"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

const (
	establishmentID = "establishment-1"
	idempotencyKey  = "idempotency-key"
	productID       = "product-1"
	orderID         = "order-1"
)

func TestCustomerUseCase_CreateOrder_ValidatesRequestBeforeTransaction(t *testing.T) {
	tests := []struct {
		name  string
		items []customer.CreateOrderItem
		err   error
	}{
		{name: "empty order", err: domain.ErrEmptyOrder},
		{name: "zero quantity", items: []customer.CreateOrderItem{{ProductID: productID}}, err: domain.ErrInvalidItemQuantity},
		{name: "negative quantity", items: []customer.CreateOrderItem{{ProductID: productID, Quantity: -1}}, err: domain.ErrInvalidItemQuantity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase, _, _, _, _ := newUseCase(t)

			order, err := useCase.CreateOrder(t.Context(), establishmentID, idempotencyKey, tt.items)

			require.ErrorIs(t, err, tt.err)
			assert.Nil(t, order)
		})
	}
}

func TestCustomerUseCase_CreateOrder_RequiresTransactor(t *testing.T) {
	orderRepo := mocks.NewMockOrderRepository(t)
	productRepo := mocks.NewMockProductRepository(t)
	establishmentRepo := mocks.NewMockEstablishmentRepository(t)
	useCase := customer.NewCustomerUseCase(orderRepo, productRepo, establishmentRepo, nil)

	order, err := useCase.CreateOrder(t.Context(), establishmentID, idempotencyKey, validItems())

	require.ErrorContains(t, err, "transactor is not configured")
	assert.Nil(t, order)
}

func TestCustomerUseCase_CreateOrder_HandlesIdempotency(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	existing := &domain.Order{
		ID:              orderID,
		EstablishmentID: establishmentID,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 1},
			{ProductID: productID, Quantity: 2},
		},
	}
	tests := []struct {
		name            string
		existing        *domain.Order
		repoErr         error
		establishmentID string
		items           []customer.CreateOrderItem
		wantErr         error
	}{
		{name: "lookup error", repoErr: repositoryErr, establishmentID: establishmentID, items: validItems(), wantErr: repositoryErr},
		{name: "same aggregated request", existing: existing, establishmentID: establishmentID, items: []customer.CreateOrderItem{{ProductID: productID, Quantity: 2}, {ProductID: productID, Quantity: 1}}},
		{name: "different establishment", existing: existing, establishmentID: "establishment-2", items: validItems(), wantErr: domain.ErrIdempotencyConflict},
		{name: "different quantity", existing: existing, establishmentID: establishmentID, items: []customer.CreateOrderItem{{ProductID: productID, Quantity: 2}}, wantErr: domain.ErrIdempotencyConflict},
		{name: "different product", existing: existing, establishmentID: establishmentID, items: []customer.CreateOrderItem{{ProductID: "product-2", Quantity: 3}}, wantErr: domain.ErrIdempotencyConflict},
		{name: "different product count", existing: existing, establishmentID: establishmentID, items: []customer.CreateOrderItem{{ProductID: productID, Quantity: 3}, {ProductID: "product-2", Quantity: 1}}, wantErr: domain.ErrIdempotencyConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase, orderRepo, _, _, tx := newUseCase(t)
			ctx, txCtx := transactionContexts(t)
			expectTransaction(tx, ctx, txCtx)
			orderRepo.EXPECT().FindOrderByIdempotencyKey(txCtx, idempotencyKey).Return(tt.existing, tt.repoErr).Once()

			actual, err := useCase.CreateOrder(ctx, tt.establishmentID, idempotencyKey, tt.items)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, actual)
				return
			}
			require.NoError(t, err)
			assert.Same(t, existing, actual)
		})
	}
}

func TestCustomerUseCase_CreateOrder_ValidatesEstablishment(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	tests := []struct {
		name          string
		establishment *domain.Establishment
		repoErr       error
		wantErr       error
	}{
		{name: "repository error", repoErr: repositoryErr, wantErr: repositoryErr},
		{name: "not found", wantErr: domain.ErrEstablishmentNotFound},
		{name: "unavailable", establishment: &domain.Establishment{ID: establishmentID}, wantErr: domain.ErrEstablishmentUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase, orderRepo, _, establishmentRepo, tx := newUseCase(t)
			ctx, txCtx := transactionContexts(t)
			expectTransaction(tx, ctx, txCtx)
			orderRepo.EXPECT().FindOrderByIdempotencyKey(txCtx, idempotencyKey).Return(nil, domain.ErrOrderNotFound).Once()
			establishmentRepo.EXPECT().FindEstablishmentById(txCtx, establishmentID).
				Return(tt.establishment, tt.repoErr).Once()

			order, err := useCase.CreateOrder(ctx, establishmentID, idempotencyKey, validItems())

			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, order)
		})
	}
}

func TestCustomerUseCase_CreateOrder_ValidatesProductsBeforeLocking(t *testing.T) {
	deletedAt := time.Now()
	repositoryErr := errors.New("database unavailable")
	deleted := validProduct(productID, domain.Unlimited, nil)
	deleted.DeletedAt = &deletedAt
	foreign := validProduct(productID, domain.Unlimited, nil)
	foreign.EstablishmentID = "establishment-2"
	unavailable := validProduct(productID, domain.Unlimited, nil)
	unavailable.Available = false
	tests := []struct {
		name    string
		product *domain.Product
		repoErr error
		wantErr error
		text    string
	}{
		{name: "repository error", repoErr: repositoryErr, wantErr: repositoryErr},
		{name: "not found", text: "product not found"},
		{name: "deleted", product: deleted, wantErr: domain.ErrProductDeleted},
		{name: "other establishment", product: foreign, wantErr: domain.ErrProductFromOtherEstablishment},
		{name: "unavailable", product: unavailable, wantErr: domain.ErrProductUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase, orderRepo, productRepo, establishmentRepo, tx := newUseCase(t)
			ctx, txCtx := transactionContexts(t)
			expectCreationPrefix(orderRepo, establishmentRepo, tx, ctx, txCtx)
			productRepo.EXPECT().FindProductById(txCtx, productID).Return(tt.product, tt.repoErr).Once()

			order, err := useCase.CreateOrder(ctx, establishmentID, idempotencyKey, validItems())

			require.Error(t, err)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.ErrorContains(t, err, tt.text)
			}
			assert.Nil(t, order)
		})
	}
}

func TestCustomerUseCase_CreateOrder_ValidatesLockedFiniteProduct(t *testing.T) {
	stock := 3
	one := 1
	repositoryErr := errors.New("database unavailable")
	deletedAt := time.Now()
	deleted := validProduct(productID, domain.Finite, &stock)
	deleted.DeletedAt = &deletedAt
	tests := []struct {
		name    string
		locked  *domain.Product
		repoErr error
		wantErr error
		text    string
	}{
		{name: "lock error", repoErr: repositoryErr, wantErr: repositoryErr},
		{name: "disappeared", text: "product not found"},
		{name: "became deleted", locked: deleted, wantErr: domain.ErrProductDeleted},
		{name: "quantity is nil", locked: validProduct(productID, domain.Finite, nil), wantErr: domain.ErrInsufficientStock},
		{name: "insufficient stock", locked: validProduct(productID, domain.Finite, &one), wantErr: domain.ErrInsufficientStock},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase, orderRepo, productRepo, establishmentRepo, tx := newUseCase(t)
			ctx, txCtx := transactionContexts(t)
			expectCreationPrefix(orderRepo, establishmentRepo, tx, ctx, txCtx)
			productRepo.EXPECT().FindProductById(txCtx, productID).
				Return(validProduct(productID, domain.Finite, &stock), nil).Once()
			productRepo.EXPECT().FindProductByIdForUpdate(txCtx, productID).Return(tt.locked, tt.repoErr).Once()

			order, err := useCase.CreateOrder(ctx, establishmentID, idempotencyKey, validItems())

			require.Error(t, err)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.ErrorContains(t, err, tt.text)
			}
			assert.Nil(t, order)
		})
	}
}

func TestCustomerUseCase_CreateOrder_CreatesSnapshotsAndDecrementsFiniteStock(t *testing.T) {
	ctx, txCtx := transactionContexts(t)
	stock := 3
	finite := validProduct(productID, domain.Finite, &stock)
	finite.ExternalID, finite.Name, finite.Price = "external-1", "Baguette", 12.5
	unlimited := validProduct("product-2", domain.Unlimited, nil)
	unlimited.ExternalID, unlimited.Name, unlimited.Price = "external-2", "Coffee", 7.25
	useCase, orderRepo, productRepo, establishmentRepo, tx := newUseCase(t)
	expectCreationPrefix(orderRepo, establishmentRepo, tx, ctx, txCtx)
	// Products are processed in sorted ID order, independently of request order.
	productRepo.EXPECT().FindProductById(txCtx, productID).Return(finite, nil).Once()
	productRepo.EXPECT().FindProductById(txCtx, "product-2").Return(unlimited, nil).Once()
	productRepo.EXPECT().FindProductByIdForUpdate(txCtx, productID).Return(finite, nil).Once()
	productRepo.EXPECT().DecrementProductQuantity(txCtx, productID, 3).Return(nil).Once()
	var inserted domain.Order
	orderRepo.EXPECT().InsertOrder(txCtx, mock.AnythingOfType("domain.Order")).
		RunAndReturn(func(_ context.Context, order domain.Order) (*domain.Order, error) {
			inserted = order
			persisted := order
			persisted.Items = nil
			persisted.TotalPrice = 0
			return &persisted, nil
		}).Once()
	orderRepo.EXPECT().InsertOrderItem(txCtx, mock.AnythingOfType("domain.OrderItem")).
		RunAndReturn(func(_ context.Context, item domain.OrderItem) (*domain.OrderItem, error) { return &item, nil }).Twice()

	before := time.Now()
	created, err := useCase.CreateOrder(ctx, establishmentID, idempotencyKey, []customer.CreateOrderItem{
		{ProductID: "product-2", Quantity: 2},
		{ProductID: productID, Quantity: 1},
		{ProductID: productID, Quantity: 2},
	})
	after := time.Now()

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.NotEmpty(t, inserted.ID)
	assert.NotEmpty(t, inserted.CustomerID)
	assert.Equal(t, inserted.CustomerID, inserted.DeliveryID)
	assert.Equal(t, idempotencyKey, inserted.IdempotencyKey)
	assert.Equal(t, establishmentID, inserted.EstablishmentID)
	assert.Equal(t, domain.OrderCreated, inserted.StatusID)
	assert.Equal(t, float32(52), created.TotalPrice)
	assert.WithinRange(t, inserted.CreatedAt, before, after)
	assert.Equal(t, inserted.CreatedAt, inserted.UpdatedAt)
	require.Len(t, created.Items, 2)
	assert.Equal(t, productID, created.Items[0].ProductID)
	assert.Equal(t, inserted.ID, created.Items[0].OrderID)
	assert.Equal(t, "external-1", created.Items[0].ExternalProductID)
	assert.Equal(t, "Baguette", created.Items[0].ProductName)
	assert.Equal(t, float32(12.5), created.Items[0].UnitPrice)
	assert.Equal(t, 3, created.Items[0].Quantity)
	assert.Equal(t, float32(37.5), created.Items[0].TotalSum)
	assert.Equal(t, "product-2", created.Items[1].ProductID)
	assert.Equal(t, float32(14.5), created.Items[1].TotalSum)
}

func TestCustomerUseCase_CreateOrder_PropagatesPersistenceErrors(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	tests := []struct {
		name      string
		decrement error
		insertErr error
		nilOrder  bool
		itemErr   error
		wantText  string
	}{
		{name: "decrement stock", decrement: repositoryErr, wantText: "decrement product"},
		{name: "insert order", insertErr: repositoryErr, wantText: "insert order"},
		{name: "nil inserted order", nilOrder: true, wantText: "repository returned a nil order"},
		{name: "insert order item", itemErr: repositoryErr, wantText: "insert order item"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stock := 3
			finite := validProduct(productID, domain.Finite, &stock)
			useCase, orderRepo, productRepo, establishmentRepo, tx := newUseCase(t)
			ctx, txCtx := transactionContexts(t)
			expectCreationPrefix(orderRepo, establishmentRepo, tx, ctx, txCtx)
			productRepo.EXPECT().FindProductById(txCtx, productID).Return(finite, nil).Once()
			productRepo.EXPECT().FindProductByIdForUpdate(txCtx, productID).Return(finite, nil).Once()
			productRepo.EXPECT().DecrementProductQuantity(txCtx, productID, 3).Return(tt.decrement).Once()
			if tt.decrement == nil {
				switch {
				case tt.nilOrder:
					orderRepo.EXPECT().InsertOrder(txCtx, mock.Anything).Return(nil, nil).Once()
				case tt.insertErr != nil:
					orderRepo.EXPECT().InsertOrder(txCtx, mock.Anything).Return(nil, tt.insertErr).Once()
				default:
					orderRepo.EXPECT().InsertOrder(txCtx, mock.Anything).
						RunAndReturn(func(_ context.Context, order domain.Order) (*domain.Order, error) { return &order, nil }).Once()
					orderRepo.EXPECT().InsertOrderItem(txCtx, mock.Anything).Return(nil, tt.itemErr).Once()
				}
			}

			order, err := useCase.CreateOrder(ctx, establishmentID, idempotencyKey, validItems())

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantText)
			if tt.decrement != nil || tt.insertErr != nil || tt.itemErr != nil {
				assert.ErrorIs(t, err, repositoryErr)
			}
			assert.Nil(t, order)
		})
	}
}

func TestCustomerUseCase_CreateOrder_PropagatesTransactionError(t *testing.T) {
	repositoryErr := errors.New("begin transaction")
	useCase, _, _, _, tx := newUseCase(t)
	tx.EXPECT().WithTx(mock.Anything, mock.Anything).Return(repositoryErr).Once()

	order, err := useCase.CreateOrder(t.Context(), establishmentID, idempotencyKey, validItems())

	require.ErrorIs(t, err, repositoryErr)
	assert.ErrorContains(t, err, "create order")
	assert.Nil(t, order)
}

func TestCustomerUseCase_FindOrder(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	tests := []struct {
		name    string
		result  *domain.Order
		repoErr error
	}{
		{name: "returns order", result: &domain.Order{ID: orderID}},
		{name: "wraps repository error", repoErr: repositoryErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase, orderRepo, _, _, _ := newUseCase(t)
			orderRepo.EXPECT().FindOrderById(mock.Anything, orderID).Return(tt.result, tt.repoErr).Once()

			actual, err := useCase.FindOrder(t.Context(), orderID)

			if tt.repoErr != nil {
				require.ErrorIs(t, err, repositoryErr)
				assert.Nil(t, actual)
				return
			}
			require.NoError(t, err)
			assert.Same(t, tt.result, actual)
		})
	}
}

func TestCustomerUseCase_CancelOrder(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	tests := []struct {
		name      string
		found     *domain.Order
		findErr   error
		updated   *domain.Order
		updateErr error
		wantErr   error
		update    bool
	}{
		{name: "lookup error", findErr: repositoryErr, wantErr: repositoryErr},
		{name: "missing order", wantErr: domain.ErrOrderNotFound},
		{name: "already cancelled", found: &domain.Order{ID: orderID, StatusID: domain.OrderCanceled}},
		{name: "created order", found: &domain.Order{ID: orderID, StatusID: domain.OrderCreated}, updated: &domain.Order{ID: orderID, StatusID: domain.OrderCanceled}, update: true},
		{name: "update error", found: &domain.Order{ID: orderID, StatusID: domain.OrderCreated}, updateErr: repositoryErr, wantErr: repositoryErr, update: true},
		{name: "accepted order", found: &domain.Order{ID: orderID, StatusID: domain.OrderAccepted}, wantErr: domain.ErrOrderCannotBeCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase, orderRepo, _, _, _ := newUseCase(t)
			orderRepo.EXPECT().FindOrderById(mock.Anything, orderID).Return(tt.found, tt.findErr).Once()
			if tt.update {
				orderRepo.EXPECT().UpdateOrderStatus(mock.Anything, orderID, int(domain.OrderCanceled)).
					Return(tt.updated, tt.updateErr).Once()
			}

			actual, err := useCase.CancelOrder(t.Context(), orderID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, actual)
				return
			}
			require.NoError(t, err)
			if tt.update {
				assert.Same(t, tt.updated, actual)
			} else {
				assert.Same(t, tt.found, actual)
			}
		})
	}
}

func newUseCase(t *testing.T) (
	*customer.CustomerUseCase,
	*mocks.MockOrderRepository,
	*mocks.MockProductRepository,
	*mocks.MockEstablishmentRepository,
	*mocks.MockTransactor,
) {
	t.Helper()
	orderRepo := mocks.NewMockOrderRepository(t)
	productRepo := mocks.NewMockProductRepository(t)
	establishmentRepo := mocks.NewMockEstablishmentRepository(t)
	tx := mocks.NewMockTransactor(t)
	return customer.NewCustomerUseCase(orderRepo, productRepo, establishmentRepo, tx),
		orderRepo, productRepo, establishmentRepo, tx
}

func transactionContexts(t *testing.T) (context.Context, context.Context) {
	t.Helper()
	type transactionKey struct{}
	ctx := t.Context()
	return ctx, context.WithValue(ctx, transactionKey{}, "transaction")
}

func expectTransaction(tx *mocks.MockTransactor, ctx, txCtx context.Context) {
	tx.EXPECT().WithTx(ctx, mock.Anything).
		RunAndReturn(func(_ context.Context, callback func(context.Context) error) error { return callback(txCtx) }).Once()
}

func expectCreationPrefix(
	orderRepo *mocks.MockOrderRepository,
	establishmentRepo *mocks.MockEstablishmentRepository,
	tx *mocks.MockTransactor,
	ctx, txCtx context.Context,
) {
	expectTransaction(tx, ctx, txCtx)
	orderRepo.EXPECT().FindOrderByIdempotencyKey(txCtx, idempotencyKey).Return(nil, domain.ErrOrderNotFound).Once()
	establishmentRepo.EXPECT().FindEstablishmentById(txCtx, establishmentID).
		Return(&domain.Establishment{ID: establishmentID, Available: true}, nil).Once()
}

func validItems() []customer.CreateOrderItem {
	return []customer.CreateOrderItem{{ProductID: productID, Quantity: 3}}
}

func validProduct(id string, inventoryType domain.InventoryType, quantity *int) *domain.Product {
	return &domain.Product{
		ID:              id,
		EstablishmentID: establishmentID,
		Available:       true,
		InventoryTypeID: inventoryType,
		Quantity:        quantity,
		Price:           1,
	}
}
