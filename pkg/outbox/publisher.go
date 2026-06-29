package outbox

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aureum/pkg/kafka"
)

// Publisher periodically polls the outbox store and publishes pending events to Kafka.
type Publisher struct {
	store    *Store
	producer *kafka.Producer
	topic    string
	ticker   *time.Ticker
	stopCh   chan struct{}
}

// NewPublisher creates a Publisher that polls the outbox and publishes events to Kafka.
func NewPublisher(store *Store, producer *kafka.Producer, topic string, interval time.Duration) *Publisher {
	return &Publisher{
		store:    store,
		producer: producer,
		topic:    topic,
		ticker:   time.NewTicker(interval),
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic polling loop for publishing pending events.
func (p *Publisher) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-p.ticker.C:
				p.publishPending(ctx)
			case <-p.stopCh:
				return
			}
		}
	}()
}

// Stop stops the periodic polling and releases internal resources.
func (p *Publisher) Stop() {
	p.ticker.Stop()
	close(p.stopCh)
}

func (p *Publisher) publishPending(ctx context.Context) {
	events, err := p.store.Pending(ctx)
	if err != nil {
		log.Printf("outbox: failed to fetch pending events: %v", err)
		return
	}

	for _, event := range events {
		if event.EventType == "" {
			continue
		}

		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("outbox: failed to marshal event %s: %v", event.ID, err)
			continue
		}

		if err := p.producer.Publish(ctx, p.topic, []byte(event.AggregateID), data); err != nil {
			log.Printf("outbox: failed to publish event %s: %v", event.ID, err)
			continue
		}

		if err := p.store.MarkPublished(ctx, event.ID); err != nil {
			log.Printf("outbox: failed to mark event %s as published: %v", event.ID, err)
		}
	}
}
