package command

import (
	"context"
	"fmt"
	"time"

	"github.com/oms/internal/orderintake/domain/order"
	"github.com/oms/pkg/events"
	"github.com/oms/pkg/money"
)

// EventPublisher publishes domain events after persistence.
type EventPublisher interface {
	Publish(ctx context.Context, events ...events.DomainEvent) error
}

// CreateOrderCommand carries the data needed to create a new order.
type CreateOrderCommand struct {
	IdempotencyKey  string
	Channel         order.Channel
	ExternalID      string
	Customer        order.Customer
	ShippingAddress order.Address
	BillingAddress  order.Address
	Lines           []CreateOrderLine
	PlacedAt        time.Time
}

// CreateOrderLine represents a single line item in a create order request.
type CreateOrderLine struct {
	SKU         string
	ProductName string
	Quantity    int
	UnitPrice   money.Money
}

// CreateOrderResult holds the outcome of a CreateOrder command.
type CreateOrderResult struct {
	Order      *order.Order
	IsExisting bool // true if returned from idempotency check
}

// CreateOrderHandler handles the creation of new orders.
type CreateOrderHandler struct {
	repo      order.Repository
	publisher EventPublisher
}

// NewCreateOrderHandler returns a handler wired with the given repository and
// event publisher.
func NewCreateOrderHandler(repo order.Repository, pub EventPublisher) *CreateOrderHandler {
	return &CreateOrderHandler{
		repo:      repo,
		publisher: pub,
	}
}

// Handle creates a new order. If the idempotency key already exists, the
// existing order is returned without side effects.
func (h *CreateOrderHandler) Handle(ctx context.Context, cmd CreateOrderCommand) (CreateOrderResult, error) {
	// Check idempotency key first.
	existing, err := h.repo.FindByIdempotencyKey(ctx, cmd.IdempotencyKey)
	if err != nil {
		return CreateOrderResult{}, fmt.Errorf("checking idempotency key: %w", err)
	}
	if existing != nil {
		return CreateOrderResult{
			Order:      existing,
			IsExisting: true,
		}, nil
	}

	// Build order lines from the command.
	lines := make([]order.OrderLine, 0, len(cmd.Lines))
	for _, cl := range cmd.Lines {
		ol, err := order.BuildOrderLine(cl.SKU, cl.ProductName, cl.Quantity, cl.UnitPrice)
		if err != nil {
			return CreateOrderResult{}, fmt.Errorf("building order line %q: %w", cl.SKU, err)
		}
		lines = append(lines, ol)
	}

	// Create the aggregate.
	o, err := order.NewOrder(
		cmd.IdempotencyKey,
		cmd.Channel,
		cmd.ExternalID,
		cmd.Customer,
		cmd.ShippingAddress,
		cmd.BillingAddress,
		lines,
		cmd.PlacedAt,
	)
	if err != nil {
		return CreateOrderResult{}, fmt.Errorf("creating order: %w", err)
	}

	// Persist.
	if err := h.repo.Save(ctx, o); err != nil {
		return CreateOrderResult{}, fmt.Errorf("saving order: %w", err)
	}

	// No events to publish on creation; the order starts as PENDING_VALIDATION.

	return CreateOrderResult{
		Order:      o,
		IsExisting: false,
	}, nil
}
