package ports

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/oms/internal/orderintake/app/command"
	"github.com/oms/internal/orderintake/app/query"
	"github.com/oms/internal/orderintake/domain/order"
	pkgerrors "github.com/oms/pkg/errors"
	"github.com/oms/pkg/money"
)

// HTTPHandler implements the order intake HTTP API.
// When oapi-codegen is available, this will implement the generated ServerInterface.
type HTTPHandler struct {
	createOrder        *command.CreateOrderHandler
	confirmOrder       *command.ConfirmOrderHandler
	cancelOrder        *command.CancelOrderHandler
	markShipped        *command.MarkShippedHandler
	markPartialShipped *command.MarkPartiallyShippedHandler
	markDelivered      *command.MarkDeliveredHandler
	markUnfulfillable  *command.MarkUnfulfillableHandler
	markCompleted      *command.MarkCompletedHandler
	getOrder           *query.GetOrderHandler
	listOrders         *query.ListOrdersHandler
	logger             *slog.Logger
}

// NewHTTPHandler creates a new HTTP handler with all command/query handlers.
func NewHTTPHandler(
	createOrder *command.CreateOrderHandler,
	confirmOrder *command.ConfirmOrderHandler,
	cancelOrder *command.CancelOrderHandler,
	markShipped *command.MarkShippedHandler,
	markPartialShipped *command.MarkPartiallyShippedHandler,
	markDelivered *command.MarkDeliveredHandler,
	markUnfulfillable *command.MarkUnfulfillableHandler,
	markCompleted *command.MarkCompletedHandler,
	getOrder *query.GetOrderHandler,
	listOrders *query.ListOrdersHandler,
	logger *slog.Logger,
) *HTTPHandler {
	return &HTTPHandler{
		createOrder:        createOrder,
		confirmOrder:       confirmOrder,
		cancelOrder:        cancelOrder,
		markShipped:        markShipped,
		markPartialShipped: markPartialShipped,
		markDelivered:      markDelivered,
		markUnfulfillable:  markUnfulfillable,
		markCompleted:      markCompleted,
		getOrder:           getOrder,
		listOrders:         listOrders,
		logger:             logger,
	}
}

// --- Request/Response DTOs ---

type createOrderRequest struct {
	Channel         string             `json:"channel"`
	ExternalID      string             `json:"externalId,omitempty"`
	Customer        customerDTO        `json:"customer"`
	ShippingAddress addressDTO         `json:"shippingAddress"`
	BillingAddress  addressDTO         `json:"billingAddress"`
	Lines           []orderLineReqDTO  `json:"lines"`
	PlacedAt        time.Time          `json:"placedAt"`
}

type customerDTO struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone,omitempty"`
}

type addressDTO struct {
	Line1         string `json:"line1"`
	Line2         string `json:"line2,omitempty"`
	City          string `json:"city"`
	StateOrRegion string `json:"stateOrRegion,omitempty"`
	PostalCode    string `json:"postalCode"`
	CountryCode   string `json:"countryCode"`
}

type moneyDTO struct {
	CurrencyCode string `json:"currencyCode"`
	Amount       string `json:"amount"`
}

type orderLineReqDTO struct {
	SKU         string   `json:"sku"`
	ProductName string   `json:"productName"`
	Quantity    int      `json:"quantity"`
	UnitPrice   moneyDTO `json:"unitPrice"`
}

type cancelOrderRequest struct {
	Reason string `json:"reason"`
}

type orderResponseDTO struct {
	ID                 string         `json:"id"`
	Channel            string         `json:"channel"`
	ExternalID         string         `json:"externalId,omitempty"`
	Customer           customerDTO    `json:"customer"`
	ShippingAddress    addressDTO     `json:"shippingAddress"`
	BillingAddress     addressDTO     `json:"billingAddress"`
	Lines              []orderLineDTO `json:"lines"`
	CurrencyCode       string         `json:"currencyCode"`
	OrderTotal         moneyDTO       `json:"orderTotal"`
	Status             string         `json:"status"`
	PlacedAt           time.Time      `json:"placedAt"`
	ConfirmedAt        *time.Time     `json:"confirmedAt,omitempty"`
	CancelledAt        *time.Time     `json:"cancelledAt,omitempty"`
	CancellationReason string         `json:"cancellationReason,omitempty"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

type orderLineDTO struct {
	ID          string   `json:"id"`
	SKU         string   `json:"sku"`
	ProductName string   `json:"productName"`
	Quantity    int      `json:"quantity"`
	UnitPrice   moneyDTO `json:"unitPrice"`
	LineTotal   moneyDTO `json:"lineTotal"`
}

type orderSummaryDTO struct {
	ID           string   `json:"id"`
	Channel      string   `json:"channel"`
	ExternalID   string   `json:"externalId,omitempty"`
	CustomerName string   `json:"customerName"`
	Status       string   `json:"status"`
	OrderTotal   moneyDTO `json:"orderTotal"`
	PlacedAt     time.Time `json:"placedAt"`
}

type paginationDTO struct {
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
}

type orderListDTO struct {
	Data       []orderSummaryDTO `json:"data"`
	Pagination paginationDTO     `json:"pagination"`
}

// --- Handlers ---

// HandleCreateOrder handles POST /v1/orders
func (h *HTTPHandler) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		h.writeProblem(w, pkgerrors.NewValidationError("Idempotency-Key header is required"))
		return
	}

	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeProblem(w, pkgerrors.NewValidationError("invalid request body: "+err.Error()))
		return
	}

	lines := make([]command.CreateOrderLine, len(req.Lines))
	for i, l := range req.Lines {
		up, err := money.NewMoney(l.UnitPrice.CurrencyCode, l.UnitPrice.Amount)
		if err != nil {
			h.writeProblem(w, pkgerrors.NewValidationError("invalid unit price: "+err.Error()))
			return
		}
		lines[i] = command.CreateOrderLine{
			SKU:         l.SKU,
			ProductName: l.ProductName,
			Quantity:    l.Quantity,
			UnitPrice:   up,
		}
	}

	cmd := command.CreateOrderCommand{
		IdempotencyKey:  idempotencyKey,
		Channel:         order.Channel(req.Channel),
		ExternalID:      req.ExternalID,
		Customer:        order.Customer{Name: req.Customer.Name, Email: req.Customer.Email, Phone: req.Customer.Phone},
		ShippingAddress: order.Address{Line1: req.ShippingAddress.Line1, Line2: req.ShippingAddress.Line2, City: req.ShippingAddress.City, StateOrRegion: req.ShippingAddress.StateOrRegion, PostalCode: req.ShippingAddress.PostalCode, CountryCode: req.ShippingAddress.CountryCode},
		BillingAddress:  order.Address{Line1: req.BillingAddress.Line1, Line2: req.BillingAddress.Line2, City: req.BillingAddress.City, StateOrRegion: req.BillingAddress.StateOrRegion, PostalCode: req.BillingAddress.PostalCode, CountryCode: req.BillingAddress.CountryCode},
		Lines:           lines,
		PlacedAt:        req.PlacedAt,
	}

	result, err := h.createOrder.Handle(r.Context(), cmd)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	status := http.StatusCreated
	if result.IsExisting {
		status = http.StatusOK
	}
	h.writeJSON(w, status, toOrderResponse(result.Order))
}

// HandleGetOrder handles GET /v1/orders/{orderId}
func (h *HTTPHandler) HandleGetOrder(w http.ResponseWriter, r *http.Request, orderID string) {
	o, err := h.getOrder.Handle(r.Context(), query.GetOrderQuery{OrderID: orderID})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, toOrderResponse(o))
}

// HandleListOrders handles GET /v1/orders
func (h *HTTPHandler) HandleListOrders(w http.ResponseWriter, r *http.Request) {
	q := query.ListOrdersQuery{
		Cursor: r.URL.Query().Get("cursor"),
	}

	if s := r.URL.Query().Get("status"); s != "" {
		status := order.OrderStatus(s)
		q.Status = &status
	}
	if c := r.URL.Query().Get("channel"); c != "" {
		ch := order.Channel(c)
		q.Channel = &ch
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		var limit int
		if _, err := json.Number(l).Int64(); err == nil {
			n, _ := json.Number(l).Int64()
			limit = int(n)
		}
		q.Limit = limit
	}

	result, err := h.listOrders.Handle(r.Context(), q)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	summaries := make([]orderSummaryDTO, len(result.Orders))
	for i, o := range result.Orders {
		summaries[i] = orderSummaryDTO{
			ID:           o.ID,
			Channel:      string(o.Channel),
			ExternalID:   o.ExternalID,
			CustomerName: o.Customer.Name,
			Status:       string(o.Status),
			OrderTotal:   moneyDTO{CurrencyCode: o.OrderTotal.CurrencyCode, Amount: o.OrderTotal.Amount},
			PlacedAt:     o.PlacedAt,
		}
	}

	h.writeJSON(w, http.StatusOK, orderListDTO{
		Data: summaries,
		Pagination: paginationDTO{
			NextCursor: result.NextCursor,
			HasMore:    result.HasMore,
		},
	})
}

// HandleListOrderLines handles GET /v1/orders/{orderId}/lines
func (h *HTTPHandler) HandleListOrderLines(w http.ResponseWriter, r *http.Request, orderID string) {
	o, err := h.getOrder.Handle(r.Context(), query.GetOrderQuery{OrderID: orderID})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	lines := make([]orderLineDTO, len(o.Lines))
	for i, l := range o.Lines {
		lines[i] = toOrderLineDTO(l)
	}
	h.writeJSON(w, http.StatusOK, lines)
}

// HandleConfirmOrder handles POST /v1/orders/{orderId}/confirm
func (h *HTTPHandler) HandleConfirmOrder(w http.ResponseWriter, r *http.Request, orderID string) {
	o, err := h.confirmOrder.Handle(r.Context(), command.ConfirmOrderCommand{OrderID: orderID})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, toOrderResponse(o))
}

// HandleCancelOrder handles POST /v1/orders/{orderId}/cancel
func (h *HTTPHandler) HandleCancelOrder(w http.ResponseWriter, r *http.Request, orderID string) {
	var req cancelOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeProblem(w, pkgerrors.NewValidationError("invalid request body"))
		return
	}

	o, err := h.cancelOrder.Handle(r.Context(), command.CancelOrderCommand{OrderID: orderID, Reason: req.Reason})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, toOrderResponse(o))
}

// --- Helpers ---

func (h *HTTPHandler) handleDomainError(w http.ResponseWriter, err error) {
	var transErr order.ErrInvalidTransition
	switch {
	case errors.Is(err, order.ErrOrderNotFound):
		h.writeProblem(w, pkgerrors.NewNotFoundError("Order", ""))
	case errors.As(err, &transErr):
		h.writeProblem(w, pkgerrors.NewConflictError(err.Error()))
	case errors.Is(err, order.ErrNoLineItems):
		h.writeProblem(w, pkgerrors.ProblemDetail{
			Type:   "https://problems.oms.io/validation-error",
			Title:  "Validation Error",
			Status: http.StatusUnprocessableEntity,
			Detail: err.Error(),
		})
	case errors.Is(err, order.ErrMixedCurrencies):
		h.writeProblem(w, pkgerrors.ProblemDetail{
			Type:   "https://problems.oms.io/validation-error",
			Title:  "Validation Error",
			Status: http.StatusUnprocessableEntity,
			Detail: err.Error(),
		})
	default:
		h.logger.Error("unhandled error", "error", err)
		h.writeProblem(w, pkgerrors.ProblemDetail{
			Type:   "https://problems.oms.io/internal-error",
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "an unexpected error occurred",
		})
	}
}

func (h *HTTPHandler) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

func (h *HTTPHandler) writeProblem(w http.ResponseWriter, problem pkgerrors.ProblemDetail) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)
	if err := json.NewEncoder(w).Encode(problem); err != nil {
		h.logger.Error("failed to encode problem response", "error", err)
	}
}

func toOrderResponse(o *order.Order) orderResponseDTO {
	lines := make([]orderLineDTO, len(o.Lines))
	for i, l := range o.Lines {
		lines[i] = toOrderLineDTO(l)
	}
	return orderResponseDTO{
		ID:                 o.ID,
		Channel:            string(o.Channel),
		ExternalID:         o.ExternalID,
		Customer:           customerDTO{Name: o.Customer.Name, Email: o.Customer.Email, Phone: o.Customer.Phone},
		ShippingAddress:    toAddressDTO(o.ShippingAddress),
		BillingAddress:     toAddressDTO(o.BillingAddress),
		Lines:              lines,
		CurrencyCode:       o.CurrencyCode,
		OrderTotal:         moneyDTO{CurrencyCode: o.OrderTotal.CurrencyCode, Amount: o.OrderTotal.Amount},
		Status:             string(o.Status),
		PlacedAt:           o.PlacedAt,
		ConfirmedAt:        o.ConfirmedAt,
		CancelledAt:        o.CancelledAt,
		CancellationReason: o.CancellationReason,
		CreatedAt:          o.CreatedAt,
		UpdatedAt:          o.UpdatedAt,
	}
}

func toOrderLineDTO(l order.OrderLine) orderLineDTO {
	return orderLineDTO{
		ID:          l.LineID,
		SKU:         l.SKU,
		ProductName: l.ProductName,
		Quantity:    l.Quantity,
		UnitPrice:   moneyDTO{CurrencyCode: l.UnitPrice.CurrencyCode, Amount: l.UnitPrice.Amount},
		LineTotal:   moneyDTO{CurrencyCode: l.LineTotal.CurrencyCode, Amount: l.LineTotal.Amount},
	}
}

func toAddressDTO(a order.Address) addressDTO {
	return addressDTO{
		Line1:         a.Line1,
		Line2:         a.Line2,
		City:          a.City,
		StateOrRegion: a.StateOrRegion,
		PostalCode:    a.PostalCode,
		CountryCode:   a.CountryCode,
	}
}
