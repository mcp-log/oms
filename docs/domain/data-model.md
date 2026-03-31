---
layout: default
title: Data Model
parent: Domain
nav_order: 1
---

# Domain Data Model
{: .no_toc }

Complete reference for the Order aggregate, value objects, and invariants.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Order Aggregate

The **Order** is the aggregate root that encapsulates all order-related data and behavior.

```
Order (Aggregate Root)
├── OrderID           : UUID v7 (PK)
├── IdempotencyKey    : string (UNIQUE)
├── ExternalReference
│   ├── Channel       : enum [ECOMMERCE, MARKETPLACE, B2B, DIRECT]
│   └── ExternalID    : string
├── Customer
│   ├── Name          : string (required)
│   ├── Email         : string (required)
│   └── Phone         : string (optional)
├── ShippingAddress   : Address (VO)
├── BillingAddress    : Address (VO)
├── Lines[]           : OrderLine[]
│   ├── LineID        : UUID v7
│   ├── SKU           : string
│   ├── ProductName   : string
│   ├── Quantity      : int (> 0)
│   ├── UnitPrice     : Money (VO)
│   └── LineTotal     : Money (computed: quantity * unitPrice)
├── CurrencyCode      : string (ISO 4217, derived from lines)
├── OrderTotal        : Money (computed: sum of lineTotals)
├── Status            : OrderStatus (state machine)
├── PlacedAt          : timestamp
├── ConfirmedAt       : timestamp (nullable)
├── CancelledAt       : timestamp (nullable)
├── CancellationReason: string (nullable)
├── CreatedAt         : timestamp
└── UpdatedAt         : timestamp
```

---

## Value Objects

### Money

Represents a monetary amount with currency. **Always stored as decimal string**, never float.

```go
type Money struct {
    CurrencyCode string  // ISO 4217 (e.g., "USD", "EUR")
    Amount       string  // Decimal string (e.g., "29.99")
}
```

**Example**:
```json
{
  "currencyCode": "USD",
  "amount": "29.99"
}
```

**Rules**:
- Amount must be a valid decimal string: `"29.99"`, `"0"`, `"1000"`
- CurrencyCode must be 3-letter ISO 4217 code
- No negative amounts (use separate fields for refunds)

### Address

Represents a physical address (shipping or billing).

```go
type Address struct {
    Line1        string  // Required: Street address
    Line2        string  // Optional: Apt, Suite, etc.
    City         string  // Required
    StateOrRegion string // Optional: State/Province
    PostalCode   string  // Required
    CountryCode  string  // Required: ISO 3166-1 alpha-2 (e.g., "US")
}
```

**Example**:
```json
{
  "line1": "123 Main St",
  "line2": "Apt 4B",
  "city": "San Francisco",
  "stateOrRegion": "CA",
  "postalCode": "94105",
  "countryCode": "US"
}
```

### Customer

Represents customer contact information (not a full user entity).

```go
type Customer struct {
    Name  string  // Required: Full name
    Email string  // Required: Email address
    Phone string  // Optional: Phone number
}
```

**Example**:
```json
{
  "name": "Jane Doe",
  "email": "jane@example.com",
  "phone": "+1-555-0199"
}
```

---

## Entities

### OrderLine

Represents a single line item in an order. **Not an aggregate root** — always accessed through Order.

```go
type OrderLine struct {
    LineID      uuid.UUID  // UUID v7
    SKU         string     // Product SKU
    ProductName string     // Product display name
    Quantity    int        // Must be > 0
    UnitPrice   Money      // Price per unit
    LineTotal   Money      // Computed: Quantity * UnitPrice
}
```

**Invariants**:
- Quantity must be > 0
- LineTotal = Quantity × UnitPrice (enforced in constructor)
- SKU and ProductName are immutable after order is confirmed

---

## Enumerations

### OrderStatus

State machine states for order lifecycle:

```go
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
```

**Terminal States**: `CANCELLED`, `UNFULFILLABLE`, `COMPLETED`

See [State Machine](../architecture/state-machine) for transition diagram.

### Channel

Source of the order:

```go
const (
    ChannelEcommerce   Channel = "ECOMMERCE"     // Website/mobile app
    ChannelMarketplace Channel = "MARKETPLACE"   // Amazon, eBay, etc.
    ChannelB2B         Channel = "B2B"           // Business partner portal
    ChannelDirect      Channel = "DIRECT"        // Phone/email orders
)
```

---

## Aggregate Invariants

Invariants are **always enforced** by the aggregate root:

### 1. Minimum One Line Item
```go
if len(order.Lines) == 0 {
    return ErrNoLineItems
}
```

An order must have at least one line item.

### 2. Single Currency

```go
for _, line := range order.Lines {
    if line.UnitPrice.CurrencyCode != order.CurrencyCode {
        return ErrMixedCurrencies
    }
}
```

All line items must use the same currency.

### 3. Total Consistency

```go
func (o *Order) RecalculateTotal() {
    total := decimal.Zero
    for _, line := range o.Lines {
        total = total.Add(line.LineTotal.ToDecimal())
    }
    o.OrderTotal = Money{
        CurrencyCode: o.CurrencyCode,
        Amount:       total.String(),
    }
}
```

OrderTotal = sum of all LineTotals (computed, not stored).

### 4. Line Total Consistency

```go
func NewOrderLine(sku, name string, qty int, unitPrice Money) (*OrderLine, error) {
    lineTotal := unitPrice.ToDecimal().Mul(decimal.NewFromInt(int64(qty)))

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

LineTotal = Quantity × UnitPrice.

### 5. State Machine Enforcement

```go
func (o *Order) Confirm() error {
    if o.Status != StatusPendingValidation {
        return ErrInvalidTransition
    }
    o.Status = StatusConfirmed
    o.ConfirmedAt = time.Now()
    return nil
}
```

Only valid transitions are allowed (see state machine).

### 6. Immutability After CONFIRMED

```go
func (o *Order) AddLine(line *OrderLine) error {
    if o.Status != StatusPendingValidation {
        return ErrOrderImmutable
    }
    o.Lines = append(o.Lines, line)
    return nil
}
```

Core data (lines, customer, addresses) cannot change after confirmation.

---

## PostgreSQL Schema

```sql
CREATE TYPE order_status AS ENUM (
    'PENDING_VALIDATION',
    'CONFIRMED',
    'PARTIALLY_SHIPPED',
    'SHIPPED',
    'UNFULFILLABLE',
    'DELIVERED',
    'CANCELLED',
    'COMPLETED'
);

CREATE TYPE channel AS ENUM (
    'ECOMMERCE',
    'MARKETPLACE',
    'B2B',
    'DIRECT'
);

CREATE TABLE orders (
    id              UUID PRIMARY KEY,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    channel         channel NOT NULL,
    external_id     VARCHAR(255),
    customer_name   VARCHAR(255) NOT NULL,
    customer_email  VARCHAR(255) NOT NULL,
    customer_phone  VARCHAR(50),
    shipping_line1  VARCHAR(255) NOT NULL,
    shipping_line2  VARCHAR(255),
    shipping_city   VARCHAR(255) NOT NULL,
    shipping_state  VARCHAR(100),
    shipping_postal VARCHAR(20) NOT NULL,
    shipping_country CHAR(2) NOT NULL,
    billing_line1   VARCHAR(255) NOT NULL,
    billing_line2   VARCHAR(255),
    billing_city    VARCHAR(255) NOT NULL,
    billing_state   VARCHAR(100),
    billing_postal  VARCHAR(20) NOT NULL,
    billing_country CHAR(2) NOT NULL,
    currency_code   CHAR(3) NOT NULL,
    order_total     NUMERIC(19,4) NOT NULL,
    status          order_status NOT NULL DEFAULT 'PENDING_VALIDATION',
    placed_at       TIMESTAMPTZ NOT NULL,
    confirmed_at    TIMESTAMPTZ,
    cancelled_at    TIMESTAMPTZ,
    cancellation_reason TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE order_lines (
    id           UUID PRIMARY KEY,
    order_id     UUID NOT NULL REFERENCES orders(id),
    sku          VARCHAR(100) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    quantity     INTEGER NOT NULL CHECK (quantity > 0),
    unit_price   NUMERIC(19,4) NOT NULL,
    line_total   NUMERIC(19,4) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_channel ON orders(channel);
CREATE INDEX idx_orders_placed_at ON orders(placed_at);
CREATE INDEX idx_order_lines_order_id ON order_lines(order_id);
```

**Key Design Decisions**:
- **UUID v7**: Time-sortable, avoids DB round-trip for ID generation
- **NUMERIC(19,4)**: Exact decimal for money (never float)
- **Flattened value objects**: Address and Customer fields directly on orders table (no normalization for aggregates)
- **Separate lines table**: 1-to-many relationship with FK constraint
- **TIMESTAMPTZ**: Always store with timezone

---

## Go Implementation

### Order Aggregate

```go
type Order struct {
    ID                 uuid.UUID
    IdempotencyKey     string
    Channel            Channel
    ExternalID         string
    Customer           Customer
    ShippingAddress    Address
    BillingAddress     Address
    Lines              []*OrderLine
    CurrencyCode       string
    OrderTotal         Money
    Status             OrderStatus
    PlacedAt           time.Time
    ConfirmedAt        *time.Time
    CancelledAt        *time.Time
    CancellationReason string
    CreatedAt          time.Time
    UpdatedAt          time.Time
}

// Factory method
func NewOrder(cmd CreateOrderCommand) (*Order, error) {
    // Validation
    if len(cmd.Lines) == 0 {
        return nil, ErrNoLineItems
    }

    // Build lines
    lines := make([]*OrderLine, len(cmd.Lines))
    for i, lineCmd := range cmd.Lines {
        line, err := NewOrderLine(lineCmd.SKU, lineCmd.ProductName, lineCmd.Quantity, lineCmd.UnitPrice)
        if err != nil {
            return nil, err
        }
        lines[i] = line
    }

    // Create aggregate
    order := &Order{
        ID:              identity.NewUUID(),
        IdempotencyKey:  cmd.IdempotencyKey,
        Channel:         cmd.Channel,
        ExternalID:      cmd.ExternalID,
        Customer:        cmd.Customer,
        ShippingAddress: cmd.ShippingAddress,
        BillingAddress:  cmd.BillingAddress,
        Lines:           lines,
        CurrencyCode:    lines[0].UnitPrice.CurrencyCode,
        Status:          StatusPendingValidation,
        PlacedAt:        cmd.PlacedAt,
        CreatedAt:       time.Now(),
        UpdatedAt:       time.Now(),
    }

    order.RecalculateTotal()
    return order, nil
}

// State transition methods
func (o *Order) Confirm() error {
    if o.Status != StatusPendingValidation {
        return ErrInvalidTransition
    }
    now := time.Now()
    o.Status = StatusConfirmed
    o.ConfirmedAt = &now
    o.UpdatedAt = now
    return nil
}

func (o *Order) Cancel(reason string) error {
    if o.Status == StatusCancelled || o.Status == StatusCompleted {
        return ErrInvalidTransition
    }
    now := time.Now()
    o.Status = StatusCancelled
    o.CancelledAt = &now
    o.CancellationReason = reason
    o.UpdatedAt = now
    return nil
}
```

---

## Next Steps

- [Invariants](invariants) - Business rules enforcement
- [State Machine](../architecture/state-machine) - Order lifecycle
- [API Reference](../api/v1/reference.html) - REST endpoints
