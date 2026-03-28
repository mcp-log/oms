package command

import (
	"context"
	"fmt"

	"github.com/oms/internal/orderintake/domain/order"
)

// MarkUnfulfillableCommand carries the data needed to mark an order as
// unfulfillable.
type MarkUnfulfillableCommand struct {
	OrderID string
}

// MarkUnfulfillableHandler handles the transition of an order to UNFULFILLABLE
// status.
type MarkUnfulfillableHandler struct {
	repo      order.Repository
	publisher EventPublisher
}

// NewMarkUnfulfillableHandler returns a handler wired with the given repository
// and event publisher.
func NewMarkUnfulfillableHandler(repo order.Repository, pub EventPublisher) *MarkUnfulfillableHandler {
	return &MarkUnfulfillableHandler{
		repo:      repo,
		publisher: pub,
	}
}

// Handle marks an order as unfulfillable by loading it, applying the domain
// transition, persisting the update, and publishing any resulting domain
// events.
func (h *MarkUnfulfillableHandler) Handle(ctx context.Context, cmd MarkUnfulfillableCommand) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		return nil, fmt.Errorf("finding order %s: %w", cmd.OrderID, err)
	}

	if err := o.MarkUnfulfillable(); err != nil {
		return nil, fmt.Errorf("marking order %s as unfulfillable: %w", cmd.OrderID, err)
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
