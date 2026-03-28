// Package postgres implements the order.Repository port using pgx and PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oms/internal/orderintake/domain/order"
	"github.com/oms/pkg/money"
	"github.com/oms/pkg/pagination"
)

// uniqueViolationCode is the PostgreSQL error code for unique constraint violations.
const uniqueViolationCode = "23505"

// OrderRepository implements order.Repository backed by PostgreSQL.
type OrderRepository struct {
	pool *pgxpool.Pool
}

// NewOrderRepository creates a new OrderRepository backed by the given connection pool.
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

// Save persists a new order aggregate within a transaction. It inserts the order
// row followed by all order line rows. If the idempotency_key already exists,
// ErrDuplicateIdempotencyKey is returned.
func (r *OrderRepository) Save(ctx context.Context, o *order.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback on committed tx is a no-op

	if err := r.insertOrder(ctx, tx, o); err != nil {
		return err
	}

	for _, line := range o.Lines {
		if err := r.insertOrderLine(ctx, tx, o.ID, line); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit tx: %w", err)
	}
	return nil
}

// FindByID retrieves a single order by its UUID, including all order lines.
// Returns order.ErrOrderNotFound if no matching row exists.
func (r *OrderRepository) FindByID(ctx context.Context, id string) (*order.Order, error) {
	return r.findOrderByColumn(ctx, "id", id)
}

// FindByIdempotencyKey retrieves an order by its idempotency key.
// Returns nil, nil if no matching row exists (per the Repository contract).
func (r *OrderRepository) FindByIdempotencyKey(ctx context.Context, key string) (*order.Order, error) {
	o, err := r.findOrderByColumn(ctx, "idempotency_key", key)
	if err != nil {
		if errors.Is(err, order.ErrOrderNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return o, nil
}

// Update persists mutations on an existing order aggregate. It updates the
// status, confirmed_at, cancelled_at, cancellation_reason, and updated_at
// columns. Lines are immutable after creation so they are not touched.
func (r *OrderRepository) Update(ctx context.Context, o *order.Order) error {
	const query = `
		UPDATE orders
		SET status              = $2,
		    confirmed_at        = $3,
		    cancelled_at        = $4,
		    cancellation_reason = $5,
		    updated_at          = $6
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query,
		o.ID,
		string(o.Status),
		o.ConfirmedAt,
		o.CancelledAt,
		nullableString(o.CancellationReason),
		o.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("postgres: update order %s: %w", o.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return order.ErrOrderNotFound
	}
	return nil
}

// List retrieves orders matching the given filter using cursor-based pagination.
// It fetches limit+1 rows to determine whether more pages exist. The returned
// cursor string is a base64-encoded UUID suitable for the next page request.
func (r *OrderRepository) List(ctx context.Context, filter order.ListFilter) ([]*order.Order, string, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = pagination.DefaultLimit
	}
	if limit > pagination.MaxLimit {
		limit = pagination.MaxLimit
	}

	// Build the query dynamically. We use parameterised placeholders throughout
	// to prevent SQL injection; only the WHERE clauses are conditional.
	query := `
		SELECT
			id, idempotency_key, channel, external_id,
			customer_name, customer_email, customer_phone,
			shipping_line1, shipping_line2, shipping_city, shipping_state, shipping_postal, shipping_country,
			billing_line1, billing_line2, billing_city, billing_state, billing_postal, billing_country,
			currency_code, order_total, status, placed_at, confirmed_at, cancelled_at, cancellation_reason,
			created_at, updated_at
		FROM orders
		WHERE 1=1`

	args := make([]any, 0, 4)
	argIdx := 1

	if filter.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(filter.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("postgres: invalid cursor: %w", err)
		}
		query += fmt.Sprintf(" AND id < $%d", argIdx)
		args = append(args, cursorID)
		argIdx++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, string(*filter.Status))
		argIdx++
	}

	if filter.Channel != nil {
		query += fmt.Sprintf(" AND channel = $%d", argIdx)
		args = append(args, string(*filter.Channel))
		argIdx++
	}

	// Fetch limit+1 to detect next page.
	fetchLimit := limit + 1
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT $%d", argIdx)
	args = append(args, fetchLimit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("postgres: list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]*order.Order, 0, limit)
	for rows.Next() {
		o, err := scanOrderRow(rows)
		if err != nil {
			return nil, "", fmt.Errorf("postgres: scan order row: %w", err)
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("postgres: iterate order rows: %w", err)
	}

	// Determine next cursor.
	var nextCursor string
	if len(orders) > limit {
		// Trim the extra row and encode the last visible row's ID as cursor.
		orders = orders[:limit]
		nextCursor = pagination.EncodeCursor(orders[limit-1].ID)
	}

	// Load lines for each order. We batch this into a single query using IN.
	if len(orders) > 0 {
		if err := r.loadLines(ctx, orders); err != nil {
			return nil, "", err
		}
	}

	return orders, nextCursor, nil
}

// --------------------------------------------------------------------------
// Private helpers
// --------------------------------------------------------------------------

func (r *OrderRepository) insertOrder(ctx context.Context, tx pgx.Tx, o *order.Order) error {
	const query = `
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
		)`

	_, err := tx.Exec(ctx, query,
		o.ID,
		o.IdempotencyKey,
		string(o.Channel),
		nullableString(o.ExternalID),
		o.Customer.Name,
		o.Customer.Email,
		nullableString(o.Customer.Phone),
		o.ShippingAddress.Line1,
		nullableString(o.ShippingAddress.Line2),
		o.ShippingAddress.City,
		nullableString(o.ShippingAddress.StateOrRegion),
		o.ShippingAddress.PostalCode,
		o.ShippingAddress.CountryCode,
		o.BillingAddress.Line1,
		nullableString(o.BillingAddress.Line2),
		o.BillingAddress.City,
		nullableString(o.BillingAddress.StateOrRegion),
		o.BillingAddress.PostalCode,
		o.BillingAddress.CountryCode,
		o.CurrencyCode,
		o.OrderTotal.Amount,
		string(o.Status),
		o.PlacedAt,
		o.ConfirmedAt,
		o.CancelledAt,
		nullableString(o.CancellationReason),
		o.CreatedAt,
		o.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return order.ErrDuplicateIdempotencyKey
		}
		return fmt.Errorf("postgres: insert order %s: %w", o.ID, err)
	}
	return nil
}

func (r *OrderRepository) insertOrderLine(ctx context.Context, tx pgx.Tx, orderID string, line order.OrderLine) error {
	const query = `
		INSERT INTO order_lines (id, order_id, sku, product_name, quantity, unit_price, line_total, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := tx.Exec(ctx, query,
		line.LineID,
		orderID,
		line.SKU,
		line.ProductName,
		line.Quantity,
		line.UnitPrice.Amount,
		line.LineTotal.Amount,
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("postgres: insert order line %s for order %s: %w", line.LineID, orderID, err)
	}
	return nil
}

// findOrderByColumn retrieves an order by the given column (id or idempotency_key)
// with its associated order lines in a single query via LEFT JOIN.
func (r *OrderRepository) findOrderByColumn(ctx context.Context, column, value string) (*order.Order, error) {
	query := fmt.Sprintf(`
		SELECT
			o.id, o.idempotency_key, o.channel, o.external_id,
			o.customer_name, o.customer_email, o.customer_phone,
			o.shipping_line1, o.shipping_line2, o.shipping_city, o.shipping_state, o.shipping_postal, o.shipping_country,
			o.billing_line1, o.billing_line2, o.billing_city, o.billing_state, o.billing_postal, o.billing_country,
			o.currency_code, o.order_total, o.status, o.placed_at, o.confirmed_at, o.cancelled_at, o.cancellation_reason,
			o.created_at, o.updated_at,
			ol.id, ol.sku, ol.product_name, ol.quantity, ol.unit_price, ol.line_total
		FROM orders o
		LEFT JOIN order_lines ol ON ol.order_id = o.id
		WHERE o.%s = $1`, column)

	rows, err := r.pool.Query(ctx, query, value)
	if err != nil {
		return nil, fmt.Errorf("postgres: query order by %s: %w", column, err)
	}
	defer rows.Close()

	var o *order.Order
	for rows.Next() {
		if o == nil {
			o = &order.Order{}
		}

		var (
			externalID      *string
			customerPhone   *string
			shippingLine2   *string
			shippingState   *string
			billingLine2    *string
			billingState    *string
			confirmedAt     *time.Time
			cancelledAt     *time.Time
			cancelReason    *string
			orderTotalStr   string
			statusStr       string
			channelStr      string
			lineID          *string
			lineSKU         *string
			lineProductName *string
			lineQty         *int
			lineUnitPrice   *string
			lineLineTotal   *string
		)

		err := rows.Scan(
			&o.ID, &o.IdempotencyKey, &channelStr, &externalID,
			&o.Customer.Name, &o.Customer.Email, &customerPhone,
			&o.ShippingAddress.Line1, &shippingLine2, &o.ShippingAddress.City,
			&shippingState, &o.ShippingAddress.PostalCode, &o.ShippingAddress.CountryCode,
			&o.BillingAddress.Line1, &billingLine2, &o.BillingAddress.City,
			&billingState, &o.BillingAddress.PostalCode, &o.BillingAddress.CountryCode,
			&o.CurrencyCode, &orderTotalStr, &statusStr,
			&o.PlacedAt, &confirmedAt, &cancelledAt, &cancelReason,
			&o.CreatedAt, &o.UpdatedAt,
			&lineID, &lineSKU, &lineProductName, &lineQty, &lineUnitPrice, &lineLineTotal,
		)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan order+line row: %w", err)
		}

		// Populate nullable order-level fields (only need to set once but safe to overwrite).
		o.Channel = order.Channel(channelStr)
		o.Status = order.OrderStatus(statusStr)
		o.ExternalID = derefString(externalID)
		o.Customer.Phone = derefString(customerPhone)
		o.ShippingAddress.Line2 = derefString(shippingLine2)
		o.ShippingAddress.StateOrRegion = derefString(shippingState)
		o.BillingAddress.Line2 = derefString(billingLine2)
		o.BillingAddress.StateOrRegion = derefString(billingState)
		o.ConfirmedAt = confirmedAt
		o.CancelledAt = cancelledAt
		o.CancellationReason = derefString(cancelReason)

		// Reconstruct money.Money for order total.
		o.OrderTotal = money.Money{
			CurrencyCode: o.CurrencyCode,
			Amount:       orderTotalStr,
		}

		// Append order line if present (LEFT JOIN can yield NULLs).
		if lineID != nil {
			ol := order.OrderLine{
				LineID:      *lineID,
				SKU:         derefString(lineSKU),
				ProductName: derefString(lineProductName),
				Quantity:    derefInt(lineQty),
				UnitPrice: money.Money{
					CurrencyCode: o.CurrencyCode,
					Amount:       derefString(lineUnitPrice),
				},
				LineTotal: money.Money{
					CurrencyCode: o.CurrencyCode,
					Amount:       derefString(lineLineTotal),
				},
			}
			o.Lines = append(o.Lines, ol)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate rows: %w", err)
	}

	if o == nil {
		return nil, order.ErrOrderNotFound
	}

	return o, nil
}

// loadLines fetches order lines for a batch of orders and attaches them to
// the respective order structs. This avoids N+1 queries in the List method.
func (r *OrderRepository) loadLines(ctx context.Context, orders []*order.Order) error {
	ids := make([]string, len(orders))
	orderMap := make(map[string]*order.Order, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
		orderMap[o.ID] = o
	}

	const query = `
		SELECT id, order_id, sku, product_name, quantity, unit_price, line_total
		FROM order_lines
		WHERE order_id = ANY($1)
		ORDER BY order_id, id`

	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("postgres: load order lines: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			lineID      string
			orderID     string
			sku         string
			productName string
			qty         int
			unitPrice   string
			lineTotal   string
		)
		if err := rows.Scan(&lineID, &orderID, &sku, &productName, &qty, &unitPrice, &lineTotal); err != nil {
			return fmt.Errorf("postgres: scan order line: %w", err)
		}

		o, ok := orderMap[orderID]
		if !ok {
			continue
		}

		o.Lines = append(o.Lines, order.OrderLine{
			LineID:      lineID,
			SKU:         sku,
			ProductName: productName,
			Quantity:    qty,
			UnitPrice: money.Money{
				CurrencyCode: o.CurrencyCode,
				Amount:       unitPrice,
			},
			LineTotal: money.Money{
				CurrencyCode: o.CurrencyCode,
				Amount:       lineTotal,
			},
		})
	}

	return rows.Err()
}

// scanOrderRow scans a single order row (without lines) from a pgx.Rows cursor.
func scanOrderRow(rows pgx.Rows) (*order.Order, error) {
	o := &order.Order{}

	var (
		externalID    *string
		customerPhone *string
		shippingLine2 *string
		shippingState *string
		billingLine2  *string
		billingState  *string
		confirmedAt   *time.Time
		cancelledAt   *time.Time
		cancelReason  *string
		orderTotalStr string
		statusStr     string
		channelStr    string
	)

	err := rows.Scan(
		&o.ID, &o.IdempotencyKey, &channelStr, &externalID,
		&o.Customer.Name, &o.Customer.Email, &customerPhone,
		&o.ShippingAddress.Line1, &shippingLine2, &o.ShippingAddress.City,
		&shippingState, &o.ShippingAddress.PostalCode, &o.ShippingAddress.CountryCode,
		&o.BillingAddress.Line1, &billingLine2, &o.BillingAddress.City,
		&billingState, &o.BillingAddress.PostalCode, &o.BillingAddress.CountryCode,
		&o.CurrencyCode, &orderTotalStr, &statusStr,
		&o.PlacedAt, &confirmedAt, &cancelledAt, &cancelReason,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	o.Channel = order.Channel(channelStr)
	o.Status = order.OrderStatus(statusStr)
	o.ExternalID = derefString(externalID)
	o.Customer.Phone = derefString(customerPhone)
	o.ShippingAddress.Line2 = derefString(shippingLine2)
	o.ShippingAddress.StateOrRegion = derefString(shippingState)
	o.BillingAddress.Line2 = derefString(billingLine2)
	o.BillingAddress.StateOrRegion = derefString(billingState)
	o.ConfirmedAt = confirmedAt
	o.CancelledAt = cancelledAt
	o.CancellationReason = derefString(cancelReason)
	o.OrderTotal = money.Money{
		CurrencyCode: o.CurrencyCode,
		Amount:       orderTotalStr,
	}

	return o, nil
}

// nullableString returns a *string that is nil if s is empty, enabling clean
// NULL storage in Postgres for optional text columns.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// derefString safely dereferences a *string, returning "" for nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// derefInt safely dereferences an *int, returning 0 for nil.
func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
