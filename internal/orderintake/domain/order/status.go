package order

import "fmt"

// OrderStatus represents the lifecycle state of an order.
type OrderStatus string

const (
	StatusPendingValidation OrderStatus = "PENDING_VALIDATION"
	StatusConfirmed         OrderStatus = "CONFIRMED"
	StatusPartiallyShipped  OrderStatus = "PARTIALLY_SHIPPED"
	StatusShipped           OrderStatus = "SHIPPED"
	StatusUnfulfillable     OrderStatus = "UNFULFILLABLE"
	StatusDelivered         OrderStatus = "DELIVERED"
	StatusCancelled         OrderStatus = "CANCELLED"
	StatusCompleted         OrderStatus = "COMPLETED"
)

// validTransitions defines the allowed state machine transitions.
var validTransitions = map[OrderStatus][]OrderStatus{
	StatusPendingValidation: {StatusConfirmed, StatusCancelled},
	StatusConfirmed:         {StatusPartiallyShipped, StatusShipped, StatusCancelled, StatusUnfulfillable},
	StatusPartiallyShipped:  {StatusShipped},
	StatusShipped:           {StatusDelivered},
	StatusDelivered:         {StatusCompleted},
}

// terminalStates are states from which no further transitions are possible.
var terminalStates = map[OrderStatus]bool{
	StatusCancelled:     true,
	StatusUnfulfillable: true,
	StatusCompleted:     true,
}

// CanTransitionTo checks whether transitioning from the current status to the
// target status is allowed by the state machine.
func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	allowed, ok := validTransitions[s]
	if !ok {
		return false
	}
	for _, t := range allowed {
		if t == target {
			return true
		}
	}
	return false
}

// IsTerminal returns true if this status is a terminal (final) state.
func (s OrderStatus) IsTerminal() bool {
	return terminalStates[s]
}

// IsValid returns true if the status is a known OrderStatus value.
func (s OrderStatus) IsValid() bool {
	switch s {
	case StatusPendingValidation, StatusConfirmed, StatusPartiallyShipped,
		StatusShipped, StatusUnfulfillable, StatusDelivered,
		StatusCancelled, StatusCompleted:
		return true
	}
	return false
}

// ErrInvalidTransition is returned when a state transition is not allowed.
type ErrInvalidTransition struct {
	From OrderStatus
	To   OrderStatus
}

func (e ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid state transition from %s to %s", e.From, e.To)
}
