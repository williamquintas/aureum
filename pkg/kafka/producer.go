package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer  *kafka.Writer
	asyncCh chan asyncMessage
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

type asyncMessage struct {
	topic string
	key   []byte
	value []byte
}

func NewProducer(brokers []string) (*Producer, error) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.Hash{},
		Async:    false,
	}

	ctx, cancel := context.WithCancel(context.Background())
	p := &Producer{
		writer:  writer,
		asyncCh: make(chan asyncMessage, 100),
		ctx:     ctx,
		cancel:  cancel,
	}

	p.wg.Add(1)
	go p.loop()

	return p, nil
}

func (p *Producer) Publish(ctx context.Context, topic string, key, value []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	return nil
}

func (p *Producer) PublishAsync(ctx context.Context, topic string, key, value []byte) {
	select {
	case p.asyncCh <- asyncMessage{topic: topic, key: key, value: value}:
	case <-ctx.Done():
	case <-p.ctx.Done():
	}
}

func (p *Producer) Close() {
	p.cancel()
	p.wg.Wait()
	if err := p.writer.Close(); err != nil {
		_ = err // log or metric in production
	}
}

func (p *Producer) loop() {
	defer p.wg.Done()
	for {
		select {
		case msg := <-p.asyncCh:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := p.writer.WriteMessages(ctx, kafka.Message{
				Topic: msg.topic,
				Key:   msg.key,
				Value: msg.value,
			})
			cancel()
			if err != nil {
				_ = err // log or metric in production
			}
		case <-p.ctx.Done():
			return
		}
	}
}
