package service

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oms/internal/orderintake/adapters/postgres"
	"github.com/oms/internal/orderintake/adapters/publisher"
	"github.com/oms/internal/orderintake/adapters/shopify"
	"github.com/oms/internal/orderintake/app/command"
	"github.com/oms/internal/orderintake/app/query"
	"github.com/oms/internal/orderintake/ports"
)

// Service holds all wired dependencies for the Order Intake bounded context.
type Service struct {
	Handler        *ports.HTTPHandler
	ShopifyAdapter *shopify.Adapter
	Router         func() interface{} // Returns chi.Router via ports.NewRouter
	publisher      *publisher.EventPublisher
}

// Config holds configuration for the service.
type Config struct {
	ShopifyWebhookSecret string
	KafkaBrokers         string
}

// New creates a fully wired Service with all dependencies.
func New(pool *pgxpool.Pool, cfg Config, logger *slog.Logger) *Service {
	// Adapters
	repo := postgres.NewOrderRepository(pool)
	pub := publisher.NewKafkaEventPublisher(cfg.KafkaBrokers, logger)
	shopifyACL := shopify.NewAdapter(cfg.ShopifyWebhookSecret)

	// Command handlers
	createOrderHandler := command.NewCreateOrderHandler(repo, pub)
	confirmOrderHandler := command.NewConfirmOrderHandler(repo, pub)
	cancelOrderHandler := command.NewCancelOrderHandler(repo, pub)
	markShippedHandler := command.NewMarkShippedHandler(repo, pub)
	markPartialShippedHandler := command.NewMarkPartiallyShippedHandler(repo, pub)
	markDeliveredHandler := command.NewMarkDeliveredHandler(repo, pub)
	markUnfulfillableHandler := command.NewMarkUnfulfillableHandler(repo, pub)
	markCompletedHandler := command.NewMarkCompletedHandler(repo, pub)

	// Query handlers
	getOrderHandler := query.NewGetOrderHandler(repo)
	listOrdersHandler := query.NewListOrdersHandler(repo)

	// HTTP handler
	httpHandler := ports.NewHTTPHandler(
		createOrderHandler,
		confirmOrderHandler,
		cancelOrderHandler,
		markShippedHandler,
		markPartialShippedHandler,
		markDeliveredHandler,
		markUnfulfillableHandler,
		markCompletedHandler,
		getOrderHandler,
		listOrdersHandler,
		logger,
	)

	return &Service{
		Handler:        httpHandler,
		ShopifyAdapter: shopifyACL,
		publisher:      pub,
	}
}

// Close gracefully shuts down service resources.
func (s *Service) Close() error {
	if s.publisher != nil {
		return s.publisher.Close()
	}
	return nil
}
