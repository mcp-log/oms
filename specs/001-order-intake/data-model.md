# Order Intake — Data Model

> **Spec Ref**: 001-order-intake
> **Derived from**: spec.md, plan.md

---

## Order Aggregate

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
│   ├── Quantity       : int (> 0)
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

## Value Objects

### Money
```
Money
├── CurrencyCode : string (ISO 4217, e.g., "USD")
└── Amount       : string (decimal, e.g., "29.99")
```

### Address
```
Address
├── Line1        : string (required)
├── Line2        : string (optional)
├── City         : string (required)
├── StateOrRegion: string (optional)
├── PostalCode   : string (required)
└── CountryCode  : string (ISO 3166-1 alpha-2, required)
```

## Aggregate Invariants

1. **Minimum one line item**: An order must have at least 1 line
2. **Single currency**: All line items must share the same currency code
3. **Total consistency**: OrderTotal = sum(line.LineTotal for each line)
4. **Line total consistency**: LineTotal = Quantity * UnitPrice
5. **State machine enforcement**: Only valid transitions are allowed
6. **Immutability after CONFIRMED**: Core data (lines, customer, addresses) cannot change

## State Machine

```
                    ┌──────────┐
        ┌──────────│  PENDING  │──────────┐
        │  cancel  │VALIDATION │  confirm  │
        ▼          └──────────┘           ▼
  ┌───────────┐                    ┌───────────┐
  │ CANCELLED │◄───── cancel ──────│ CONFIRMED │
  └───────────┘                    └─────┬─────┘
                                         │
                    ┌────────────────┬────┴────────────────┐
                    │ partial_ship   │ ship_all             │ unfulfillable
                    ▼                ▼                      ▼
            ┌──────────────┐  ┌──────────┐         ┌──────────────┐
            │  PARTIALLY   │  │ SHIPPED  │         │UNFULFILLABLE │
            │  SHIPPED     │  └────┬─────┘         └──────────────┘
            └──────┬───────┘       │
                   │ all_shipped   │ delivered
                   ▼               ▼
            ┌──────────┐    ┌───────────┐
            │ SHIPPED  │    │ DELIVERED │
            └────┬─────┘    └─────┬─────┘
                 │ delivered      │ complete
                 ▼                ▼
          ┌───────────┐   ┌───────────┐
          │ DELIVERED │   │ COMPLETED │
          └─────┬─────┘   └───────────┘
                │ complete
                ▼
          ┌───────────┐
          │ COMPLETED │
          └───────────┘
```

**Terminal states**: CANCELLED, UNFULFILLABLE, COMPLETED

## Valid State Transitions

| From | To | Trigger | Event |
|------|----|---------|-------|
| PENDING_VALIDATION | CONFIRMED | confirm | order.confirmed |
| PENDING_VALIDATION | CANCELLED | cancel | order.cancelled |
| CONFIRMED | PARTIALLY_SHIPPED | partial_ship | order.status_changed |
| CONFIRMED | SHIPPED | ship_all | order.shipped |
| CONFIRMED | CANCELLED | cancel | order.cancelled |
| CONFIRMED | UNFULFILLABLE | unfulfillable | order.status_changed |
| PARTIALLY_SHIPPED | SHIPPED | all_shipped | order.shipped |
| SHIPPED | DELIVERED | delivered | order.delivered |
| DELIVERED | COMPLETED | complete | order.status_changed |

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
