//nolint:goconst
package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProducer_EmptyBrokers(t *testing.T) {
	producer, err := NewProducer([]string{})
	require.NoError(t, err)
	require.NotNil(t, producer)
	defer producer.Close()

	// Publish with cancelled context to avoid actually connecting
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = producer.Publish(ctx, "test-topic", []byte("key"), []byte("value"))
	require.Error(t, err) // context cancelled or no brokers
}

func TestNewProducer_NilBrokers(t *testing.T) {
	producer, err := NewProducer(nil)
	require.NoError(t, err)
	require.NotNil(t, producer)
	producer.Close()
}

func TestHashBalancer_Consistency(t *testing.T) {
	balancer := &kafka.Hash{}

	partitions := []int{0, 1, 2, 3, 4}

	key1 := []byte("user-42")
	msg1 := kafka.Message{Key: key1}
	partition1a := balancer.Balance(msg1, partitions...)
	partition1b := balancer.Balance(msg1, partitions...)
	assert.Equal(t, partition1a, partition1b, "same key must hash to same partition")

	key2 := []byte("user-99")
	msg2 := kafka.Message{Key: key2}
	partition2a := balancer.Balance(msg2, partitions...)
	partition2b := balancer.Balance(msg2, partitions...)
	assert.Equal(t, partition2a, partition2b, "same key must hash to same partition")
}

func TestHashBalancer_DifferentKeys(t *testing.T) {
	balancer := &kafka.Hash{}

	partitions := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	msg1 := kafka.Message{Key: []byte("key-alpha")}
	msg2 := kafka.Message{Key: []byte("key-beta")}

	p1 := balancer.Balance(msg1, partitions...)
	p2 := balancer.Balance(msg2, partitions...)

	// Different keys may or may not collide; just verify both are valid
	assert.GreaterOrEqual(t, p1, 0)
	assert.Less(t, p1, len(partitions))
	assert.GreaterOrEqual(t, p2, 0)
	assert.Less(t, p2, len(partitions))
}

func TestHashBalancer_EmptyKey(t *testing.T) {
	balancer := &kafka.Hash{}
	partitions := []int{0, 1, 2, 3, 4}

	msg := kafka.Message{Key: []byte{}}
	partition := balancer.Balance(msg, partitions...)
	assert.GreaterOrEqual(t, partition, 0)
	assert.Less(t, partition, len(partitions))
}

func TestHashBalancer_SinglePartition(t *testing.T) {
	balancer := &kafka.Hash{}
	partitions := []int{0}

	msg := kafka.Message{Key: []byte("any-key")}
	partition := balancer.Balance(msg, partitions...)
	assert.Equal(t, 0, partition)
}

func TestNewConsumerGroup_Valid(t *testing.T) {
	cg, err := NewConsumerGroup([]string{"localhost:9092"}, "test-group", []string{"test-topic"})
	require.NoError(t, err)
	require.NotNil(t, cg)
	defer func() { _ = cg.Close() }()
}

func TestNewConsumerGroup_EmptyTopics(t *testing.T) {
	_, err := NewConsumerGroup([]string{"localhost:9092"}, "test-group", []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one topic")
}

func TestProducer_PublishAsyncNonBlocking(t *testing.T) {
	producer, err := NewProducer([]string{})
	require.NoError(t, err)
	defer producer.Close()

	// PublishAsync should not block even without brokers (buffered channel)
	ctx := context.Background()
	producer.PublishAsync(ctx, "test-topic", []byte("key"), []byte("value"))
	// No assertion on delivery — async path is fire-and-forget
}

func TestConsumer_ConsumeCancelledContext(t *testing.T) {
	cg, err := NewConsumerGroup([]string{"localhost:9092"}, "test-group", []string{"test-topic"})
	require.NoError(t, err)
	defer func() { _ = cg.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	handler := func(msg kafka.Message) error {
		return nil
	}

	err = cg.Consume(ctx, handler)
	require.Error(t, err)
}

func TestConsumer_Close(t *testing.T) {
	cg, err := NewConsumerGroup([]string{"localhost:9092"}, "test-group", []string{"test-topic"})
	require.NoError(t, err)

	err = cg.Close()
	require.NoError(t, err)

	// Closing again should not panic
	err = cg.Close()
	require.NoError(t, err)
}

func TestProducer_Close(t *testing.T) {
	producer, err := NewProducer([]string{"localhost:9092"})
	require.NoError(t, err)

	// Close should not panic
	producer.Close()

	// Second close should not panic either
	producer.Close()
}

func TestProducer_CloseCancelsLoop(t *testing.T) {
	producer, err := NewProducer([]string{"localhost:9092"})
	require.NoError(t, err)

	// Send an async message before closing
	producer.PublishAsync(context.Background(), "t", []byte("k"), []byte("v"))

	// Close should drain gracefully
	done := make(chan struct{})
	go func() {
		producer.Close()
		close(done)
	}()

	select {
	case <-done:
		// OK — close completed
	case <-time.After(5 * time.Second):
		t.Fatal("Close timed out — loop may not be exiting")
	}
}
