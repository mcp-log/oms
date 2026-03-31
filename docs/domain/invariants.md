---
layout: default
title: Business Rules
parent: Domain
nav_order: 2
---

# Domain Invariants & Business Rules
{: .no_toc }

Complete reference for business rules enforced by the Order aggregate.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

**Invariants** are business rules that must **always** be true for an aggregate to be valid. The Order aggregate enforces these rules at construction and before any state transition.

**Key Principle**: If an invariant violation occurs, the operation fails immediately with a domain error. The aggregate is never left in an invalid state.

---

## Order-Level Invariants

### INV-01: Minimum One Line Item

**Rule**: An order must have at least one line item.

**Rationale**: An order without items has no business meaning.

**Enforcement**:
```go
func NewOrder(cmd CreateOrderCommand) (*Order, error) {
    if len(cmd.Lines) == 0 {
        return nil, ErrNoLineItems
    }
    // ...
}
```

**Error Response**:
```json
{
  "type": "https://api.example.com/problems/no-line-items",
  "title": "Order must have at least one line item",
  "status": 422,
  "detail": "Cannot create an order with zero line items"
}
```

---

### INV-02: Single Currency

**Rule**: All line items in an order must use the same currency.

**Rationale**: Multi-currency orders require complex FX handling and are out of scope for Order Intake.

**Enforcement**:
```go
func NewOrder(cmd CreateOrderCommand) (*Order, error) {
    firstCurrency := cmd.Lines[0].UnitPrice.CurrencyCode
    for _, line := range cmd.Lines {
        if line.UnitPrice.CurrencyCode != firstCurrency {
            return nil, ErrMixedCurrencies
        }
    }
    // ...
}
```

**Error Response**:
```json
{
  "type": "https://api.example.com/problems/mixed-currencies",
  "title": "Mixed currencies not allowed",
  "status": 422,
  "detail": "All line items must use the same currency. Found: USD, EUR"
}
```

---

### INV-03: Order Total Consistency

**Rule**: `OrderTotal` must equal the sum of all `LineTotal` values.

**Rationale**: Prevents pricing discrepancies and ensures accurate billing.

**Enforcement**:
```go
func (o *Order) RecalculateTotal() {
    total := decimal.Zero
    for _, line := range o.Lines {
        lineAmount, _ := decimal.NewFromString(line.LineTotal.Amount)
        total = total.Add(lineAmount)
    }
    o.OrderTotal = Money{
        CurrencyCode: o.CurrencyCode,
        Amount:       total.String(),
    }
}
```

**Note**: Total is **computed**, not user-provided. Recalculated after any line changes.

---

### INV-04: Immutability After Confirmation

**Rule**: Core order data (lines, customer, addresses) cannot be modified after the order is confirmed.

**Rationale**: Once confirmed, inventory is allocated and downstream systems are notified. Changing core data would require complex compensation logic.

**Enforcement**:
```go
func (o *Order) AddLine(line *OrderLine) error {
    if o.Status != StatusPendingValidation {
        return ErrOrderImmutable
    }
    o.Lines = append(o.Lines, line)
    return nil
}

func (o *Order) RemoveLine(lineID uuid.UUID) error {
    if o.Status != StatusPendingValidation {
        return ErrOrderImmutable
    }
    // ...
}
```

**Error Response**:
```json
{
  "type": "https://api.example.com/problems/order-immutable",
  "title": "Order cannot be modified",
  "status": 409,
  "detail": "Cannot modify order after confirmation. Current status: CONFIRMED"
}
```

---

### INV-05: Idempotency Key Uniqueness

**Rule**: Each `IdempotencyKey` must be unique across all orders.

**Rationale**: Prevents duplicate order creation when clients retry requests.

**Enforcement**:
```sql
CREATE TABLE orders (
    id              UUID PRIMARY KEY,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    -- ...
);
```

**Error Response** (on duplicate):
```json
{
  "type": "https://api.example.com/problems/duplicate-idempotency-key",
  "title": "Idempotency key already used",
  "status": 409,
  "detail": "An order with this idempotency key already exists: order-2024-01-15-abc123"
}
```

---

## Line-Level Invariants

### INV-06: Positive Quantity

**Rule**: Line item `Quantity` must be greater than zero.

**Rationale**: Zero or negative quantities have no business meaning.

**Enforcement**:
```go
func NewOrderLine(sku, name string, qty int, unitPrice Money) (*OrderLine, error) {
    if qty <= 0 {
        return nil, ErrInvalidQuantity
    }
    // ...
}
```

**Database Constraint**:
```sql
CREATE TABLE order_lines (
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    -- ...
);
```

**Error Response**:
```json
{
  "type": "https://api.example.com/problems/invalid-quantity",
  "title": "Invalid quantity",
  "status": 422,
  "detail": "Line item quantity must be greater than zero. Got: 0"
}
```

---

### INV-07: Line Total Consistency

**Rule**: `LineTotal = Quantity × UnitPrice`

**Rationale**: Prevents pricing manipulation.

**Enforcement**:
```go
func NewOrderLine(sku, name string, qty int, unitPrice Money) (*OrderLine, error) {
    unitPriceDecimal, _ := decimal.NewFromString(unitPrice.Amount)
    lineTotal := unitPriceDecimal.Mul(decimal.NewFromInt(int64(qty)))

    return &OrderLine{
        LineID:      identity.NewUUID(),
        SKU:         sku,
        ProductName: name,
        Quantity:    qty,
        UnitPrice:   unitPrice,
        LineTotal: Money{
            CurrencyCode: unitPrice.CurrencyCode,
            Amount:       lineTotal.String(),
        },
    }, nil
}
```

**Note**: LineTotal is **computed**, not user-provided.

---

## State Transition Rules

### INV-08: Valid State Transitions Only

**Rule**: Orders can only transition through valid state paths (see state machine).

**Rationale**: Prevents invalid order lifecycle progressions (e.g., SHIPPED → PENDING_VALIDATION).

**Valid Transitions**:

| From | To | Command |
|------|-----|---------|
| PENDING_VALIDATION | CONFIRMED | `Confirm()` |
| PENDING_VALIDATION | CANCELLED | `Cancel()` |
| CONFIRMED | PARTIALLY_SHIPPED | `MarkPartiallyShipped()` |
| CONFIRMED | SHIPPED | `MarkShipped()` |
| CONFIRMED | CANCELLED | `Cancel()` |
| CONFIRMED | UNFULFILLABLE | `MarkUnfulfillable()` |
| PARTIALLY_SHIPPED | SHIPPED | `MarkShipped()` |
| SHIPPED | DELIVERED | `MarkDelivered()` |
| DELIVERED | COMPLETED | `Complete()` |

**Enforcement**:
```go
func (o *Order) Confirm() error {
    if o.Status != StatusPendingValidation {
        return ErrInvalidTransition{
            From:      o.Status,
            To:        StatusConfirmed,
            Operation: "Confirm",
        }
    }
    // ...
}
```

**Error Response**:
```json
{
  "type": "https://api.example.com/problems/invalid-state-transition",
  "title": "Invalid state transition",
  "status": 409,
  "detail": "Cannot transition from SHIPPED to PENDING_VALIDATION via Confirm()"
}
```

---

### INV-09: Terminal State Immutability

**Rule**: Orders in terminal states (`CANCELLED`, `UNFULFILLABLE`, `COMPLETED`) cannot transition to other states.

**Rationale**: Terminal states are final — the order lifecycle is complete.

**Enforcement**:
```go
func (o *Order) isTerminalState() bool {
    return o.Status == StatusCancelled ||
           o.Status == StatusUnfulfillable ||
           o.Status == StatusCompleted
}

func (o *Order) Confirm() error {
    if o.isTerminalState() {
        return ErrTerminalState
    }
    // ...
}
```

**Error Response**:
```json
{
  "type": "https://api.example.com/problems/terminal-state",
  "title": "Order in terminal state",
  "status": 409,
  "detail": "Cannot modify order in terminal state: CANCELLED"
}
```

---

## Value Object Invariants

### INV-10: Valid Money

**Rule**: Money amounts must be valid decimal strings with valid ISO 4217 currency codes.

**Enforcement**:
```go
func NewMoney(amount string, currencyCode string) (Money, error) {
    // Validate decimal
    _, err := decimal.NewFromString(amount)
    if err != nil {
        return Money{}, ErrInvalidAmount
    }

    // Validate currency code (3 letters)
    if len(currencyCode) != 3 {
        return Money{}, ErrInvalidCurrencyCode
    }

    return Money{
        Amount:       amount,
        CurrencyCode: strings.ToUpper(currencyCode),
    }, nil
}
```

**Error Response**:
```json
{
  "type": "https://api.example.com/problems/invalid-money",
  "title": "Invalid money value",
  "status": 422,
  "detail": "Amount must be a valid decimal string. Got: 'abc'"
}
```

---

### INV-11: Valid Address

**Rule**: Addresses must have required fields (line1, city, postalCode, countryCode) and valid ISO 3166-1 alpha-2 country codes.

**Enforcement**:
```go
func NewAddress(line1, city, postalCode, countryCode string) (Address, error) {
    if line1 == "" || city == "" || postalCode == "" {
        return Address{}, ErrInvalidAddress
    }

    if len(countryCode) != 2 {
        return Address{}, ErrInvalidCountryCode
    }

    return Address{
        Line1:       line1,
        City:        city,
        PostalCode:  postalCode,
        CountryCode: strings.ToUpper(countryCode),
    }, nil
}
```

---

## Error Hierarchy

Domain errors form a hierarchy:

```
ErrDomain (base)
├── ErrNoLineItems
├── ErrMixedCurrencies
├── ErrInvalidQuantity
├── ErrOrderImmutable
├── ErrInvalidTransition
├── ErrTerminalState
├── ErrInvalidAmount
└── ErrInvalidAddress
```

All domain errors:
- Implement `error` interface
- Include context (field names, values)
- Map to HTTP status codes (422 for validation, 409 for conflicts)

---

## Testing Invariants

### Unit Test Example

```go
func TestOrder_CannotAddLineAfterConfirmation(t *testing.T) {
    // Arrange
    order := fixtures.ConfirmedOrder()
    line := fixtures.ValidOrderLine()

    // Act
    err := order.AddLine(line)

    // Assert
    assert.Error(t, err)
    assert.ErrorIs(t, err, ErrOrderImmutable)
}

func TestOrder_TotalMustEqualSumOfLines(t *testing.T) {
    // Arrange
    order := fixtures.OrderWithLines(
        fixtures.Line("SKU-001", 2, "10.00", "USD"),
        fixtures.Line("SKU-002", 1, "5.00", "USD"),
    )

    // Act
    order.RecalculateTotal()

    // Assert
    assert.Equal(t, "25.00", order.OrderTotal.Amount)  // Note: shopspring/decimal strips trailing zeros
    assert.Equal(t, "USD", order.OrderTotal.CurrencyCode)
}
```

---

## Invariant Enforcement Strategy

| Layer | Enforcement Method | Examples |
|-------|-------------------|----------|
| **Domain** | Constructor validation | `NewOrder()`, `NewOrderLine()` |
| **Domain** | Method preconditions | `Confirm()`, `Cancel()` |
| **Database** | Constraints | `UNIQUE`, `CHECK`, `NOT NULL`, `FK` |
| **API** | Input validation | OpenAPI schema validation |
| **Integration Tests** | Aggregate invariant tests | 27 domain tests |

**Defense in Depth**: Multiple layers enforce the same rules to prevent bugs at different stages.

---

## Next Steps

- [Data Model](data-model) - Aggregate structure
- [State Machine](../architecture/state-machine) - Valid transitions
- [API Reference](../api/v1/reference.html) - Validation rules in API
