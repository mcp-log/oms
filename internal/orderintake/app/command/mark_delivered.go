package command

import (
	"context"
	"fmt"

	"github.com/oms/internal/orderintake/domain/order"
)

// MarkDeliveredCommand carries the data needed to mark an order as delivered.
type MarkDeliveredCommand struct {
	OrderID string
}

// MarkDeliveredHandler handles the transition of an order to DELIVERED status.
type MarkDeliveredHandler struct {
	repo      order.Repository
	publisher EventPublisher
}

// NewMarkDeliveredHandler returns a handler wired with the given repository and
// event publisher.
func NewMarkDeliveredHandler(repo order.Repository, pub EventPublisher) *MarkDeliveredHandler {
	return &MarkDeliveredHandler{
		repo:      repo,
		publisher: pub,
	}
}

// Handle marks an order as delivered by loading it, applying the domain
// transition, persisting the update, and publishing any resulting domain
// events.
func (h *MarkDeliveredHandler) Handle(ctx context.Context, cmd MarkDeliveredCommand) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		return nil, fmt.Errorf("finding order %s: %w", cmd.OrderID, err)
	}

	if err := o.MarkDelivered(); err != nil {
		return nil, fmt.Errorf("marking order %s as delivered: %w", cmd.OrderID, err)
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
