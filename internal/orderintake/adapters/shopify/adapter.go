// Package shopify implements an Anti-Corruption Layer that translates Shopify
// webhook payloads into the Order Intake bounded context's command model. It
// isolates the domain from Shopify's data format and verifies webhook
// authenticity via HMAC-SHA256.
package shopify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oms/internal/orderintake/app/command"
	"github.com/oms/internal/orderintake/domain/order"
	"github.com/oms/pkg/money"
)

// Adapter translates Shopify webhook payloads into domain commands and verifies
// webhook HMAC signatures.
type Adapter struct {
	webhookSecret string
}

// NewAdapter creates a new Shopify Anti-Corruption Layer adapter.
// webhookSecret is the shared secret from the Shopify webhook configuration.
func NewAdapter(webhookSecret string) *Adapter {
	return &Adapter{webhookSecret: webhookSecret}
}

// VerifyHMAC verifies the X-Shopify-Hmac-Sha256 header value against the raw
// request body using the configured webhook secret. Returns true if the
// signature is valid.
func (a *Adapter) VerifyHMAC(body []byte, hmacHeader string) bool {
	mac := hmac.New(sha256.New, []byte(a.webhookSecret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	expectedB64 := base64.StdEncoding.EncodeToString(expectedMAC)

	return hmac.Equal([]byte(expectedB64), []byte(hmacHeader))
}

// TranslateOrder converts a Shopify webhook JSON body into a CreateOrderCommand
// suitable for the Order Intake bounded context. It maps Shopify's nested JSON
// structure to the domain's flat value objects and generates an idempotency key
// derived from the Shopify order ID.
func (a *Adapter) TranslateOrder(body []byte) (command.CreateOrderCommand, error) {
	var so shopifyOrder
	if err := json.Unmarshal(body, &so); err != nil {
		return command.CreateOrderCommand{}, fmt.Errorf("shopify: unmarshal order: %w", err)
	}

	if so.ID == 0 {
		return command.CreateOrderCommand{}, fmt.Errorf("shopify: order id is required")
	}

	// Parse placed_at from Shopify's created_at timestamp.
	placedAt, err := time.Parse(time.RFC3339, so.CreatedAt)
	if err != nil {
		return command.CreateOrderCommand{}, fmt.Errorf("shopify: parse created_at %q: %w", so.CreatedAt, err)
	}

	// Resolve the currency code. Shopify provides it at the order level.
	currencyCode := so.Currency
	if currencyCode == "" {
		currencyCode = "USD"
	}

	// Translate line items.
	lines := make([]command.CreateOrderLine, 0, len(so.LineItems))
	for _, li := range so.LineItems {
		unitPrice, err := money.NewMoney(currencyCode, li.Price)
		if err != nil {
			return command.CreateOrderCommand{}, fmt.Errorf("shopify: parse line item price for sku %q: %w", li.SKU, err)
		}
		lines = append(lines, command.CreateOrderLine{
			SKU:         li.SKU,
			ProductName: li.Title,
			Quantity:    li.Quantity,
			UnitPrice:   unitPrice,
		})
	}

	// Translate shipping address.
	shippingAddress := order.Address{}
	if so.ShippingAddress != nil {
		shippingAddress = order.Address{
			Line1:         so.ShippingAddress.Address1,
			Line2:         so.ShippingAddress.Address2,
			City:          so.ShippingAddress.City,
			StateOrRegion: so.ShippingAddress.Province,
			PostalCode:    so.ShippingAddress.Zip,
			CountryCode:   so.ShippingAddress.CountryCode,
		}
	}

	// Translate billing address.
	billingAddress := order.Address{}
	if so.BillingAddress != nil {
		billingAddress = order.Address{
			Line1:         so.BillingAddress.Address1,
			Line2:         so.BillingAddress.Address2,
			City:          so.BillingAddress.City,
			StateOrRegion: so.BillingAddress.Province,
			PostalCode:    so.BillingAddress.Zip,
			CountryCode:   so.BillingAddress.CountryCode,
		}
	}

	// Build the customer from Shopify's top-level fields.
	customer := order.Customer{
		Name:  fullName(so.Customer.FirstName, so.Customer.LastName),
		Email: so.Customer.Email,
		Phone: so.Customer.Phone,
	}

	// Generate idempotency key from Shopify order ID to ensure exactly-once
	// processing of webhook deliveries.
	idempotencyKey := fmt.Sprintf("shopify-%d", so.ID)

	return command.CreateOrderCommand{
		IdempotencyKey:  idempotencyKey,
		Channel:         order.ChannelEcommerce,
		ExternalID:      fmt.Sprintf("%d", so.ID),
		Customer:        customer,
		ShippingAddress: shippingAddress,
		BillingAddress:  billingAddress,
		Lines:           lines,
		PlacedAt:        placedAt,
	}, nil
}

// --------------------------------------------------------------------------
// Shopify webhook JSON model (private)
// --------------------------------------------------------------------------

// shopifyOrder represents the relevant fields from a Shopify orders/create
// webhook payload. Only the fields needed for translation are mapped.
type shopifyOrder struct {
	ID               int64             `json:"id"`
	Name             string            `json:"name"`
	CreatedAt        string            `json:"created_at"`
	Currency         string            `json:"currency"`
	TotalPrice       string            `json:"total_price"`
	Customer         shopifyCustomer   `json:"customer"`
	LineItems        []shopifyLineItem `json:"line_items"`
	ShippingAddress  *shopifyAddress   `json:"shipping_address"`
	BillingAddress   *shopifyAddress   `json:"billing_address"`
}

// shopifyCustomer represents the customer block inside a Shopify order.
type shopifyCustomer struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

// shopifyLineItem represents a single line item in a Shopify order.
type shopifyLineItem struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	Price     string `json:"price"`
}

// shopifyAddress represents a shipping or billing address in a Shopify order.
type shopifyAddress struct {
	Address1    string `json:"address1"`
	Address2    string `json:"address2"`
	City        string `json:"city"`
	Province    string `json:"province"`
	Zip         string `json:"zip"`
	CountryCode string `json:"country_code"`
}

// fullName combines first and last name with a space separator.
func fullName(first, last string) string {
	if first == "" {
		return last
	}
	if last == "" {
		return first
	}
	return first + " " + last
}
