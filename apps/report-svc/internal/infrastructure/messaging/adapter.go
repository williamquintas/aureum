package messaging

import (
	"context"

	"github.com/aureum/pkg/kafka"
	kafkago "github.com/segmentio/kafka-go"
)

// ConsumerAdapter bridges pkg/kafka.ConsumerGroup (which delivers kafka.Message)
// to EventHandler (which expects raw []byte payloads).
//
// This adapter exists because the application-layer KafkaConsumer interface
// consumes func(msg []byte) error while the concrete ConsumerGroup delivers
// kafka.Message structs from the segmentio/kafka-go library.
type ConsumerAdapter struct {
	consumer *kafka.ConsumerGroup
	handler  *EventHandler
}

// NewConsumerAdapter wraps a ConsumerGroup and EventHandler into a single
// unit that can be started and stopped together.
func NewConsumerAdapter(consumer *kafka.ConsumerGroup, handler *EventHandler) *ConsumerAdapter {
	return &ConsumerAdapter{consumer: consumer, handler: handler}
}

// Start begins consuming messages. It blocks until the context is cancelled
// or a non-recoverable error occurs. Callers should run this in a goroutine.
func (a *ConsumerAdapter) Start(ctx context.Context) error {
	return a.consumer.Consume(ctx, a.handleMessage)
}

// handleMessage extracts the raw Value bytes from a kafka.Message and
// delegates to the EventHandler for routing and projection.
func (a *ConsumerAdapter) handleMessage(msg kafkago.Message) error {
	return a.handler.HandleMessage(context.Background(), msg.Value)
}

// Close shuts down the underlying ConsumerGroup reader.
func (a *ConsumerAdapter) Close() error {
	return a.consumer.Close()
}
