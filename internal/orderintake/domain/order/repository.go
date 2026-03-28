package order

import "context"

// Repository is the port for persisting and retrieving Order aggregates.
type Repository interface {
	// Save persists a new order. Returns ErrDuplicateIdempotencyKey if the key exists.
	Save(ctx context.Context, order *Order) error

	// FindByID retrieves an order by its ID. Returns ErrOrderNotFound if not found.
	FindByID(ctx context.Context, id string) (*Order, error)

	// FindByIdempotencyKey retrieves an order by its idempotency key.
	// Returns nil, nil if not found.
	FindByIdempotencyKey(ctx context.Context, key string) (*Order, error)

	// Update persists changes to an existing order.
	Update(ctx context.Context, order *Order) error

	// List retrieves orders matching the given filter with cursor-based pagination.
	List(ctx context.Context, filter ListFilter) ([]*Order, string, error)
}

// ListFilter contains filtering and pagination parameters for listing orders.
type ListFilter struct {
	Status  *OrderStatus
	Channel *Channel
	Cursor  string
	Limit   int
}
