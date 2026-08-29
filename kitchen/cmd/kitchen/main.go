package main

import (
	"log/slog"
	"os"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/app"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/config"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}

	if err = app.NewApi(cfg).Run(); err != nil {
		slog.Error("run application", "error", err)
		os.Exit(1)
	}
}
