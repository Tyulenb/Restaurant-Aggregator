// Command establishment runs a small partner-system simulator for local demos.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := newSimulator(
		cfg,
		logger,
		&client{
			baseURL: strings.TrimRight(cfg.baseURL, "/"),
			apiKey:  cfg.apiKey,
			http:    &http.Client{Timeout: cfg.requestTimeout},
		},
	)

	if err := s.run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("simulator stopped", "error", err)
		os.Exit(1)
	}
	logger.Info("simulator stopped")
}
