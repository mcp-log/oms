package query

import (
	"context"
	"fmt"

	"github.com/oms/internal/orderintake/domain/order"
)

// GetOrderQuery carries the data needed to retrieve a single order.
type GetOrderQuery struct {
	OrderID string
}

// GetOrderHandler handles queries for retrieving a single order by ID.
type GetOrderHandler struct {
	repo order.Repository
}

// NewGetOrderHandler returns a handler wired with the given repository.
func NewGetOrderHandler(repo order.Repository) *GetOrderHandler {
	return &GetOrderHandler{
		repo: repo,
	}
}

// Handle retrieves an order by its ID. Returns ErrOrderNotFound if the order
// does not exist.
func (h *GetOrderHandler) Handle(ctx context.Context, q GetOrderQuery) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, q.OrderID)
	if err != nil {
		return nil, fmt.Errorf("finding order %s: %w", q.OrderID, err)
	}
	return o, nil
}
