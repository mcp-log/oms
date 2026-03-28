package command

import (
	"context"
	"fmt"

	"github.com/oms/internal/orderintake/domain/order"
)

// MarkCompletedCommand carries the data needed to mark an order as completed.
type MarkCompletedCommand struct {
	OrderID string
}

// MarkCompletedHandler handles the transition of an order to COMPLETED status.
type MarkCompletedHandler struct {
	repo      order.Repository
	publisher EventPublisher
}

// NewMarkCompletedHandler returns a handler wired with the given repository and
// event publisher.
func NewMarkCompletedHandler(repo order.Repository, pub EventPublisher) *MarkCompletedHandler {
	return &MarkCompletedHandler{
		repo:      repo,
		publisher: pub,
	}
}

// Handle marks an order as completed by loading it, applying the domain
// transition, persisting the update, and publishing any resulting domain
// events.
func (h *MarkCompletedHandler) Handle(ctx context.Context, cmd MarkCompletedCommand) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		return nil, fmt.Errorf("finding order %s: %w", cmd.OrderID, err)
	}

	if err := o.MarkCompleted(); err != nil {
		return nil, fmt.Errorf("marking order %s as completed: %w", cmd.OrderID, err)
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
