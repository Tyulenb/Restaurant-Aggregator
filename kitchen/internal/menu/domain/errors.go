package domain

import "errors"

var (
	ErrCategoryNotFound           = errors.New("category not found")
	ErrCategoryConflict           = errors.New("category external ID already exists")
	ErrCategoryValidation         = errors.New("category validation failed")
	ErrProductNotFound            = errors.New("product not found")
	ErrProductConflict            = errors.New("product external ID already exists")
	ErrProductValidation          = errors.New("product validation failed")
	ErrStockAdjustmentConflict    = errors.New("stock adjustment cannot be applied")
	ErrProductIdempotencyConflict = errors.New("product idempotency key conflict")
)
