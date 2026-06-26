package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvent_Success(t *testing.T) {
	payload := map[string]interface{}{
		"description": "Test income",
		"amount":      int64(50000),
	}

	event, err := NewEvent("income", "inc-123", "income.created", payload)
	require.NoError(t, err)
	require.NotNil(t, event)

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, "income", event.AggregateType)
	assert.Equal(t, "inc-123", event.AggregateID)
	assert.Equal(t, "income.created", event.EventType)
	assert.NotNil(t, event.CreatedAt)
	assert.True(t, time.Since(*event.CreatedAt) < 5*time.Second)
	assert.Nil(t, event.PublishedAt)

	// Verify payload was serialized correctly
	var decoded map[string]interface{}
	err = json.Unmarshal(event.Payload, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "Test income", decoded["description"])
	assert.Equal(t, float64(50000), decoded["amount"])
}

func TestNewEvent_NilPayload(t *testing.T) {
	event, err := NewEvent("fixed_expense", "fe-456", "fixed_expense.created", nil)
	require.NoError(t, err)
	require.NotNil(t, event)

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, "fixed_expense", event.AggregateType)
	assert.Equal(t, "fe-456", event.AggregateID)
	assert.Equal(t, "fixed_expense.created", event.EventType)
	assert.Empty(t, event.Payload)
}

func TestNewEvent_EmptyPayload(t *testing.T) {
	event, err := NewEvent("income", "inc-789", "income.deleted", map[string]interface{}{})
	require.NoError(t, err)
	require.NotNil(t, event)

	assert.Equal(t, "income.deleted", event.EventType)
	assert.NotNil(t, event.Payload)
	assert.Equal(t, json.RawMessage("{}"), event.Payload)
}

func TestNewEvent_InvalidPayload(t *testing.T) {
	// Functions cannot be marshaled to JSON
	_, err := NewEvent("test", "id-1", "test.event", func() {})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "json")
}

func TestEvent_Defaults(t *testing.T) {
	event, err := NewEvent("income", "inc-1", "income.created", nil)
	require.NoError(t, err)

	// EventType should follow "domain.event" pattern (contains a dot)
	assert.Contains(t, event.EventType, ".")
	parts := strings.SplitN(event.EventType, ".", 2)
	assert.Len(t, parts, 2)
	assert.NotEmpty(t, parts[0])
	assert.NotEmpty(t, parts[1])

	// ID must not be empty
	assert.NotEmpty(t, event.ID)

	// CreatedAt must be set
	assert.NotNil(t, event.CreatedAt)
	assert.False(t, event.CreatedAt.IsZero())

	// PublishedAt must be nil for new events
	assert.Nil(t, event.PublishedAt)
}

// ---------------------------------------------------------------------------
// Outbox polling loop tests
// ---------------------------------------------------------------------------

// inMemoryStore implements the Repository interface purely in memory.
type inMemoryStore struct {
	mu     sync.Mutex
	events []Event
}

func (s *inMemoryStore) Save(_ context.Context, _ any, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, *event)
	return nil
}

func (s *inMemoryStore) Pending(_ context.Context) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var pending []Event
	for _, e := range s.events {
		if e.PublishedAt == nil {
			pending = append(pending, e)
		}
	}
	return pending, nil
}

func (s *inMemoryStore) MarkPublished(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.events {
		if s.events[i].ID == id {
			now := time.Now().UTC()
			s.events[i].PublishedAt = &now
			return nil
		}
	}
	return fmt.Errorf("event not found: %s", id)
}

// publishRecord represents a single call to the mock publisher.
type publishRecord struct {
	Topic string
	Key   string
	Value json.RawMessage
}

// mockPublisher simulates a Kafka producer; it can be configured to fail on a
// specific call index to test at-least-once retry semantics.
type mockPublisher struct {
	mu          sync.Mutex
	calls       []publishRecord
	failAtIndex int // -1 = never fail; 0-based index of the call to fail
}

func (m *mockPublisher) Publish(_ context.Context, topic string, key, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failAtIndex >= 0 && len(m.calls) == m.failAtIndex {
		return fmt.Errorf("simulated publish failure")
	}

	m.calls = append(m.calls, publishRecord{
		Topic: topic,
		Key:   string(key),
		Value: json.RawMessage(value),
	})
	return nil
}

// pollAndPublish simulates the outbox polling loop: fetch pending events from
// the store, publish each via the publisher, and mark as published on success.
// On publish failure the function stops immediately so that remaining events
// stay pending (at-least-once guarantee).
func pollAndPublish(ctx context.Context, store Repository, pub func(context.Context, string, []byte, []byte) error, topic string) error {
	events, err := store.Pending(ctx)
	if err != nil {
		return err
	}

	for _, event := range events {
		if event.EventType == "" {
			continue
		}

		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event %s: %w", event.ID, err)
		}

		if err := pub(ctx, topic, []byte(event.AggregateID), data); err != nil {
			return fmt.Errorf("publish event %s: %w", event.ID, err)
		}

		if err := store.MarkPublished(ctx, event.ID); err != nil {
			return fmt.Errorf("mark published %s: %w", event.ID, err)
		}
	}

	return nil
}

// verifyPublishRecord asserts that a publishRecord matches the expected topic
// and carries the target event's aggregate ID and serialised event ID.
func verifyPublishRecord(t *testing.T, rec publishRecord, topic string, event *Event) {
	t.Helper()
	assert.Equal(t, topic, rec.Topic)
	assert.Equal(t, event.AggregateID, rec.Key)

	var decoded Event
	err := json.Unmarshal(rec.Value, &decoded)
	require.NoError(t, err)
	assert.Equal(t, event.ID, decoded.ID)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestOutboxPublisher_PollAndPublish(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := &inMemoryStore{}
	pub := &mockPublisher{failAtIndex: -1}

	// Store 3 pending events.
	payload := map[string]string{"test": "data"}
	ev1, err := NewEvent("account", "acc-1", "account.created", payload)
	require.NoError(t, err)
	ev2, err := NewEvent("account", "acc-2", "account.created", payload)
	require.NoError(t, err)
	ev3, err := NewEvent("account", "acc-3", "account.created", payload)
	require.NoError(t, err)

	require.NoError(t, store.Save(ctx, nil, ev1))
	require.NoError(t, store.Save(ctx, nil, ev2))
	require.NoError(t, store.Save(ctx, nil, ev3))

	// Act — run one poll cycle.
	err = pollAndPublish(ctx, store, pub.Publish, "outbox.test_events")
	require.NoError(t, err)

	// Assert all 3 events were published.
	assert.Len(t, pub.calls, 3)
	verifyPublishRecord(t, pub.calls[0], "outbox.test_events", ev1)
	verifyPublishRecord(t, pub.calls[1], "outbox.test_events", ev2)
	verifyPublishRecord(t, pub.calls[2], "outbox.test_events", ev3)

	// Assert all events are marked as published.
	pending, err := store.Pending(ctx)
	require.NoError(t, err)
	assert.Empty(t, pending, "no events should remain pending after successful publish")

	for _, ev := range store.events {
		assert.NotNil(t, ev.PublishedAt, "event %s should be marked published", ev.ID)
	}
}

func TestOutboxPublisher_PollEmptyTable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := &inMemoryStore{}
	pub := &mockPublisher{failAtIndex: -1}

	// Act — no events in the store.
	err := pollAndPublish(ctx, store, pub.Publish, "outbox.test_events")
	require.NoError(t, err)

	// Assert nothing was published.
	assert.Len(t, pub.calls, 0)

	// Assert still no pending events.
	pending, err := store.Pending(ctx)
	require.NoError(t, err)
	assert.Empty(t, pending)
}

func TestOutboxPublisher_PollWithPublishFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := &inMemoryStore{}
	pub := &mockPublisher{failAtIndex: 1} // 2nd call fails

	// Store 3 pending events.
	payload := map[string]string{"test": "data"}
	ev1, err := NewEvent("account", "acc-1", "account.created", payload)
	require.NoError(t, err)
	ev2, err := NewEvent("account", "acc-2", "account.created", payload)
	require.NoError(t, err)
	ev3, err := NewEvent("account", "acc-3", "account.created", payload)
	require.NoError(t, err)

	require.NoError(t, store.Save(ctx, nil, ev1))
	require.NoError(t, store.Save(ctx, nil, ev2))
	require.NoError(t, store.Save(ctx, nil, ev3))

	// Act — run one poll cycle.
	err = pollAndPublish(ctx, store, pub.Publish, "outbox.test_events")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simulated publish failure")

	// Assert only the 1st event was published.
	assert.Len(t, pub.calls, 1)
	verifyPublishRecord(t, pub.calls[0], "outbox.test_events", ev1)

	// Assert 1st event is marked as published.
	assert.NotNil(t, store.events[0].PublishedAt,
		"event %s should be marked published", store.events[0].ID)

	// Assert 2nd and 3rd events are still pending (not marked).
	assert.Nil(t, store.events[1].PublishedAt,
		"event %s should NOT be marked published", store.events[1].ID)
	assert.Nil(t, store.events[2].PublishedAt,
		"event %s should NOT be marked published", store.events[2].ID)

	// Confirm the store reports 2 pending events.
	pending, err := store.Pending(ctx)
	require.NoError(t, err)
	assert.Len(t, pending, 2)
	for _, ev := range pending {
		assert.Nil(t, ev.PublishedAt,
			"pending event %s should not be marked published", ev.ID)
	}
}
