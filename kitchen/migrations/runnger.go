package migrations

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed schemes/*.sql
var embedMigrations embed.FS

func SetupPostgres(ctx context.Context, pool *pgxpool.Pool) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set postgres dialect in goose: %w", err)
	}

	goose.SetTableName("onboarding_goose_db_version")

	db := stdlib.OpenDBFromPool(pool)
	defer func() {
		_ = db.Close()
	}()

	if err := goose.UpContext(ctx, db, "schemes"); err != nil {
		return fmt.Errorf("failed to setup goose migrations: %w", err)
	}

	return nil
}
