package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type ConsumerGroup struct {
	reader *kafka.Reader
}

func NewConsumerGroup(brokers []string, groupID string, topics []string) (*ConsumerGroup, error) {
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

func (c *ConsumerGroup) Close() error {
	return c.reader.Close()
}
