package query

import (
	"context"
	"fmt"

	"github.com/oms/internal/orderintake/domain/order"
	"github.com/oms/pkg/pagination"
)

// ListOrdersQuery carries the filtering and pagination parameters for listing
// orders.
type ListOrdersQuery struct {
	Status  *order.OrderStatus
	Channel *order.Channel
	Cursor  string
	Limit   int
}

// ListOrdersResult holds the paginated result of a list orders query.
type ListOrdersResult struct {
	Orders     []*order.Order
	NextCursor string
	HasMore    bool
}

// ListOrdersHandler handles queries for listing orders with filtering and
// cursor-based pagination.
type ListOrdersHandler struct {
	repo order.Repository
}

// NewListOrdersHandler returns a handler wired with the given repository.
func NewListOrdersHandler(repo order.Repository) *ListOrdersHandler {
	return &ListOrdersHandler{
		repo: repo,
	}
}

// Handle lists orders matching the query filters. The limit is clamped between
// the default (20) and maximum (100) values. An extra record is fetched to
// determine whether more results exist beyond the current page.
func (h *ListOrdersHandler) Handle(ctx context.Context, q ListOrdersQuery) (ListOrdersResult, error) {
	page := pagination.NewPage(q.Cursor, q.Limit)

	filter := order.ListFilter{
		Status:  q.Status,
		Channel: q.Channel,
		Cursor:  page.Cursor,
		Limit:   page.Limit + 1, // fetch one extra to detect next page
	}

	orders, nextCursor, err := h.repo.List(ctx, filter)
	if err != nil {
		return ListOrdersResult{}, fmt.Errorf("listing orders: %w", err)
	}

	hasMore := len(orders) > page.Limit
	if hasMore {
		orders = orders[:page.Limit]
	}

	// Only set NextCursor when there are more results. If the repository
	// already returned a cursor, use it; otherwise derive one from the last
	// returned order.
	cursor := ""
	if hasMore {
		if nextCursor != "" {
			cursor = nextCursor
		} else if len(orders) > 0 {
			cursor = pagination.EncodeCursor(orders[len(orders)-1].ID)
		}
	}

	return ListOrdersResult{
		Orders:     orders,
		NextCursor: cursor,
		HasMore:    hasMore,
	}, nil
}
