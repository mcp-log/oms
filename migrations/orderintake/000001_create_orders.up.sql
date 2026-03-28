CREATE TYPE order_status AS ENUM (
    'PENDING_VALIDATION', 'CONFIRMED', 'PARTIALLY_SHIPPED', 'SHIPPED',
    'UNFULFILLABLE', 'DELIVERED', 'CANCELLED', 'COMPLETED'
);

CREATE TYPE channel AS ENUM (
    'ECOMMERCE', 'MARKETPLACE', 'B2B', 'DIRECT'
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
