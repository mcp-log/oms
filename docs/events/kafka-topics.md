---
layout: default
title: Kafka Topics
parent: Events
nav_order: 2
---

# Kafka Topics Configuration
{: .no_toc }

Complete guide to Kafka topic configuration, producer/consumer patterns, and monitoring for Order Intake events.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Topic Naming Convention

All Order Intake topics follow the pattern: `oms.orders.<event-type>`

| Event Type | Kafka Topic | Purpose |
|------------|-------------|---------|
| `order.confirmed` | `oms.orders.confirmed` | Order confirmation events |
| `order.cancelled` | `oms.orders.cancelled` | Order cancellation events |
| `order.shipped` | `oms.orders.shipped` | Order shipment events |
| `order.delivered` | `oms.orders.delivered` | Order delivery events |
| `order.status_changed` | `oms.orders.status-changed` | All status change events |

**Note**: Event types use underscores (`order.status_changed`), but topic names use hyphens (`oms.orders.status-changed`) for Kafka naming conventions.

---

## Topic Configuration

### Production Settings

Recommended configuration for production deployments:

```properties
# Replication & Durability
replication.factor=3
min.insync.replicas=2
unclean.leader.election.enable=false

# Partitioning
num.partitions=6
partition.assignment.strategy=org.apache.kafka.clients.consumer.RangeAssignor

# Retention
retention.ms=604800000  # 7 days
retention.bytes=-1      # No size limit

# Compression
compression.type=snappy
```

### Local Development Settings

For local development with `docker-compose.yml`:

```properties
# Minimal replication for single-broker setup
replication.factor=1
min.insync.replicas=1

# Single partition for simplicity
num.partitions=1

# Short retention for testing
retention.ms=86400000  # 1 day

# No compression
compression.type=none
```

---

## Creating Topics

### Using kafka-topics.sh

Create all Order Intake topics at once:

```bash
# Production
for topic in confirmed cancelled shipped delivered status-changed; do
  kafka-topics.sh --create \
    --bootstrap-server localhost:9092 \
    --topic "oms.orders.$topic" \
    --partitions 6 \
    --replication-factor 3 \
    --config min.insync.replicas=2 \
    --config retention.ms=604800000
done

# Development
for topic in confirmed cancelled shipped delivered status-changed; do
  kafka-topics.sh --create \
    --bootstrap-server localhost:9092 \
    --topic "oms.orders.$topic" \
    --partitions 1 \
    --replication-factor 1
done
```

### Using Terraform (Infrastructure as Code)

```hcl
resource "kafka_topic" "order_confirmed" {
  name               = "oms.orders.confirmed"
  replication_factor = 3
  partitions         = 6

  config = {
    "retention.ms"           = "604800000"
    "min.insync.replicas"    = "2"
    "compression.type"       = "snappy"
  }
}

resource "kafka_topic" "order_cancelled" {
  name               = "oms.orders.cancelled"
  replication_factor = 3
  partitions         = 6

  config = {
    "retention.ms"           = "604800000"
    "min.insync.replicas"    = "2"
    "compression.type"       = "snappy"
  }
}

# ... repeat for shipped, delivered, status-changed
```

---

## Producer Configuration

### Order Intake Service (Go)

The Order Intake service produces events using `segmentio/kafka-go`:

```go
package kafka

import (
    "context"
    "encoding/json"

    "github.com/segmentio/kafka-go"
)

type Publisher struct {
    writer *kafka.Writer
}

func NewPublisher(brokers []string) *Publisher {
    return &Publisher{
        writer: &kafka.Writer{
            Addr:         kafka.TCP(brokers...),
            Balancer:     &kafka.LeastBytes{},
            RequiredAcks: kafka.RequireAll,  // Wait for all ISRs
            Compression:  kafka.Snappy,
            MaxAttempts:  3,
        },
    }
}

func (p *Publisher) PublishOrderConfirmed(ctx context.Context, event OrderConfirmedEvent) error {
    payload, err := json.Marshal(event)
    if err != nil {
        return err
    }

    return p.writer.WriteMessages(ctx, kafka.Message{
        Topic: "oms.orders.confirmed",
        Key:   []byte(event.OrderID),  // Order UUID for partition ordering
        Value: payload,
    })
}
```

**Key Configuration**:
- `RequiredAcks: kafka.RequireAll` - Wait for all in-sync replicas (durability)
- `Message.Key: orderID` - Ensures all events for the same order go to the same partition (ordering guarantee)
- `Compression: kafka.Snappy` - Fast compression with reasonable size reduction

---

## Consumer Configuration

### Consumer Groups

Each downstream service should use a unique consumer group ID:

| Service | Consumer Group ID | Subscribed Topics |
|---------|-------------------|-------------------|
| Fulfillment | `fulfillment-service` | `oms.orders.confirmed`, `oms.orders.cancelled` |
| Billing | `billing-service` | `oms.orders.confirmed`, `oms.orders.shipped`, `oms.orders.cancelled` |
| Notification | `notification-service` | `oms.orders.confirmed`, `oms.orders.shipped`, `oms.orders.delivered` |
| Analytics | `analytics-service` | `oms.orders.status-changed` |

### Go Consumer Example

```go
package main

import (
    "context"
    "encoding/json"
    "log"

    "github.com/segmentio/kafka-go"
)

func main() {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:         []string{"localhost:9092"},
        Topic:           "oms.orders.confirmed",
        GroupID:         "fulfillment-service",
        MinBytes:        10e3,  // 10KB
        MaxBytes:        10e6,  // 10MB
        CommitInterval:  1000,  // Commit offsets every 1s
        StartOffset:     kafka.LastOffset,
    })
    defer reader.Close()

    for {
        msg, err := reader.ReadMessage(context.Background())
        if err != nil {
            log.Fatal(err)
        }

        var event OrderConfirmedEvent
        if err := json.Unmarshal(msg.Value, &event); err != nil {
            log.Printf("Failed to unmarshal: %v", err)
            continue
        }

        // Process event...
        log.Printf("Processing order: %s", event.OrderID)

        // Commit offset automatically handled by CommitInterval
    }
}
```

### Python Consumer Example

```python
from kafka import KafkaConsumer
import json

consumer = KafkaConsumer(
    'oms.orders.confirmed',
    bootstrap_servers=['localhost:9092'],
    group_id='billing-service',
    value_deserializer=lambda m: json.loads(m.decode('utf-8')),
    auto_offset_reset='latest',
    enable_auto_commit=True,
    auto_commit_interval_ms=1000
)

for message in consumer:
    event = message.value
    print(f"Processing order: {event['orderId']}")

    # Generate invoice...
    # Authorize payment...
```

---

## Partition Strategy

### Message Key = Order UUID

All events for the same order use the **Order UUID** as the message key. This ensures:

1. **Ordering Guarantee**: All events for `order-123` go to the same partition, maintaining event order
2. **Consumer Affinity**: The same consumer instance processes all events for a given order (stateful processing)
3. **Load Distribution**: Orders are distributed evenly across partitions

```go
kafka.Message{
    Topic: "oms.orders.confirmed",
    Key:   []byte(event.OrderID),  // UUID v7
    Value: payload,
}
```

### Partition Count Recommendations

| Environment | Partitions | Rationale |
|-------------|-----------|-----------|
| Development | 1 | Simplicity |
| Staging | 3 | Test parallelism |
| Production | 6-12 | Scale with throughput (allow 2-3 consumers per partition) |

---

## Monitoring & Operations

### Key Metrics to Monitor

#### Producer Metrics
- **Message Send Rate**: Events published per second
- **Send Latency**: Time to publish an event (p50, p95, p99)
- **Error Rate**: Failed publishes
- **Batch Size**: Messages per batch

#### Consumer Metrics
- **Message Consume Rate**: Events consumed per second
- **Consumer Lag**: Offset difference between producer and consumer
- **Processing Time**: Time to process each event
- **Error Rate**: Failed processing attempts

### Kafka CLI Monitoring

**Check consumer lag**:
```bash
kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --group fulfillment-service \
  --describe
```

**Output**:
```
TOPIC                    PARTITION  CURRENT-OFFSET  LOG-END-OFFSET  LAG
oms.orders.confirmed     0          1245            1245            0
oms.orders.confirmed     1          1198            1198            0
oms.orders.confirmed     2          1203            1210            7   ← Consumer lag!
```

**List all topics**:
```bash
kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --list | grep oms.orders
```

**Describe topic configuration**:
```bash
kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --topic oms.orders.confirmed \
  --describe
```

---

## Error Handling

### Retry Pattern

For transient errors (network issues, temporary downstream unavailability):

```go
func (c *Consumer) ProcessWithRetry(ctx context.Context, msg kafka.Message) error {
    maxRetries := 3
    backoff := time.Second

    for attempt := 0; attempt < maxRetries; attempt++ {
        err := c.Process(ctx, msg)
        if err == nil {
            return nil
        }

        log.Warnf("Attempt %d failed: %v", attempt+1, err)
        time.Sleep(backoff * time.Duration(attempt+1))
    }

    return fmt.Errorf("failed after %d retries", maxRetries)
}
```

### Dead Letter Queue (DLQ)

For poison messages or permanent failures:

```go
func (c *Consumer) ProcessOrDLQ(ctx context.Context, msg kafka.Message) {
    err := c.ProcessWithRetry(ctx, msg)
    if err != nil {
        // Send to DLQ
        dlqMsg := kafka.Message{
            Topic: "oms.orders.dlq",
            Key:   msg.Key,
            Value: msg.Value,
            Headers: []kafka.Header{
                {Key: "error", Value: []byte(err.Error())},
                {Key: "original-topic", Value: []byte(msg.Topic)},
            },
        }
        c.dlqWriter.WriteMessages(ctx, dlqMsg)
    }
}
```

---

## Troubleshooting

### Consumer Not Receiving Messages

**Check 1**: Verify topic exists
```bash
kafka-topics.sh --bootstrap-server localhost:9092 --list | grep oms.orders
```

**Check 2**: Verify consumer group is active
```bash
kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --group your-group-id \
  --describe
```

**Check 3**: Check consumer offset position
```bash
kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --group your-group-id \
  --reset-offsets --to-earliest \
  --topic oms.orders.confirmed \
  --execute
```

### High Consumer Lag

**Cause**: Consumer processing is slower than producer publishing

**Solutions**:
1. Scale horizontally: Add more consumer instances (up to partition count)
2. Optimize processing: Profile slow operations
3. Increase batch size: Process events in batches
4. Add partitions: Increase parallelism (requires data migration)

### Message Loss

**Symptoms**: Events not appearing in consumer

**Check 1**: Producer acknowledgment setting
```go
RequiredAcks: kafka.RequireAll  // Must wait for all ISRs
```

**Check 2**: Minimum in-sync replicas
```properties
min.insync.replicas=2  # Require 2 replicas before ack
```

**Check 3**: Check Kafka broker logs for errors

---

## Local Development

### Start Kafka with Docker Compose

Order Intake includes Kafka in `docker-compose.yml`:

```bash
make docker-up
```

This starts Kafka on `localhost:9092`.

### Verify Kafka is Running

```bash
docker ps | grep oms-kafka
```

### View Published Events

Use `kcat` (formerly `kafkacat`) to consume events:

```bash
kcat -b localhost:9092 \
     -t oms.orders.confirmed \
     -C -f 'Key: %k\nValue: %s\n---\n'
```

---

## Next Steps

- [Event Catalog](catalog) - View event schemas and payloads
- [API Reference](../api/v1/reference.html) - Endpoints that trigger events
- [Architecture](../architecture/) - Event-driven architecture overview
