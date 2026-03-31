package publisher

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oms/pkg/events"
	"github.com/oms/pkg/money"
	"github.com/oms/internal/orderintake/domain/order"
)

// mockKafkaWriter is a test double for kafka.Writer that records written messages.
type mockKafkaWriter struct {
	messages []kafka.Message
	writeErr error
}

func (m *mockKafkaWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.messages = append(m.messages, msgs...)
	return nil
}

func (m *mockKafkaWriter) Close() error {
	return nil
}

func TestKafkaEventPublisher_Publish_SerializesToJSON(t *testing.T) {
	mock := &mockKafkaWriter{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	pub := &EventPublisher{
		writer: mock,
		logger: logger,
	}

	evt := order.OrderConfirmed{
		BaseEvent: events.NewBaseEvent("order.confirmed", "order-123", "Order"),
		OrderID:   "order-123",
		Channel:   order.ChannelEcommerce,
		OrderTotal: money.Money{
			CurrencyCode: "USD",
			Amount:       "100.00",
		},
		ConfirmedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	err := pub.Publish(context.Background(), evt)
	require.NoError(t, err)
	require.Len(t, mock.messages, 1)

	msg := mock.messages[0]
	var payload map[string]interface{}
	err = json.Unmarshal(msg.Value, &payload)
	require.NoError(t, err)

	assert.Equal(t, "order.confirmed", payload["Type"])
	assert.Equal(t, "order-123", payload["AggregateId"])
}

func TestKafkaEventPublisher_Publish_SetsMessageKey(t *testing.T) {
	mock := &mockKafkaWriter{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	pub := &EventPublisher{
		writer: mock,
		logger: logger,
	}

	evt := order.OrderCancelled{
		BaseEvent:   events.NewBaseEvent("order.cancelled", "order-456", "Order"),
		OrderID:     "order-456",
		Reason:      "Customer request",
		CancelledAt: time.Now().UTC(),
	}

	err := pub.Publish(context.Background(), evt)
	require.NoError(t, err)
	require.Len(t, mock.messages, 1)

	msg := mock.messages[0]
	assert.Equal(t, "order-456", string(msg.Key))
}

func TestKafkaEventPublisher_Publish_DerivesTopicFromEventType(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		wantTopic string
	}{
		{
			name:      "order confirmed",
			eventType: "order.confirmed",
			wantTopic: "oms.orders.confirmed",
		},
		{
			name:      "order cancelled",
			eventType: "order.cancelled",
			wantTopic: "oms.orders.cancelled",
		},
		{
			name:      "order shipped",
			eventType: "order.shipped",
			wantTopic: "oms.orders.shipped",
		},
		{
			name:      "order delivered",
			eventType: "order.delivered",
			wantTopic: "oms.orders.delivered",
		},
		{
			name:      "order status changed",
			eventType: "order.status_changed",
			wantTopic: "oms.orders.status-changed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockKafkaWriter{}
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			pub := &EventPublisher{
				writer: mock,
				logger: logger,
			}

			evt := events.NewBaseEvent(tt.eventType, "order-123", "Order")

			err := pub.Publish(context.Background(), evt)
			require.NoError(t, err)
			require.Len(t, mock.messages, 1)

			msg := mock.messages[0]
			assert.Equal(t, tt.wantTopic, msg.Topic)
		})
	}
}

func TestKafkaEventPublisher_Publish_HandlesKafkaWriteError(t *testing.T) {
	mock := &mockKafkaWriter{
		writeErr: errors.New("kafka connection failed"),
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	pub := &EventPublisher{
		writer: mock,
		logger: logger,
	}

	evt := order.OrderConfirmed{
		BaseEvent: events.NewBaseEvent("order.confirmed", "order-123", "Order"),
		OrderID:   "order-123",
	}

	err := pub.Publish(context.Background(), evt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kafka connection failed")
}

func TestKafkaEventPublisher_Publish_MultipleEvents(t *testing.T) {
	mock := &mockKafkaWriter{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	pub := &EventPublisher{
		writer: mock,
		logger: logger,
	}

	evt1 := order.OrderConfirmed{
		BaseEvent: events.NewBaseEvent("order.confirmed", "order-123", "Order"),
		OrderID:   "order-123",
	}
	evt2 := order.OrderStatusChanged{
		BaseEvent:      events.NewBaseEvent("order.status_changed", "order-123", "Order"),
		OrderID:        "order-123",
		PreviousStatus: order.StatusPendingValidation,
		NewStatus:      order.StatusConfirmed,
		ChangedAt:      time.Now().UTC(),
	}

	err := pub.Publish(context.Background(), evt1, evt2)
	require.NoError(t, err)
	require.Len(t, mock.messages, 2)

	assert.Equal(t, "oms.orders.confirmed", mock.messages[0].Topic)
	assert.Equal(t, "oms.orders.status-changed", mock.messages[1].Topic)
}
