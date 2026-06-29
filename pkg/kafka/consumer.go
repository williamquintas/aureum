// Package kafka provides consumer and producer wrappers for Kafka messaging.
package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// ConsumerGroup wraps a kafka.Reader for consuming messages from a consumer group.
type ConsumerGroup struct {
	reader *kafka.Reader
}

// NewConsumerGroup creates a new Kafka consumer group reader.
func NewConsumerGroup(brokers []string, groupID string, topics []string) (*ConsumerGroup, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("at least one topic is required")
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupID,
		Topic:          topics[0],
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        3 * time.Second,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	return &ConsumerGroup{reader: reader}, nil
}

// Consume starts a blocking loop that reads messages and passes them to the handler.
func (c *ConsumerGroup) Consume(ctx context.Context, handler func(msg kafka.Message) error) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return fmt.Errorf("read message: %w", err)
		}

		if err := handler(msg); err != nil {
			return fmt.Errorf("handle message: %w", err)
		}
	}
}

// Close shuts down the consumer group reader.
func (c *ConsumerGroup) Close() error {
	return c.reader.Close()
}
