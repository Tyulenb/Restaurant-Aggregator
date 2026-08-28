package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application/mocks"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

const (
	establishmentID = "establishment-1"
	idempotencyKey  = "idempotency-key"
	productID       = "product-1"
)

func TestCustomerUseCase_CreateOrder_ValidatesRequestBeforeRepositories(t *testing.T) {
	tests := []struct {
		name  string
		items []domain.CreateOrderItem
		err   error
	}{
		{name: "empty order", err: application.ErrEmptyOrder},
		{
			name:  "zero quantity",
			items: []domain.CreateOrderItem{{ProductID: productID, Quantity: 0}},
			err:   application.ErrInvalidItemQuantity,
		},
		{
			name:  "negative quantity",
			items: []domain.CreateOrderItem{{ProductID: productID, Quantity: -1}},
			err:   application.ErrInvalidItemQuantity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := mocks.NewMockOrderRepository(t)
			productRepo := mocks.NewMockProductRepository(t)
			useCase := application.NewCustomerUseCase(orderRepo, productRepo)

			order, err := useCase.CreateOrder(t.Context(), establishmentID, idempotencyKey, tt.items)

			require.ErrorIs(t, err, tt.err)
			assert.Nil(t, order)
		})
	}
}

func TestCustomerUseCase_CreateOrder_ReturnsExistingOrderForSameIdempotentRequest(t *testing.T) {
	ctx := t.Context()
	existing := &domain.Order{
		ID:              "order-1",
		EstablishmentID: establishmentID,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 1},
			{ProductID: productID, Quantity: 2},
			{ProductID: "product-2", Quantity: 1},
		},
	}
	orderRepo := mocks.NewMockOrderRepository(t)
	productRepo := mocks.NewMockProductRepository(t)
	orderRepo.EXPECT().FindOrderByIdempotencyKey(ctx, idempotencyKey).Return(existing, nil).Once()
	useCase := application.NewCustomerUseCase(orderRepo, productRepo)

	order, err := useCase.CreateOrder(ctx, establishmentID, idempotencyKey, []domain.CreateOrderItem{
		{ProductID: productID, Quantity: 2},
		{ProductID: "product-2", Quantity: 1},
		{ProductID: productID, Quantity: 1},
	})

	require.NoError(t, err)
	assert.Same(t, existing, order)
}

func TestCustomerUseCase_CreateOrder_RejectsDifferentRequestForIdempotencyKey(t *testing.T) {
	tests := []struct {
		name            string
		establishmentID string
		items           []domain.CreateOrderItem
	}{
		{
			name:            "different establishment",
			establishmentID: "establishment-2",
			items:           []domain.CreateOrderItem{{ProductID: productID, Quantity: 2}},
		},
		{
			name:            "different quantity",
			establishmentID: establishmentID,
			items:           []domain.CreateOrderItem{{ProductID: productID, Quantity: 3}},
		},
		{
			name:            "different products",
			establishmentID: establishmentID,
			items:           []domain.CreateOrderItem{{ProductID: "product-2", Quantity: 2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := &domain.Order{
				EstablishmentID: establishmentID,
				Items:           []domain.OrderItem{{ProductID: productID, Quantity: 2}},
			}
			orderRepo := mocks.NewMockOrderRepository(t)
			productRepo := mocks.NewMockProductRepository(t)
			orderRepo.EXPECT().FindOrderByIdempotencyKey(mock.Anything, idempotencyKey).Return(existing, nil).Once()
			useCase := application.NewCustomerUseCase(orderRepo, productRepo)

			order, err := useCase.CreateOrder(t.Context(), tt.establishmentID, idempotencyKey, tt.items)

			require.ErrorIs(t, err, application.ErrIdempotencyConflict)
			assert.Nil(t, order)
		})
	}
}

func TestCustomerUseCase_CreateOrder_ValidatesProducts(t *testing.T) {
	deletedAt := time.Now()
	one := 1
	repositoryErr := errors.New("database unavailable")

	tests := []struct {
		name       string
		product    *domain.Product
		productErr error
		wantErr    error
	}{
		{name: "repository error", productErr: repositoryErr, wantErr: repositoryErr},
		{name: "nil product", wantErr: errors.New("product not found")},
		{
			name:    "deleted product",
			product: validProduct(productID, domain.Unlimited, nil),
			wantErr: application.ErrProductDeleted,
		},
		{
			name:    "product from another establishment",
			product: validProduct(productID, domain.Unlimited, nil),
			wantErr: application.ErrProductFromOtherEstablishment,
		},
		{
			name:    "unavailable product",
			product: validProduct(productID, domain.Unlimited, nil),
			wantErr: application.ErrProductUnavailable,
		},
		{
			name:    "finite product without quantity",
			product: validProduct(productID, domain.Finite, nil),
			wantErr: application.ErrInsufficientStock,
		},
		{
			name:    "finite product with insufficient stock",
			product: validProduct(productID, domain.Finite, &one),
			wantErr: application.ErrInsufficientStock,
		},
	}
	tests[2].product.DeletedAt = &deletedAt
	tests[3].product.EstablishmentID = "establishment-2"
	tests[4].product.Available = false
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := mocks.NewMockOrderRepository(t)
			productRepo := mocks.NewMockProductRepository(t)
			orderRepo.EXPECT().FindOrderByIdempotencyKey(mock.Anything, idempotencyKey).
				Return(nil, application.ErrOrderNotFound).Once()
			productRepo.EXPECT().FindProductById(mock.Anything, productID).Return(tt.product, tt.productErr).Once()
			useCase := application.NewCustomerUseCase(orderRepo, productRepo)

			order, err := useCase.CreateOrder(t.Context(), establishmentID, idempotencyKey, []domain.CreateOrderItem{
				{ProductID: productID, Quantity: 2},
			})

			require.Error(t, err)
			if tt.name == "nil product" {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.Nil(t, order)
		})
	}
}

func TestCustomerUseCase_CreateOrder_CreatesPricedProductSnapshots(t *testing.T) {
	ctx := t.Context()
	stock := 3
	finiteProduct := validProduct(productID, domain.Finite, &stock)
	finiteProduct.ExternalID = "external-1"
	finiteProduct.Name = "Baguette"
	finiteProduct.Price = 12.5
	unlimitedProduct := validProduct("product-2", domain.Unlimited, nil)
	unlimitedProduct.ExternalID = "external-2"
	unlimitedProduct.Name = "Coffee"
	unlimitedProduct.Price = 7.25

	orderRepo := mocks.NewMockOrderRepository(t)
	productRepo := mocks.NewMockProductRepository(t)
	orderRepo.EXPECT().FindOrderByIdempotencyKey(ctx, idempotencyKey).
		Return(nil, application.ErrOrderNotFound).Once()
	productRepo.EXPECT().FindProductById(ctx, productID).Return(finiteProduct, nil).Once()
	productRepo.EXPECT().FindProductById(ctx, "product-2").Return(unlimitedProduct, nil).Once()
	var inserted domain.Order
	orderRepo.EXPECT().InsertOrder(ctx, mock.AnythingOfType("domain.Order")).
		Run(func(_ context.Context, order domain.Order) { inserted = order }).
		RunAndReturn(func(_ context.Context, order domain.Order) (*domain.Order, error) { return &order, nil }).Once()
	useCase := application.NewCustomerUseCase(orderRepo, productRepo)

	before := time.Now()
	created, err := useCase.CreateOrder(ctx, establishmentID, idempotencyKey, []domain.CreateOrderItem{
		{ProductID: productID, Quantity: 1},
		{ProductID: "product-2", Quantity: 2},
		{ProductID: productID, Quantity: 2},
	})
	after := time.Now()

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, inserted, *created)
	assert.NotEmpty(t, inserted.ID)
	assert.NotEmpty(t, inserted.CustomerID)
	assert.Equal(t, inserted.CustomerID, inserted.DeliveryID)
	assert.Equal(t, idempotencyKey, inserted.IdempotencyKey)
	assert.Equal(t, establishmentID, inserted.EstablishmentID)
	assert.Equal(t, domain.OrderCreated, inserted.StatusID)
	assert.Equal(t, float32(52), inserted.TotalPrice)
	assert.WithinRange(t, inserted.CreatedAt, before, after)
	assert.Equal(t, inserted.CreatedAt, inserted.UpdatedAt)
	require.Len(t, inserted.Items, 2)

	items := make(map[string]domain.OrderItem, len(inserted.Items))
	for _, item := range inserted.Items {
		assert.NotEmpty(t, item.ID)
		items[item.ProductID] = item
	}
	assert.Equal(t, domain.OrderItem{
		ID:                items[productID].ID,
		ProductID:         productID,
		ExternalProductID: "external-1",
		ProductName:       "Baguette",
		UnitPrice:         12.5,
		Quantity:          3,
		TotalSum:          37.5,
	}, items[productID])
	assert.Equal(t, domain.OrderItem{
		ID:                items["product-2"].ID,
		ProductID:         "product-2",
		ExternalProductID: "external-2",
		ProductName:       "Coffee",
		UnitPrice:         7.25,
		Quantity:          2,
		TotalSum:          14.5,
	}, items["product-2"])
}

func TestCustomerUseCase_CreateOrder_PropagatesOrderRepositoryErrors(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	t.Run("idempotency lookup", func(t *testing.T) {
		orderRepo := mocks.NewMockOrderRepository(t)
		productRepo := mocks.NewMockProductRepository(t)
		orderRepo.EXPECT().FindOrderByIdempotencyKey(mock.Anything, idempotencyKey).
			Return(nil, repositoryErr).Once()
		useCase := application.NewCustomerUseCase(orderRepo, productRepo)

		order, err := useCase.CreateOrder(t.Context(), establishmentID, idempotencyKey,
			[]domain.CreateOrderItem{{ProductID: productID, Quantity: 1}})

		require.ErrorIs(t, err, repositoryErr)
		assert.ErrorContains(t, err, "find order by idempotency key")
		assert.Nil(t, order)
	})

	t.Run("insert", func(t *testing.T) {
		orderRepo := mocks.NewMockOrderRepository(t)
		productRepo := mocks.NewMockProductRepository(t)
		orderRepo.EXPECT().FindOrderByIdempotencyKey(mock.Anything, idempotencyKey).
			Return(nil, application.ErrOrderNotFound).Once()
		productRepo.EXPECT().FindProductById(mock.Anything, productID).
			Return(validProduct(productID, domain.Unlimited, nil), nil).Once()
		orderRepo.EXPECT().InsertOrder(mock.Anything, mock.Anything).Return(nil, repositoryErr).Once()
		useCase := application.NewCustomerUseCase(orderRepo, productRepo)

		order, err := useCase.CreateOrder(t.Context(), establishmentID, idempotencyKey,
			[]domain.CreateOrderItem{{ProductID: productID, Quantity: 1}})

		require.ErrorIs(t, err, repositoryErr)
		assert.ErrorContains(t, err, "insert order")
		assert.Nil(t, order)
	})
}

func TestCustomerUseCase_FindOrder(t *testing.T) {
	t.Run("returns order", func(t *testing.T) {
		expected := &domain.Order{ID: "order-1"}
		orderRepo := mocks.NewMockOrderRepository(t)
		orderRepo.EXPECT().FindOrderById(mock.Anything, expected.ID).Return(expected, nil).Once()
		useCase := application.NewCustomerUseCase(orderRepo, mocks.NewMockProductRepository(t))

		actual, err := useCase.FindOrder(t.Context(), expected.ID)

		require.NoError(t, err)
		assert.Same(t, expected, actual)
	})

	t.Run("wraps repository error", func(t *testing.T) {
		repositoryErr := errors.New("database unavailable")
		orderRepo := mocks.NewMockOrderRepository(t)
		orderRepo.EXPECT().FindOrderById(mock.Anything, "order-1").Return(nil, repositoryErr).Once()
		useCase := application.NewCustomerUseCase(orderRepo, mocks.NewMockProductRepository(t))

		order, err := useCase.FindOrder(t.Context(), "order-1")

		require.ErrorIs(t, err, repositoryErr)
		assert.ErrorContains(t, err, `find order "order-1"`)
		assert.Nil(t, order)
	})
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
		{name: "missing order", wantErr: application.ErrOrderNotFound},
		{
			name:  "already cancelled is idempotent",
			found: &domain.Order{ID: "order-1", StatusID: domain.OrderCanceled},
		},
		{
			name:    "created order is cancelled",
			found:   &domain.Order{ID: "order-1", StatusID: domain.OrderCreated},
			updated: &domain.Order{ID: "order-1", StatusID: domain.OrderCanceled},
			update:  true,
		},
		{
			name:      "update error",
			found:     &domain.Order{ID: "order-1", StatusID: domain.OrderCreated},
			updateErr: repositoryErr,
			wantErr:   repositoryErr,
			update:    true,
		},
		{
			name:    "accepted order cannot be cancelled",
			found:   &domain.Order{ID: "order-1", StatusID: domain.OrderAccepted},
			wantErr: application.ErrOrderCannotBeCancelled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := mocks.NewMockOrderRepository(t)
			productRepo := mocks.NewMockProductRepository(t)
			orderRepo.EXPECT().FindOrderById(mock.Anything, "order-1").Return(tt.found, tt.findErr).Once()
			if tt.update {
				orderRepo.EXPECT().UpdateOrderStatus(mock.Anything, "order-1", int(domain.OrderCanceled)).
					Return(tt.updated, tt.updateErr).Once()
			}
			useCase := application.NewCustomerUseCase(orderRepo, productRepo)

			order, err := useCase.CancelOrder(t.Context(), "order-1")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, order)
				return
			}
			require.NoError(t, err)
			if tt.update {
				assert.Same(t, tt.updated, order)
			} else {
				assert.Same(t, tt.found, order)
			}
		})
	}
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
