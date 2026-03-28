package command

import (
	"context"
	"fmt"

	"github.com/oms/internal/orderintake/domain/order"
)

// MarkShippedCommand carries the data needed to mark an order as shipped.
type MarkShippedCommand struct {
	OrderID string
}

// MarkShippedHandler handles the transition of an order to SHIPPED status.
type MarkShippedHandler struct {
	repo      order.Repository
	publisher EventPublisher
}

// NewMarkShippedHandler returns a handler wired with the given repository and
// event publisher.
func NewMarkShippedHandler(repo order.Repository, pub EventPublisher) *MarkShippedHandler {
	return &MarkShippedHandler{
		repo:      repo,
		publisher: pub,
	}
}

// Handle marks an order as shipped by loading it, applying the domain
// transition, persisting the update, and publishing any resulting domain
// events.
func (h *MarkShippedHandler) Handle(ctx context.Context, cmd MarkShippedCommand) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		return nil, fmt.Errorf("finding order %s: %w", cmd.OrderID, err)
	}

	if err := o.MarkShipped(); err != nil {
		return nil, fmt.Errorf("marking order %s as shipped: %w", cmd.OrderID, err)
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
