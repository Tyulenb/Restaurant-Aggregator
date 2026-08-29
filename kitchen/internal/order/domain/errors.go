package domain

import "errors"

var (
	ErrOrderNotFound                 = errors.New("order not found")
	ErrEstablishmentNotFound         = errors.New("establishment not found")
	ErrEstablishmentUnavailable      = errors.New("establishment is unavailable")
	ErrEmptyOrder                    = errors.New("order must contain at least one item")
	ErrInvalidItemQuantity           = errors.New("order item quantity must be positive")
	ErrProductDeleted                = errors.New("product is deleted")
	ErrProductFromOtherEstablishment = errors.New("product belongs to another establishment")
	ErrProductUnavailable            = errors.New("product is unavailable")
	ErrInsufficientStock             = errors.New("insufficient product stock")
	ErrIdempotencyConflict           = errors.New("idempotency key was already used for another request")
	ErrOrderCannotBeCancelled        = errors.New("order can be cancelled only before it is accepted")
	ErrInvalidOrderTransition        = errors.New("requested order state transition is not allowed")
)
