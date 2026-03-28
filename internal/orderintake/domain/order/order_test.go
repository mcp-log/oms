package order_test

import (
	"testing"
	"time"

	"github.com/oms/internal/orderintake/domain/order"
	"github.com/oms/pkg/money"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validLine(t *testing.T) order.OrderLine {
	t.Helper()
	unitPrice, err := money.NewMoney("USD", "29.99")
	require.NoError(t, err)
	line, err := order.BuildOrderLine("SKU-001", "Blue Widget", 2, unitPrice)
	require.NoError(t, err)
	return line
}

func validOrder(t *testing.T) *order.Order {
	t.Helper()
	o, err := order.NewOrder(
		"idem-key-1",
		order.ChannelEcommerce,
		"ext-123",
		order.Customer{Name: "Jane Doe", Email: "jane@example.com"},
		order.Address{Line1: "123 Main St", City: "Portland", PostalCode: "97201", CountryCode: "US"},
		order.Address{Line1: "123 Main St", City: "Portland", PostalCode: "97201", CountryCode: "US"},
		[]order.OrderLine{validLine(t)},
		time.Now().UTC(),
	)
	require.NoError(t, err)
	return o
}

// --- Invariant Tests ---

func TestNewOrder_ValidOrder(t *testing.T) {
	o := validOrder(t)

	assert.NotEmpty(t, o.ID)
	assert.Equal(t, "idem-key-1", o.IdempotencyKey)
	assert.Equal(t, order.ChannelEcommerce, o.Channel)
	assert.Equal(t, order.StatusPendingValidation, o.Status)
	assert.Equal(t, "USD", o.CurrencyCode)
	assert.Len(t, o.Lines, 1)
	assert.NotEmpty(t, o.Lines[0].LineID)
}

func TestNewOrder_RejectsZeroLines(t *testing.T) {
	_, err := order.NewOrder(
		"idem-key-2",
		order.ChannelEcommerce,
		"ext-456",
		order.Customer{Name: "Jane", Email: "jane@example.com"},
		order.Address{Line1: "123 Main", City: "Portland", PostalCode: "97201", CountryCode: "US"},
		order.Address{Line1: "123 Main", City: "Portland", PostalCode: "97201", CountryCode: "US"},
		[]order.OrderLine{},
		time.Now().UTC(),
	)
	assert.ErrorIs(t, err, order.ErrNoLineItems)
}

func TestNewOrder_RejectsMixedCurrencies(t *testing.T) {
	usd, err := money.NewMoney("USD", "10.00")
	require.NoError(t, err)
	eur, err := money.NewMoney("EUR", "20.00")
	require.NoError(t, err)

	line1, err := order.BuildOrderLine("SKU-1", "Widget A", 1, usd)
	require.NoError(t, err)
	line2, err := order.BuildOrderLine("SKU-2", "Widget B", 1, eur)
	require.NoError(t, err)

	_, err = order.NewOrder(
		"idem-key-3",
		order.ChannelEcommerce,
		"ext-789",
		order.Customer{Name: "Jane", Email: "jane@example.com"},
		order.Address{Line1: "123 Main", City: "Portland", PostalCode: "97201", CountryCode: "US"},
		order.Address{Line1: "123 Main", City: "Portland", PostalCode: "97201", CountryCode: "US"},
		[]order.OrderLine{line1, line2},
		time.Now().UTC(),
	)
	assert.ErrorIs(t, err, order.ErrMixedCurrencies)
}

func TestNewOrder_TotalIsSumOfLineTotals(t *testing.T) {
	price1, err := money.NewMoney("USD", "10.00")
	require.NoError(t, err)
	price2, err := money.NewMoney("USD", "25.50")
	require.NoError(t, err)

	line1, err := order.BuildOrderLine("SKU-1", "Widget A", 2, price1) // 20.00
	require.NoError(t, err)
	line2, err := order.BuildOrderLine("SKU-2", "Widget B", 3, price2) // 76.50
	require.NoError(t, err)

	o, err := order.NewOrder(
		"idem-key-4",
		order.ChannelEcommerce,
		"ext-101",
		order.Customer{Name: "Jane", Email: "jane@example.com"},
		order.Address{Line1: "123 Main", City: "Portland", PostalCode: "97201", CountryCode: "US"},
		order.Address{Line1: "123 Main", City: "Portland", PostalCode: "97201", CountryCode: "US"},
		[]order.OrderLine{line1, line2},
		time.Now().UTC(),
	)
	require.NoError(t, err)
	assert.Equal(t, "96.5", o.OrderTotal.Amount) // 20.00 + 76.50
	assert.Equal(t, "USD", o.OrderTotal.CurrencyCode)
}

func TestBuildOrderLine_LineTotalIsQuantityTimesUnitPrice(t *testing.T) {
	price, err := money.NewMoney("USD", "29.99")
	require.NoError(t, err)

	line, err := order.BuildOrderLine("SKU-001", "Widget", 3, price)
	require.NoError(t, err)

	assert.Equal(t, "89.97", line.LineTotal.Amount)
	assert.Equal(t, "USD", line.LineTotal.CurrencyCode)
}

func TestBuildOrderLine_RejectsZeroQuantity(t *testing.T) {
	price, err := money.NewMoney("USD", "10.00")
	require.NoError(t, err)

	_, err = order.BuildOrderLine("SKU-001", "Widget", 0, price)
	assert.ErrorIs(t, err, order.ErrInvalidQuantity)
}

func TestBuildOrderLine_RejectsNegativeQuantity(t *testing.T) {
	price, err := money.NewMoney("USD", "10.00")
	require.NoError(t, err)

	_, err = order.BuildOrderLine("SKU-001", "Widget", -1, price)
	assert.ErrorIs(t, err, order.ErrInvalidQuantity)
}

// --- State Transition Tests on Aggregate ---

func TestOrder_Confirm(t *testing.T) {
	o := validOrder(t)
	err := o.Confirm()
	require.NoError(t, err)

	assert.Equal(t, order.StatusConfirmed, o.Status)
	assert.NotNil(t, o.ConfirmedAt)

	// Should have domain events
	evts := o.DomainEvents()
	require.Len(t, evts, 1)
	assert.Equal(t, "order.confirmed", evts[0].EventType())
}

func TestOrder_Confirm_InvalidFromShipped(t *testing.T) {
	o := validOrder(t)
	require.NoError(t, o.Confirm())
	require.NoError(t, o.MarkShipped())

	err := o.Confirm()
	assert.Error(t, err)
	var transErr order.ErrInvalidTransition
	assert.ErrorAs(t, err, &transErr)
}

func TestOrder_Cancel_FromPending(t *testing.T) {
	o := validOrder(t)
	err := o.Cancel("customer request")
	require.NoError(t, err)

	assert.Equal(t, order.StatusCancelled, o.Status)
	assert.NotNil(t, o.CancelledAt)
	assert.Equal(t, "customer request", o.CancellationReason)

	evts := o.DomainEvents()
	require.Len(t, evts, 1)
	assert.Equal(t, "order.cancelled", evts[0].EventType())
}

func TestOrder_Cancel_FromConfirmed(t *testing.T) {
	o := validOrder(t)
	require.NoError(t, o.Confirm())
	o.ClearEvents()

	err := o.Cancel("out of stock")
	require.NoError(t, err)
	assert.Equal(t, order.StatusCancelled, o.Status)
}

func TestOrder_Cancel_InvalidFromShipped(t *testing.T) {
	o := validOrder(t)
	require.NoError(t, o.Confirm())
	require.NoError(t, o.MarkShipped())

	err := o.Cancel("too late")
	assert.Error(t, err)
}

func TestOrder_MarkPartiallyShipped(t *testing.T) {
	o := validOrder(t)
	require.NoError(t, o.Confirm())
	o.ClearEvents()

	err := o.MarkPartiallyShipped()
	require.NoError(t, err)
	assert.Equal(t, order.StatusPartiallyShipped, o.Status)
}

func TestOrder_MarkShipped_FromConfirmed(t *testing.T) {
	o := validOrder(t)
	require.NoError(t, o.Confirm())
	o.ClearEvents()

	err := o.MarkShipped()
	require.NoError(t, err)
	assert.Equal(t, order.StatusShipped, o.Status)

	evts := o.DomainEvents()
	// Should have OrderShipped + OrderStatusChanged
	assert.GreaterOrEqual(t, len(evts), 1)
}

func TestOrder_MarkShipped_FromPartiallyShipped(t *testing.T) {
	o := validOrder(t)
	require.NoError(t, o.Confirm())
	require.NoError(t, o.MarkPartiallyShipped())
	o.ClearEvents()

	err := o.MarkShipped()
	require.NoError(t, err)
	assert.Equal(t, order.StatusShipped, o.Status)
}

func TestOrder_MarkDelivered(t *testing.T) {
	o := validOrder(t)
	require.NoError(t, o.Confirm())
	require.NoError(t, o.MarkShipped())
	o.ClearEvents()

	err := o.MarkDelivered()
	require.NoError(t, err)
	assert.Equal(t, order.StatusDelivered, o.Status)
}

func TestOrder_MarkUnfulfillable(t *testing.T) {
	o := validOrder(t)
	require.NoError(t, o.Confirm())
	o.ClearEvents()

	err := o.MarkUnfulfillable()
	require.NoError(t, err)
	assert.Equal(t, order.StatusUnfulfillable, o.Status)
}

func TestOrder_MarkCompleted(t *testing.T) {
	o := validOrder(t)
	require.NoError(t, o.Confirm())
	require.NoError(t, o.MarkShipped())
	require.NoError(t, o.MarkDelivered())
	o.ClearEvents()

	err := o.MarkCompleted()
	require.NoError(t, err)
	assert.Equal(t, order.StatusCompleted, o.Status)
}

func TestOrder_FullLifecycle_HappyPath(t *testing.T) {
	o := validOrder(t)
	assert.Equal(t, order.StatusPendingValidation, o.Status)

	require.NoError(t, o.Confirm())
	assert.Equal(t, order.StatusConfirmed, o.Status)

	require.NoError(t, o.MarkShipped())
	assert.Equal(t, order.StatusShipped, o.Status)

	require.NoError(t, o.MarkDelivered())
	assert.Equal(t, order.StatusDelivered, o.Status)

	require.NoError(t, o.MarkCompleted())
	assert.Equal(t, order.StatusCompleted, o.Status)

	// All events should be collected
	assert.NotEmpty(t, o.DomainEvents())
}

func TestOrder_ClearEvents(t *testing.T) {
	o := validOrder(t)
	require.NoError(t, o.Confirm())
	assert.NotEmpty(t, o.DomainEvents())

	o.ClearEvents()
	assert.Empty(t, o.DomainEvents())
}

func TestNewOrder_AssignsLineIDs(t *testing.T) {
	line := order.OrderLine{
		SKU:         "SKU-1",
		ProductName: "Widget",
		Quantity:    1,
		UnitPrice:   money.Money{CurrencyCode: "USD", Amount: "10.00"},
		LineTotal:   money.Money{CurrencyCode: "USD", Amount: "10.00"},
	}

	o, err := order.NewOrder(
		"idem-key-5",
		order.ChannelB2B,
		"",
		order.Customer{Name: "Acme Corp", Email: "orders@acme.com"},
		order.Address{Line1: "456 Oak Ave", City: "Seattle", PostalCode: "98101", CountryCode: "US"},
		order.Address{Line1: "456 Oak Ave", City: "Seattle", PostalCode: "98101", CountryCode: "US"},
		[]order.OrderLine{line},
		time.Now().UTC(),
	)
	require.NoError(t, err)
	assert.NotEmpty(t, o.Lines[0].LineID, "line ID should be auto-assigned")
}

func TestOrder_MultipleLines_SameCurrency(t *testing.T) {
	price1, _ := money.NewMoney("USD", "10.00")
	price2, _ := money.NewMoney("USD", "20.00")
	price3, _ := money.NewMoney("USD", "5.00")

	line1, _ := order.BuildOrderLine("SKU-1", "Widget A", 1, price1) // 10.00
	line2, _ := order.BuildOrderLine("SKU-2", "Widget B", 2, price2) // 40.00
	line3, _ := order.BuildOrderLine("SKU-3", "Widget C", 4, price3) // 20.00

	o, err := order.NewOrder(
		"idem-key-multi",
		order.ChannelDirect,
		"",
		order.Customer{Name: "Test", Email: "test@test.com"},
		order.Address{Line1: "1 St", City: "NYC", PostalCode: "10001", CountryCode: "US"},
		order.Address{Line1: "1 St", City: "NYC", PostalCode: "10001", CountryCode: "US"},
		[]order.OrderLine{line1, line2, line3},
		time.Now().UTC(),
	)
	require.NoError(t, err)
	assert.Equal(t, "70", o.OrderTotal.Amount) // 10 + 40 + 20
	assert.Len(t, o.Lines, 3)
}
