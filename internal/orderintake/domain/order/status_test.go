package order_test

import (
	"testing"

	"github.com/oms/internal/orderintake/domain/order"
	"github.com/stretchr/testify/assert"
)

func TestValidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from order.OrderStatus
		to   order.OrderStatus
	}{
		// From PENDING_VALIDATION
		{"pending -> confirmed", order.StatusPendingValidation, order.StatusConfirmed},
		{"pending -> cancelled", order.StatusPendingValidation, order.StatusCancelled},
		// From CONFIRMED
		{"confirmed -> partially_shipped", order.StatusConfirmed, order.StatusPartiallyShipped},
		{"confirmed -> shipped", order.StatusConfirmed, order.StatusShipped},
		{"confirmed -> cancelled", order.StatusConfirmed, order.StatusCancelled},
		{"confirmed -> unfulfillable", order.StatusConfirmed, order.StatusUnfulfillable},
		// From PARTIALLY_SHIPPED
		{"partially_shipped -> shipped", order.StatusPartiallyShipped, order.StatusShipped},
		// From SHIPPED
		{"shipped -> delivered", order.StatusShipped, order.StatusDelivered},
		// From DELIVERED
		{"delivered -> completed", order.StatusDelivered, order.StatusCompleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.from.CanTransitionTo(tt.to),
				"expected %s -> %s to be valid", tt.from, tt.to)
		})
	}
}

func TestInvalidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from order.OrderStatus
		to   order.OrderStatus
	}{
		// Cannot go backwards
		{"confirmed -> pending", order.StatusConfirmed, order.StatusPendingValidation},
		{"shipped -> confirmed", order.StatusShipped, order.StatusConfirmed},
		{"delivered -> shipped", order.StatusDelivered, order.StatusShipped},
		// Cannot skip states
		{"pending -> shipped", order.StatusPendingValidation, order.StatusShipped},
		{"pending -> delivered", order.StatusPendingValidation, order.StatusDelivered},
		{"confirmed -> delivered", order.StatusConfirmed, order.StatusDelivered},
		{"confirmed -> completed", order.StatusConfirmed, order.StatusCompleted},
		// Terminal states cannot transition
		{"cancelled -> confirmed", order.StatusCancelled, order.StatusConfirmed},
		{"cancelled -> pending", order.StatusCancelled, order.StatusPendingValidation},
		{"unfulfillable -> confirmed", order.StatusUnfulfillable, order.StatusConfirmed},
		{"completed -> delivered", order.StatusCompleted, order.StatusDelivered},
		// Cannot cancel after shipping
		{"shipped -> cancelled", order.StatusShipped, order.StatusCancelled},
		{"delivered -> cancelled", order.StatusDelivered, order.StatusCancelled},
		{"partially_shipped -> cancelled", order.StatusPartiallyShipped, order.StatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, tt.from.CanTransitionTo(tt.to),
				"expected %s -> %s to be invalid", tt.from, tt.to)
		})
	}
}

func TestTerminalStates(t *testing.T) {
	terminals := []order.OrderStatus{
		order.StatusCancelled,
		order.StatusUnfulfillable,
		order.StatusCompleted,
	}
	for _, s := range terminals {
		t.Run(string(s), func(t *testing.T) {
			assert.True(t, s.IsTerminal(), "expected %s to be terminal", s)
		})
	}

	nonTerminals := []order.OrderStatus{
		order.StatusPendingValidation,
		order.StatusConfirmed,
		order.StatusPartiallyShipped,
		order.StatusShipped,
		order.StatusDelivered,
	}
	for _, s := range nonTerminals {
		t.Run(string(s)+"_non_terminal", func(t *testing.T) {
			assert.False(t, s.IsTerminal(), "expected %s to be non-terminal", s)
		})
	}
}

func TestIsValid(t *testing.T) {
	assert.True(t, order.StatusPendingValidation.IsValid())
	assert.True(t, order.StatusConfirmed.IsValid())
	assert.True(t, order.StatusCancelled.IsValid())
	assert.False(t, order.OrderStatus("INVALID").IsValid())
	assert.False(t, order.OrderStatus("").IsValid())
}

func TestErrInvalidTransition(t *testing.T) {
	err := order.ErrInvalidTransition{
		From: order.StatusShipped,
		To:   order.StatusCancelled,
	}
	assert.Contains(t, err.Error(), "SHIPPED")
	assert.Contains(t, err.Error(), "CANCELLED")
}
