package order

import (
	"fmt"
	"time"

	"github.com/oms/pkg/events"
	"github.com/oms/pkg/identity"
	"github.com/oms/pkg/money"
)

// Channel represents the origin of an order.
type Channel string

const (
	ChannelEcommerce   Channel = "ECOMMERCE"
	ChannelMarketplace Channel = "MARKETPLACE"
	ChannelB2B         Channel = "B2B"
	ChannelDirect      Channel = "DIRECT"
)

// Address is a value object representing a physical address.
type Address struct {
	Line1         string
	Line2         string
	City          string
	StateOrRegion string
	PostalCode    string
	CountryCode   string
}

// Customer holds customer information.
type Customer struct {
	Name  string
	Email string
	Phone string
}

// OrderLine is a value object representing a single line in an order.
type OrderLine struct {
	LineID      string
	SKU         string
	ProductName string
	Quantity    int
	UnitPrice   money.Money
	LineTotal   money.Money
}

// Order is the aggregate root for the Order Intake bounded context.
type Order struct {
	ID                 string
	IdempotencyKey     string
	Channel            Channel
	ExternalID         string
	Customer           Customer
	ShippingAddress    Address
	BillingAddress     Address
	Lines              []OrderLine
	CurrencyCode       string
	OrderTotal         money.Money
	Status             OrderStatus
	PlacedAt           time.Time
	ConfirmedAt        *time.Time
	CancelledAt        *time.Time
	CancellationReason string
	CreatedAt          time.Time
	UpdatedAt          time.Time

	domainEvents []events.DomainEvent
}

// NewOrder creates a new Order aggregate and validates all invariants.
func NewOrder(
	idempotencyKey string,
	channel Channel,
	externalID string,
	customer Customer,
	shippingAddress Address,
	billingAddress Address,
	lines []OrderLine,
	placedAt time.Time,
) (*Order, error) {
	if len(lines) == 0 {
		return nil, ErrNoLineItems
	}

	// Validate all lines have the same currency
	currency := lines[0].UnitPrice.CurrencyCode
	for i, line := range lines {
		if line.UnitPrice.CurrencyCode != currency {
			return nil, fmt.Errorf("%w: line %d has currency %s, expected %s",
				ErrMixedCurrencies, i, line.UnitPrice.CurrencyCode, currency)
		}
		if line.LineTotal.CurrencyCode != currency {
			return nil, fmt.Errorf("%w: line %d total has currency %s, expected %s",
				ErrMixedCurrencies, i, line.LineTotal.CurrencyCode, currency)
		}
	}

	// Compute order total
	total, err := money.NewMoney(currency, "0")
	if err != nil {
		return nil, fmt.Errorf("initializing total: %w", err)
	}
	for _, line := range lines {
		total, err = total.Add(line.LineTotal)
		if err != nil {
			return nil, fmt.Errorf("summing line totals: %w", err)
		}
	}

	now := time.Now().UTC()
	id := identity.NewID()

	// Assign IDs to lines that don't have them
	for i := range lines {
		if lines[i].LineID == "" {
			lines[i].LineID = identity.NewID()
		}
	}

	return &Order{
		ID:              id,
		IdempotencyKey:  idempotencyKey,
		Channel:         channel,
		ExternalID:      externalID,
		Customer:        customer,
		ShippingAddress: shippingAddress,
		BillingAddress:  billingAddress,
		Lines:           lines,
		CurrencyCode:    currency,
		OrderTotal:      total,
		Status:          StatusPendingValidation,
		PlacedAt:        placedAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// Confirm transitions the order to CONFIRMED status.
func (o *Order) Confirm() error {
	if err := o.transitionTo(StatusConfirmed); err != nil {
		return err
	}
	now := time.Now().UTC()
	o.ConfirmedAt = &now
	o.UpdatedAt = now

	lineSnapshots := make([]LineSnapshot, len(o.Lines))
	for i, l := range o.Lines {
		lineSnapshots[i] = LineSnapshot(l)
	}

	o.addEvent(OrderConfirmed{
		BaseEvent:       events.NewBaseEvent("order.confirmed", o.ID, "Order"),
		OrderID:         o.ID,
		Channel:         o.Channel,
		CustomerName:    o.Customer.Name,
		CustomerEmail:   o.Customer.Email,
		ShippingAddress: o.ShippingAddress,
		Lines:           lineSnapshots,
		OrderTotal:      o.OrderTotal,
		ConfirmedAt:     now,
	})
	return nil
}

// Cancel transitions the order to CANCELLED status.
func (o *Order) Cancel(reason string) error {
	if err := o.transitionTo(StatusCancelled); err != nil {
		return err
	}
	now := time.Now().UTC()
	o.CancelledAt = &now
	o.CancellationReason = reason
	o.UpdatedAt = now

	o.addEvent(OrderCancelled{
		BaseEvent:   events.NewBaseEvent("order.cancelled", o.ID, "Order"),
		OrderID:     o.ID,
		Reason:      reason,
		CancelledAt: now,
	})
	return nil
}

// MarkPartiallyShipped transitions the order to PARTIALLY_SHIPPED.
func (o *Order) MarkPartiallyShipped() error {
	if err := o.transitionTo(StatusPartiallyShipped); err != nil {
		return err
	}
	o.UpdatedAt = time.Now().UTC()

	o.addEvent(OrderStatusChanged{
		BaseEvent:      events.NewBaseEvent("order.status_changed", o.ID, "Order"),
		OrderID:        o.ID,
		PreviousStatus: StatusConfirmed,
		NewStatus:      StatusPartiallyShipped,
		ChangedAt:      o.UpdatedAt,
	})
	return nil
}

// MarkShipped transitions the order to SHIPPED.
func (o *Order) MarkShipped() error {
	prev := o.Status
	if err := o.transitionTo(StatusShipped); err != nil {
		return err
	}
	now := time.Now().UTC()
	o.UpdatedAt = now

	o.addEvent(OrderShipped{
		BaseEvent: events.NewBaseEvent("order.shipped", o.ID, "Order"),
		OrderID:   o.ID,
		ShippedAt: now,
	})
	o.addEvent(OrderStatusChanged{
		BaseEvent:      events.NewBaseEvent("order.status_changed", o.ID, "Order"),
		OrderID:        o.ID,
		PreviousStatus: prev,
		NewStatus:      StatusShipped,
		ChangedAt:      now,
	})
	return nil
}

// MarkDelivered transitions the order to DELIVERED.
func (o *Order) MarkDelivered() error {
	if err := o.transitionTo(StatusDelivered); err != nil {
		return err
	}
	now := time.Now().UTC()
	o.UpdatedAt = now

	o.addEvent(OrderDelivered{
		BaseEvent:   events.NewBaseEvent("order.delivered", o.ID, "Order"),
		OrderID:     o.ID,
		DeliveredAt: now,
	})
	return nil
}

// MarkUnfulfillable transitions the order to UNFULFILLABLE.
func (o *Order) MarkUnfulfillable() error {
	if err := o.transitionTo(StatusUnfulfillable); err != nil {
		return err
	}
	o.UpdatedAt = time.Now().UTC()

	o.addEvent(OrderStatusChanged{
		BaseEvent:      events.NewBaseEvent("order.status_changed", o.ID, "Order"),
		OrderID:        o.ID,
		PreviousStatus: StatusConfirmed,
		NewStatus:      StatusUnfulfillable,
		ChangedAt:      o.UpdatedAt,
	})
	return nil
}

// MarkCompleted transitions the order to COMPLETED.
func (o *Order) MarkCompleted() error {
	if err := o.transitionTo(StatusCompleted); err != nil {
		return err
	}
	o.UpdatedAt = time.Now().UTC()

	o.addEvent(OrderStatusChanged{
		BaseEvent:      events.NewBaseEvent("order.status_changed", o.ID, "Order"),
		OrderID:        o.ID,
		PreviousStatus: StatusDelivered,
		NewStatus:      StatusCompleted,
		ChangedAt:      o.UpdatedAt,
	})
	return nil
}

// DomainEvents returns the domain events collected during aggregate operations.
func (o *Order) DomainEvents() []events.DomainEvent {
	return o.domainEvents
}

// ClearEvents clears collected domain events (call after publishing).
func (o *Order) ClearEvents() {
	o.domainEvents = nil
}

func (o *Order) transitionTo(target OrderStatus) error {
	if !o.Status.CanTransitionTo(target) {
		return ErrInvalidTransition{From: o.Status, To: target}
	}
	o.Status = target
	return nil
}

func (o *Order) addEvent(event events.DomainEvent) {
	o.domainEvents = append(o.domainEvents, event)
}

// BuildOrderLine creates an OrderLine with computed LineTotal.
func BuildOrderLine(sku, productName string, quantity int, unitPrice money.Money) (OrderLine, error) {
	if quantity <= 0 {
		return OrderLine{}, ErrInvalidQuantity
	}
	lineTotal := unitPrice.Multiply(quantity)
	return OrderLine{
		LineID:      identity.NewID(),
		SKU:         sku,
		ProductName: productName,
		Quantity:    quantity,
		UnitPrice:   unitPrice,
		LineTotal:   lineTotal,
	}, nil
}
