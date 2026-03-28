package command

import (
	"context"
	"fmt"

	"github.com/oms/internal/orderintake/domain/order"
)

// ConfirmOrderCommand carries the data needed to confirm an order.
type ConfirmOrderCommand struct {
	OrderID string
}

// ConfirmOrderHandler handles the confirmation of orders.
type ConfirmOrderHandler struct {
	repo      order.Repository
	publisher EventPublisher
}

// NewConfirmOrderHandler returns a handler wired with the given repository and
// event publisher.
func NewConfirmOrderHandler(repo order.Repository, pub EventPublisher) *ConfirmOrderHandler {
	return &ConfirmOrderHandler{
		repo:      repo,
		publisher: pub,
	}
}

// Handle confirms an order by loading it, applying the domain transition,
// persisting the update, and publishing any resulting domain events.
func (h *ConfirmOrderHandler) Handle(ctx context.Context, cmd ConfirmOrderCommand) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		return nil, fmt.Errorf("finding order %s: %w", cmd.OrderID, err)
	}

	if err := o.Confirm(); err != nil {
		return nil, fmt.Errorf("confirming order %s: %w", cmd.OrderID, err)
	}

	if err := h.repo.Update(ctx, o); err != nil {
		return nil, fmt.Errorf("updating order %s: %w", cmd.OrderID, err)
	}

	if err := h.publisher.Publish(ctx, o.DomainEvents()...); err != nil {
		return nil, fmt.Errorf("publishing events for order %s: %w", cmd.OrderID, err)
	}
	o.ClearEvents()

	return o, nil
}
