package main

import (
	"context"
	"fmt"
	"net/http"
)

type category struct {
	ExternalID string `json:"external_id"`
}

type categoryList struct {
	Items []category `json:"items"`
}

type product struct {
	ExternalID string `json:"external_id"`
}

type productList struct {
	Items []product `json:"items"`
}

type categorySeed struct {
	ExternalID  string `json:"external_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type productSeed struct {
	ExternalID         string `json:"external_id"`
	CategoryExternalID string `json:"category_external_id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	Price              int    `json:"price"`
	Available          bool   `json:"available"`
	InventoryType      string `json:"inventory_type"`
	Quantity           *int   `json:"quantity,omitempty"`
}

var menu = struct {
	categories []categorySeed
	products   []productSeed
}{
	categories: []categorySeed{
		{ExternalID: "pizza", Name: "Pizza", Description: "Freshly baked pizza"},
		{ExternalID: "drinks", Name: "Drinks", Description: "Cold drinks"},
	},
	products: []productSeed{
		{ExternalID: "margherita", CategoryExternalID: "pizza", Name: "Margherita", Description: "Tomato, mozzarella and basil", Price: 590, Available: true, InventoryType: "untracked"},
		{ExternalID: "pepperoni", CategoryExternalID: "pizza", Name: "Pepperoni", Description: "Pepperoni and mozzarella", Price: 690, Available: true, InventoryType: "untracked"},
		{ExternalID: "lemonade", CategoryExternalID: "drinks", Name: "Lemonade", Description: "House lemonade", Price: 190, Available: true, InventoryType: "finite", Quantity: intPointer(50)},
	},
}

func (s *simulator) bootstrap(ctx context.Context) error {
	if err := s.client.do(ctx, http.MethodGet, "/api/partner/v1/establishment", nil, nil); err != nil {
		return fmt.Errorf("get establishment: %w", err)
	}
	if err := s.ensureCategories(ctx); err != nil {
		return err
	}
	if err := s.ensureProducts(ctx); err != nil {
		return err
	}
	if err := s.client.do(ctx, http.MethodPatch, "/api/partner/v1/establishment/availability", map[string]bool{"available": true}, nil); err != nil {
		return fmt.Errorf("enable establishment: %w", err)
	}
	return nil
}

func (s *simulator) ensureCategories(ctx context.Context) error {
	var existing categoryList
	if err := s.client.do(ctx, http.MethodGet, "/api/partner/v1/categories", nil, &existing); err != nil {
		return fmt.Errorf("list categories: %w", err)
	}
	present := make(map[string]bool, len(existing.Items))
	for _, item := range existing.Items {
		present[item.ExternalID] = true
	}
	for _, item := range menu.categories {
		if present[item.ExternalID] {
			continue
		}
		if err := s.client.do(ctx, http.MethodPost, "/api/partner/v1/categories", item, nil); err != nil {
			return fmt.Errorf("create category %q: %w", item.ExternalID, err)
		}
		s.logger.Info("created category", "external_id", item.ExternalID)
	}
	return nil
}

func (s *simulator) ensureProducts(ctx context.Context) error {
	var existing productList
	if err := s.client.do(ctx, http.MethodGet, "/api/partner/v1/products", nil, &existing); err != nil {
		return fmt.Errorf("list products: %w", err)
	}
	present := make(map[string]bool, len(existing.Items))
	for _, item := range existing.Items {
		present[item.ExternalID] = true
	}
	for _, item := range menu.products {
		if present[item.ExternalID] {
			continue
		}
		if err := s.client.do(ctx, http.MethodPost, "/api/partner/v1/products", item, nil); err != nil {
			return fmt.Errorf("create product %q: %w", item.ExternalID, err)
		}
		s.logger.Info("created product", "external_id", item.ExternalID)
	}
	return nil
}

func intPointer(value int) *int { return &value }
