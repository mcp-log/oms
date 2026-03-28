package order

import "errors"

var (
	// ErrNoLineItems is returned when an order has zero line items.
	ErrNoLineItems = errors.New("order must have at least one line item")

	// ErrMixedCurrencies is returned when line items have different currencies.
	ErrMixedCurrencies = errors.New("all line items must use the same currency")

	// ErrInvalidQuantity is returned when a line item quantity is <= 0.
	ErrInvalidQuantity = errors.New("quantity must be greater than zero")

	// ErrOrderNotFound is returned when an order cannot be found by ID.
	ErrOrderNotFound = errors.New("order not found")

	// ErrDuplicateIdempotencyKey is returned when an order with the same key exists.
	ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")
)
