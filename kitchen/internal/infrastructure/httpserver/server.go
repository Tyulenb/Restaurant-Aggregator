package infra_httpserver

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/config"
)

type Handler interface {
	RegisterRoutes(router *http.ServeMux)
}

func ServerRun(cfg *config.Config, handlers ...Handler) error {
	router := http.NewServeMux()

	for _, handler := range handlers {
		handler.RegisterRoutes(router)
	}

	srv := &http.Server{
		Addr:         cfg.ListenAddress,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-srvErr:
		return err
	case <-ctx.Done():
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownGrace)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
