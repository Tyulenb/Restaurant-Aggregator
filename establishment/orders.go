package main

import (
	"context"
	"math/rand/v2"
	"net/http"
	"strconv"
)

const (
	statusCreated   = 1
	statusPreparing = 3
	statusReady     = 4
	statusCompleted = 5
)

type order struct {
	ID       string `json:"id"`
	StatusID int    `json:"status_id"`
}

type orderList struct {
	Items []order `json:"items"`
}

func (s *simulator) poll(ctx context.Context) error {
	var orders orderList
	if err := s.client.do(ctx, http.MethodGet, "/api/partner/v1/orders?status="+strconv.Itoa(statusCreated), nil, &orders); err != nil {
		return err
	}
	for _, item := range orders.Items {
		if !s.markWorking(item.ID) {
			continue
		}
		s.wg.Add(1)
		go func(orderID string) {
			defer s.wg.Done()
			defer s.unmarkWorking(orderID)
			s.process(ctx, orderID)
		}(item.ID)
	}
	return nil
}

func (s *simulator) process(ctx context.Context, orderID string) {
	if rand.IntN(100) < s.cfg.rejectionPercent {
		err := s.client.do(ctx, http.MethodPost, "/api/partner/v1/orders/"+orderID+"/reject", map[string]string{"reason": "DEMO_RANDOM_REJECTION"}, nil)
		if err != nil {
			s.logger.Error("reject order", "order_id", orderID, "error", err)
			return
		}
		s.logger.Info("order rejected", "order_id", orderID)
		return
	}

	if err := s.client.do(ctx, http.MethodPost, "/api/partner/v1/orders/"+orderID+"/accept", nil, nil); err != nil {
		s.logger.Error("accept order", "order_id", orderID, "error", err)
		return
	}
	s.logger.Info("order accepted", "order_id", orderID)

	for _, status := range []int{statusPreparing, statusReady, statusCompleted} {
		if err := wait(ctx, s.cfg.transitionDelay); err != nil {
			return
		}
		if err := s.client.do(ctx, http.MethodPatch, "/api/partner/v1/orders/"+orderID+"/status", map[string]int{"status_id": status}, nil); err != nil {
			s.logger.Error("advance order", "order_id", orderID, "status_id", status, "error", err)
			return
		}
		s.logger.Info("order status changed", "order_id", orderID, "status_id", status)
	}
}
