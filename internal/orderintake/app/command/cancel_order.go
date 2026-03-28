package command

import (
	"context"
	"fmt"

	"github.com/oms/internal/orderintake/domain/order"
)

// CancelOrderCommand carries the data needed to cancel an order.
type CancelOrderCommand struct {
	OrderID string
	Reason  string
}

// CancelOrderHandler handles the cancellation of orders.
type CancelOrderHandler struct {
	repo      order.Repository
	publisher EventPublisher
}

// NewCancelOrderHandler returns a handler wired with the given repository and
// event publisher.
func NewCancelOrderHandler(repo order.Repository, pub EventPublisher) *CancelOrderHandler {
	return &CancelOrderHandler{
		repo:      repo,
		publisher: pub,
	}
}

// Handle cancels an order by loading it, applying the domain transition with
// the given reason, persisting the update, and publishing any resulting domain
// events.
func (h *CancelOrderHandler) Handle(ctx context.Context, cmd CancelOrderCommand) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		return nil, fmt.Errorf("finding order %s: %w", cmd.OrderID, err)
	}

	if err := o.Cancel(cmd.Reason); err != nil {
		return nil, fmt.Errorf("cancelling order %s: %w", cmd.OrderID, err)
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
