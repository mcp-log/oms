package order

import (
	"time"

	"github.com/oms/pkg/events"
	"github.com/oms/pkg/money"
)

// OrderConfirmed is emitted when an order transitions to CONFIRMED.
type OrderConfirmed struct {
	events.BaseEvent
	OrderID         string
	Channel         Channel
	CustomerName    string
	CustomerEmail   string
	ShippingAddress Address
	Lines           []LineSnapshot
	OrderTotal      money.Money
	ConfirmedAt     time.Time
}

// LineSnapshot is an immutable snapshot of an order line for events.
type LineSnapshot struct {
	LineID      string
	SKU         string
	ProductName string
	Quantity    int
	UnitPrice   money.Money
	LineTotal   money.Money
}

// OrderCancelled is emitted when an order transitions to CANCELLED.
type OrderCancelled struct {
	events.BaseEvent
	OrderID     string
	Reason      string
	CancelledAt time.Time
}

// OrderShipped is emitted when an order transitions to SHIPPED.
type OrderShipped struct {
	events.BaseEvent
	OrderID   string
	ShippedAt time.Time
}

// OrderDelivered is emitted when an order transitions to DELIVERED.
type OrderDelivered struct {
	events.BaseEvent
	OrderID     string
	DeliveredAt time.Time
}

// OrderStatusChanged is emitted on any state transition (for audit/CQRS).
type OrderStatusChanged struct {
	events.BaseEvent
	OrderID        string
	PreviousStatus OrderStatus
	NewStatus      OrderStatus
	ChangedAt      time.Time
}
