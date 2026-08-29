package app

import (
	"context"
	"fmt"
	"log/slog"

	establishmenthttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/adapter/http"
	establishmentpostgres "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/adapter/postgres"
	establishmentapp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/establishment/application"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/config"
	infrahttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/http"
	infrapostgres "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/postgres"
	categoryhttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/adapter/http/category"
	menuestablishmenthttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/adapter/http/establishment"
	producthttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/adapter/http/product"
	categorypostgres "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/adapter/postgres/category"
	menuestablishmentpostgres "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/adapter/postgres/establishment"
	productpostgres "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/adapter/postgres/product"
	categoryapp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/category"
	menuestablishmentapp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/establishment"
	productapp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/product"
	ordercustomerhttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/adapter/http/customer"
	orderpartnerhttp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/adapter/http/partner"
	orderestablishmentpostgres "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/adapter/postgres/establishment"
	orderpostgres "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/adapter/postgres/order"
	orderproductpostgres "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/adapter/postgres/product"
	ordercustomerapp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application/customer"
	orderpartnerapp "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application/partner"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/migrations"
)

type Api struct {
	cfg *config.Config
}

func NewApi(cfg *config.Config) *Api {
	return &Api{
		cfg: cfg,
	}
}

func (a *Api) Run() error {
	ctx := context.Background()
	pool, err := infrapostgres.NewPool(ctx, a.cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("initialize postgres: %w", err)
	}
	defer pool.Close()

	if err = migrations.SetupPostgres(ctx, pool); err != nil {
		return fmt.Errorf("run database migrations: %w", err)
	}

	logger := slog.Default()
	transactor := infrapostgres.NewTransactor(pool)
	authMiddleware := infrahttp.EstablishmentAuthMiddleware(
		a.cfg.PartnerAPIKey,
		a.cfg.PartnerEstablishmentID,
	)

	establishmentRepository := establishmentpostgres.NewEstablishmentRepository(pool)
	establishmentUseCase := establishmentapp.NewUseCase(establishmentRepository)
	establishmentHandler := establishmenthttp.NewCustomerHandler(logger, establishmentUseCase, authMiddleware)

	categoryRepository := categorypostgres.NewCategoryRepository(pool)
	productRepository := productpostgres.NewProductRepository(pool)
	categoryUseCase := categoryapp.NewCategoryUseCase(categoryRepository, productRepository, transactor)
	productUseCase := productapp.NewProductUseCase(productRepository, categoryRepository, transactor)
	categoryHandler := categoryhttp.NewCategoryHandler(logger, categoryUseCase, authMiddleware)
	productHandler := producthttp.NewProductHandler(logger, productUseCase, authMiddleware)
	menuEstablishmentRepository := menuestablishmentpostgres.NewEstablishmentRepository(pool)
	menuUseCase := menuestablishmentapp.NewEstablishmentUseCase(
		productRepository, categoryRepository, menuEstablishmentRepository,
	)
	menuHandler := menuestablishmenthttp.NewEstablishmentHandler(logger, menuUseCase)

	orderRepository := orderpostgres.NewOrderRepository(pool)
	orderProductRepository := orderproductpostgres.NewProductRepository(pool)
	orderEstablishmentRepository := orderestablishmentpostgres.NewEstablishmentRepository(pool)
	customerOrderUseCase := ordercustomerapp.NewCustomerUseCase(
		orderRepository,
		orderProductRepository,
		orderEstablishmentRepository,
		transactor,
	)
	partnerOrderUseCase := orderpartnerapp.NewPartnerUseCase(orderRepository, transactor, orderProductRepository)
	customerOrderHandler := ordercustomerhttp.NewOrderHandler(logger, customerOrderUseCase)
	partnerOrderHandler := orderpartnerhttp.NewPartnerHandler(logger, partnerOrderUseCase, authMiddleware)

	if err = infrahttp.ServerRun(
		a.cfg,
		establishmentHandler,
		menuHandler,
		categoryHandler,
		productHandler,
		customerOrderHandler,
		partnerOrderHandler,
	); err != nil {
		return fmt.Errorf("run HTTP server: %w", err)
	}

	return nil
}
