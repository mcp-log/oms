-- name: InsertOrder :exec
INSERT INTO orders (
    id, idempotency_key, channel, external_id,
    customer_name, customer_email, customer_phone,
    shipping_line1, shipping_line2, shipping_city, shipping_state, shipping_postal, shipping_country,
    billing_line1, billing_line2, billing_city, billing_state, billing_postal, billing_country,
    currency_code, order_total, status, placed_at, confirmed_at, cancelled_at, cancellation_reason,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7,
    $8, $9, $10, $11, $12, $13,
    $14, $15, $16, $17, $18, $19,
    $20, $21, $22, $23, $24, $25, $26,
    $27, $28
);

-- name: InsertOrderLine :exec
INSERT INTO order_lines (
    id, order_id, sku, product_name, quantity, unit_price, line_total, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: GetOrderByID :many
SELECT
    o.id, o.idempotency_key, o.channel, o.external_id,
    o.customer_name, o.customer_email, o.customer_phone,
    o.shipping_line1, o.shipping_line2, o.shipping_city, o.shipping_state, o.shipping_postal, o.shipping_country,
    o.billing_line1, o.billing_line2, o.billing_city, o.billing_state, o.billing_postal, o.billing_country,
    o.currency_code, o.order_total, o.status, o.placed_at, o.confirmed_at, o.cancelled_at, o.cancellation_reason,
    o.created_at, o.updated_at,
    ol.id AS line_id, ol.sku, ol.product_name, ol.quantity, ol.unit_price, ol.line_total, ol.created_at AS line_created_at
FROM orders o
LEFT JOIN order_lines ol ON ol.order_id = o.id
WHERE o.id = $1;

-- name: GetOrderByIdempotencyKey :many
SELECT
    o.id, o.idempotency_key, o.channel, o.external_id,
    o.customer_name, o.customer_email, o.customer_phone,
    o.shipping_line1, o.shipping_line2, o.shipping_city, o.shipping_state, o.shipping_postal, o.shipping_country,
    o.billing_line1, o.billing_line2, o.billing_city, o.billing_state, o.billing_postal, o.billing_country,
    o.currency_code, o.order_total, o.status, o.placed_at, o.confirmed_at, o.cancelled_at, o.cancellation_reason,
    o.created_at, o.updated_at,
    ol.id AS line_id, ol.sku, ol.product_name, ol.quantity, ol.unit_price, ol.line_total, ol.created_at AS line_created_at
FROM orders o
LEFT JOIN order_lines ol ON ol.order_id = o.id
WHERE o.idempotency_key = $1;

-- name: UpdateOrderStatus :exec
UPDATE orders
SET
    status              = $2,
    confirmed_at        = $3,
    cancelled_at        = $4,
    cancellation_reason = $5,
    updated_at          = $6
WHERE id = $1;

-- name: ListOrders :many
-- Cursor-based pagination: fetches limit+1 rows to determine hasMore.
-- Optional filters on status and channel are applied dynamically.
SELECT
    o.id, o.idempotency_key, o.channel, o.external_id,
    o.customer_name, o.customer_email, o.customer_phone,
    o.shipping_line1, o.shipping_line2, o.shipping_city, o.shipping_state, o.shipping_postal, o.shipping_country,
    o.billing_line1, o.billing_line2, o.billing_city, o.billing_state, o.billing_postal, o.billing_country,
    o.currency_code, o.order_total, o.status, o.placed_at, o.confirmed_at, o.cancelled_at, o.cancellation_reason,
    o.created_at, o.updated_at
FROM orders o
WHERE
    ($1::uuid IS NULL OR o.id < $1)
    AND ($2::order_status IS NULL OR o.status = $2)
    AND ($3::channel IS NULL OR o.channel = $3)
ORDER BY o.id DESC
LIMIT $4;
