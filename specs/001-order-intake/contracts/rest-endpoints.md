# Order Intake — REST API Contracts

> **Spec Ref**: 001-order-intake
> **Base Path**: `/v1`

---

## Endpoints

| Method | Path | OperationId | Spec Ref | Description |
|--------|------|-------------|----------|-------------|
| POST | /v1/orders | createOrder | CAP-01 | Create a new order (idempotent) |
| GET | /v1/orders | listOrders | CAP-04 | List orders with filtering + cursor pagination |
| GET | /v1/orders/{orderId} | getOrderById | CAP-04 | Get single order with lines |
| GET | /v1/orders/{orderId}/lines | listOrderLines | CAP-04 | Get order line items |
| POST | /v1/orders/{orderId}/confirm | confirmOrder | CAP-02 | Confirm an order |
| POST | /v1/orders/{orderId}/cancel | cancelOrder | CAP-03 | Cancel an order |
| POST | /v1/webhooks/shopify | handleShopifyWebhook | CAP-05 | Receive Shopify webhook |

---

## Request/Response Details

### POST /v1/orders (createOrder)

**Headers:**
- `Idempotency-Key` (required): Unique key for idempotent creation
- `Content-Type: application/json`

**Request Body:**
```json
{
  "channel": "ECOMMERCE",
  "externalId": "shopify-order-12345",
  "customer": {
    "name": "Jane Doe",
    "email": "jane@example.com",
    "phone": "+1-555-0123"
  },
  "shippingAddress": {
    "line1": "123 Main St",
    "city": "Portland",
    "stateOrRegion": "OR",
    "postalCode": "97201",
    "countryCode": "US"
  },
  "billingAddress": {
    "line1": "123 Main St",
    "city": "Portland",
    "stateOrRegion": "OR",
    "postalCode": "97201",
    "countryCode": "US"
  },
  "lines": [
    {
      "sku": "WIDGET-001",
      "productName": "Blue Widget",
      "quantity": 2,
      "unitPrice": { "currencyCode": "USD", "amount": "29.99" }
    }
  ],
  "placedAt": "2024-01-15T10:30:00Z"
}
```

**Response 201 Created:**
```json
{
  "id": "01912345-6789-7abc-def0-123456789abc",
  "status": "PENDING_VALIDATION",
  "channel": "ECOMMERCE",
  "orderTotal": { "currencyCode": "USD", "amount": "59.98" },
  "placedAt": "2024-01-15T10:30:00Z",
  "createdAt": "2024-01-15T10:30:01Z"
}
```

**Response 200 OK** (duplicate Idempotency-Key): Returns existing order.

**Response 422:** RFC 7807 Problem Details for validation errors.

---

### GET /v1/orders (listOrders)

**Query Parameters:**
- `status` (optional): Filter by order status
- `channel` (optional): Filter by channel
- `cursor` (optional): Pagination cursor (opaque base64)
- `limit` (optional, default 20, max 100): Page size

**Response 200:**
```json
{
  "data": [ /* order summaries */ ],
  "pagination": {
    "nextCursor": "base64encodedcursor",
    "hasMore": true
  }
}
```

---

### GET /v1/orders/{orderId} (getOrderById)

**Response 200:** Full order with all line items.

**Response 404:** RFC 7807 when order not found.

---

### POST /v1/orders/{orderId}/confirm (confirmOrder)

**Response 200:** Updated order with status CONFIRMED.

**Response 409:** RFC 7807 when transition not allowed.

---

### POST /v1/orders/{orderId}/cancel (cancelOrder)

**Request Body:**
```json
{
  "reason": "Customer requested cancellation"
}
```

**Response 200:** Updated order with status CANCELLED.

**Response 409:** RFC 7807 when transition not allowed.

---

### POST /v1/webhooks/shopify (handleShopifyWebhook)

**Headers:**
- `X-Shopify-Hmac-Sha256` (required): HMAC signature

**Response 200:** Acknowledgement.

**Response 401:** Invalid signature.
