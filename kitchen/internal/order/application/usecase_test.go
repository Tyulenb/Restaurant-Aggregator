package application

import (
	"context"
	"errors"
	"testing"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type orderRepositoryStub struct {
	byID             *domain.Order
	byIdempotencyKey *domain.Order
	inserted         *domain.Order
	updatedStatus    int
}

func (s *orderRepositoryStub) InsertOrder(_ context.Context, order domain.Order) (*domain.Order, error) {
	s.inserted = &order
	return &order, nil
}

func (s *orderRepositoryStub) FindOrderById(context.Context, string) (*domain.Order, error) {
	return s.byID, nil
}

func (s *orderRepositoryStub) FindOrderByIdempotencyKey(context.Context, string) (*domain.Order, error) {
	return s.byIdempotencyKey, nil
}

func (s *orderRepositoryStub) UpdateOrderStatus(_ context.Context, _ string, status int) (*domain.Order, error) {
	s.updatedStatus = status
	return &domain.Order{ID: s.byID.ID, Status: cancelledOrderStatus}, nil
}

type productRepositoryStub struct {
	products map[string]*domain.Product
	calls    map[string]int
}

func (s *productRepositoryStub) FindProductById(_ context.Context, productID string) (*domain.Product, error) {
	s.calls[productID]++
	return s.products[productID], nil
}

func TestCustomerUseCase_CreateOrder(t *testing.T) {
	t.Parallel()

	stock := 3
	orderRepo := &orderRepositoryStub{}
	productRepo := &productRepositoryStub{
		products: map[string]*domain.Product{
			"product-1": {
				ID:              "product-1",
				ExternalID:      "external-1",
				EstablishmentID: "establishment-1",
				Name:            "Baguette",
				Available:       true,
				InventoryTypeID: finiteInventoryTypeID,
				Quantity:        &stock,
				Price:           125,
			},
		},
		calls: make(map[string]int),
	}
	useCase := NewCustomerUseCase(orderRepo, productRepo)

	order, err := useCase.CreateOrder(context.Background(), "establishment-1", "key-1", []domain.CreateOrderItem{
		{ProductID: "product-1", Quantity: 1},
		{ProductID: "product-1", Quantity: 2},
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if orderRepo.inserted == nil {
		t.Fatal("CreateOrder() did not insert the order")
	}
	if order.Status != createdOrderStatus {
		t.Errorf("status = %q, want %q", order.Status, createdOrderStatus)
	}
	if order.TotalPrice != 375 {
		t.Errorf("total price = %v, want 375", order.TotalPrice)
	}
	if len(order.Items) != 1 {
		t.Fatalf("item count = %d, want 2", len(order.Items))
	}
	if order.Items[0].ProductName != "Baguette" || order.Items[0].UnitPrice != 125 {
		t.Errorf("item does not contain the product snapshot: %+v", order.Items[0])
	}
	if order.CustomerID == "" || order.CustomerID != order.DeliveryID {
		t.Errorf("customer and delivery placeholders = %q and %q", order.CustomerID, order.DeliveryID)
	}
	if productRepo.calls["product-1"] != 1 {
		t.Errorf("product lookup calls = %d, want 1", productRepo.calls["product-1"])
	}
}

func TestCustomerUseCase_CreateOrderRejectsAggregatedInsufficientStock(t *testing.T) {
	t.Parallel()

	stock := 2
	orderRepo := &orderRepositoryStub{}
	productRepo := &productRepositoryStub{
		products: map[string]*domain.Product{
			"product-1": {
				ID:              "product-1",
				EstablishmentID: "establishment-1",
				Available:       true,
				InventoryTypeID: finiteInventoryTypeID,
				Quantity:        &stock,
			},
		},
		calls: make(map[string]int),
	}

	_, err := NewCustomerUseCase(orderRepo, productRepo).CreateOrder(
		context.Background(),
		"establishment-1",
		"key-1",
		[]domain.CreateOrderItem{
			{ProductID: "product-1", Quantity: 1},
			{ProductID: "product-1", Quantity: 2},
		},
	)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("CreateOrder() error = %v, want ErrInsufficientStock", err)
	}
	if orderRepo.inserted != nil {
		t.Error("CreateOrder() inserted an invalid order")
	}
}

func TestCustomerUseCase_CreateOrderIsIdempotent(t *testing.T) {
	t.Parallel()

	existing := &domain.Order{
		ID:              "order-1",
		EstablishmentID: "establishment-1",
		Items: []domain.OrderItem{
			{ProductID: "product-1", Quantity: 2},
		},
	}
	orderRepo := &orderRepositoryStub{byIdempotencyKey: existing}
	productRepo := &productRepositoryStub{calls: make(map[string]int)}
	useCase := NewCustomerUseCase(orderRepo, productRepo)

	order, err := useCase.CreateOrder(context.Background(), "establishment-1", "key-1", []domain.CreateOrderItem{
		{ProductID: "product-1", Quantity: 2},
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if order != existing {
		t.Error("CreateOrder() did not return the existing order")
	}
	if orderRepo.inserted != nil || len(productRepo.calls) != 0 {
		t.Error("idempotent replay performed creation work")
	}

	_, err = useCase.CreateOrder(context.Background(), "establishment-1", "key-1", []domain.CreateOrderItem{
		{ProductID: "product-1", Quantity: 3},
	})
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("CreateOrder() error = %v, want ErrIdempotencyConflict", err)
	}
}

func TestCustomerUseCase_CancelOrder(t *testing.T) {
	t.Parallel()

	t.Run("created order", func(t *testing.T) {
		orderRepo := &orderRepositoryStub{byID: &domain.Order{ID: "order-1", Status: createdOrderStatus}}

		order, err := NewCustomerUseCase(orderRepo, nil).CancelOrder(context.Background(), "order-1")
		if err != nil {
			t.Fatalf("CancelOrder() error = %v", err)
		}
		if order.Status != cancelledOrderStatus || orderRepo.updatedStatus != cancelledOrderStatusID {
			t.Errorf("cancelled order = %+v, updated status ID = %d", order, orderRepo.updatedStatus)
		}
	})

	t.Run("already cancelled order", func(t *testing.T) {
		existing := &domain.Order{ID: "order-1", Status: cancelledOrderStatus}
		orderRepo := &orderRepositoryStub{byID: existing}

		order, err := NewCustomerUseCase(orderRepo, nil).CancelOrder(context.Background(), "order-1")
		if err != nil {
			t.Fatalf("CancelOrder() error = %v", err)
		}
		if order != existing || orderRepo.updatedStatus != 0 {
			t.Error("cancelling an already cancelled order was not idempotent")
		}
	})

	t.Run("accepted order", func(t *testing.T) {
		orderRepo := &orderRepositoryStub{byID: &domain.Order{ID: "order-1", Status: "accepted"}}

		_, err := NewCustomerUseCase(orderRepo, nil).CancelOrder(context.Background(), "order-1")
		if !errors.Is(err, ErrOrderCannotBeCancelled) {
			t.Fatalf("CancelOrder() error = %v, want ErrOrderCannotBeCancelled", err)
		}
	})
}
