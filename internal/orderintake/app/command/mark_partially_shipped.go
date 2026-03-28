package command

import (
	"context"
	"fmt"

	"github.com/oms/internal/orderintake/domain/order"
)

// MarkPartiallyShippedCommand carries the data needed to mark an order as
// partially shipped.
type MarkPartiallyShippedCommand struct {
	OrderID string
}

// MarkPartiallyShippedHandler handles the transition of an order to
// PARTIALLY_SHIPPED status.
type MarkPartiallyShippedHandler struct {
	repo      order.Repository
	publisher EventPublisher
}

// NewMarkPartiallyShippedHandler returns a handler wired with the given
// repository and event publisher.
func NewMarkPartiallyShippedHandler(repo order.Repository, pub EventPublisher) *MarkPartiallyShippedHandler {
	return &MarkPartiallyShippedHandler{
		repo:      repo,
		publisher: pub,
	}
}

// Handle marks an order as partially shipped by loading it, applying the
// domain transition, persisting the update, and publishing any resulting
// domain events.
func (h *MarkPartiallyShippedHandler) Handle(ctx context.Context, cmd MarkPartiallyShippedCommand) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		return nil, fmt.Errorf("finding order %s: %w", cmd.OrderID, err)
	}

	if err := o.MarkPartiallyShipped(); err != nil {
		return nil, fmt.Errorf("marking order %s as partially shipped: %w", cmd.OrderID, err)
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
