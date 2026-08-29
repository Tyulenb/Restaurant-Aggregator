package main

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type simulator struct {
	client  *client
	cfg     config
	logger  *slog.Logger
	mu      sync.Mutex
	working map[string]struct{}
	wg      sync.WaitGroup
}

func newSimulator(cfg config, logger *slog.Logger, client *client) *simulator {
	return &simulator{
		client:  client,
		cfg:     cfg,
		logger:  logger,
		working: make(map[string]struct{}),
	}
}

func (s *simulator) run(ctx context.Context) error {
	for {
		err := s.bootstrap(ctx)
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		s.logger.Warn("kitchen is not ready; retrying", "error", err, "after", s.cfg.retryInterval)
		if err := wait(ctx, s.cfg.retryInterval); err != nil {
			return err
		}
	}

	s.logger.Info("demo establishment is ready", "rejection_percent", s.cfg.rejectionPercent)
	ticker := time.NewTicker(s.cfg.pollInterval)
	defer ticker.Stop()

	for {
		if err := s.poll(ctx); err != nil && ctx.Err() == nil {
			s.logger.Error("poll orders", "error", err)
		}
		select {
		case <-ctx.Done():
			s.wg.Wait()
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *simulator) markWorking(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.working[id]; ok {
		return false
	}
	s.working[id] = struct{}{}
	return true
}

func (s *simulator) unmarkWorking(id string) {
	s.mu.Lock()
	delete(s.working, id)
	s.mu.Unlock()
}

func wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
