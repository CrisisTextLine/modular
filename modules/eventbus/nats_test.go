package eventbus

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startTestNATSServer starts an embedded NATS server on a random port and
// returns the client URL. The server is automatically shut down when the
// test completes.
func startTestNATSServer(t *testing.T) string {
	t.Helper()

	srv, err := server.NewServer(&server.Options{
		Host:   "127.0.0.1",
		Port:   -1, // random free port
		NoLog:  true,
		NoSigs: true,
	})
	require.NoError(t, err, "failed to create embedded NATS server")

	srv.Start()
	t.Cleanup(srv.Shutdown)

	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatal("embedded NATS server failed to become ready")
	}

	return srv.ClientURL()
}

// TestNatsEventBusCreation tests creating a NATS event bus
func TestNatsEventBusCreation(t *testing.T) {
	url := startTestNATSServer(t)

	t.Run("creates with default configuration", func(t *testing.T) {
		config := map[string]interface{}{
			"url": url,
		}

		bus, err := NewNatsEventBus(config)
		require.NoError(t, err)
		require.NotNil(t, bus)
		defer bus.Stop(context.Background())
	})

	t.Run("creates with custom configuration", func(t *testing.T) {
		config := map[string]interface{}{
			"url":              url,
			"connectionName":   "test-connection",
			"maxReconnects":    5,
			"reconnectWait":    1,
			"allowReconnect":   true,
			"pingInterval":     10,
			"maxPingsOut":      3,
			"subscribeTimeout": 10,
		}

		bus, err := NewNatsEventBus(config)
		require.NoError(t, err)
		require.NotNil(t, bus)
		defer bus.Stop(context.Background())
	})
}

// TestNatsEventBusInterface verifies that NatsEventBus implements EventBus
func TestNatsEventBusInterface(t *testing.T) {
	url := startTestNATSServer(t)

	config := map[string]interface{}{
		"url": url,
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err)
	defer bus.Stop(context.Background())

	// Verify it implements the EventBus interface
	var _ EventBus = bus
}

// TestNatsTopicToSubject tests the topic to subject conversion
func TestNatsTopicToSubject(t *testing.T) {
	t.Parallel()

	// topicToSubject doesn't require a connection
	natsBus := &NatsEventBus{}

	tests := []struct {
		name     string
		topic    string
		expected string
	}{
		{"exact topic", "user.created", "user.created"},
		{"wildcard topic", "user.*", "user.>"},
		{"all topics wildcard", "*", ">"},
		{"nested exact", "events.user.created", "events.user.created"},
		{"nested wildcard", "events.user.*", "events.user.>"},
		{"multiple segments with wildcard", "app.services.api.*", "app.services.api.>"},
		{"single segment", "test", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := natsBus.topicToSubject(tt.topic)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNatsEventBusLifecycle tests the lifecycle of the NATS event bus
func TestNatsEventBusLifecycle(t *testing.T) {
	url := startTestNATSServer(t)

	config := map[string]interface{}{
		"url": url,
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test starting
	err = bus.Start(ctx)
	require.NoError(t, err)

	// Test starting again (should be idempotent)
	err = bus.Start(ctx)
	require.NoError(t, err)

	// Test stopping
	err = bus.Stop(ctx)
	require.NoError(t, err)

	// Test stopping again (should be idempotent)
	err = bus.Stop(ctx)
	require.NoError(t, err)
}

// TestNatsEventBusPubSub tests basic publish/subscribe functionality
func TestNatsEventBusPubSub(t *testing.T) {
	url := startTestNATSServer(t)

	config := map[string]interface{}{
		"url": url,
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err)
	defer bus.Stop(context.Background())

	ctx := context.Background()
	err = bus.Start(ctx)
	require.NoError(t, err)

	// Create a channel to receive events
	eventReceived := make(chan Event, 1)

	// Subscribe to a topic
	handler := func(ctx context.Context, event Event) error {
		eventReceived <- event
		return nil
	}

	sub, err := bus.Subscribe(ctx, "test.topic", handler)
	require.NoError(t, err)
	require.NotNil(t, sub)

	// Give subscription time to be established
	time.Sleep(100 * time.Millisecond)

	// Publish an event
	testPayload := map[string]string{"message": "hello"}
	event := newTestCloudEvent("test.topic", testPayload)

	err = bus.Publish(ctx, event)
	require.NoError(t, err)

	// Wait for event to be received
	select {
	case receivedEvent := <-eventReceived:
		assert.Equal(t, "test.topic", receivedEvent.Type())
		assert.NotNil(t, receivedEvent.Data())
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event")
	}

	// Unsubscribe
	err = bus.Unsubscribe(ctx, sub)
	require.NoError(t, err)
}

// TestNatsEventBusWildcardSubscription tests wildcard subscriptions
func TestNatsEventBusWildcardSubscription(t *testing.T) {
	url := startTestNATSServer(t)

	config := map[string]interface{}{
		"url": url,
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err)
	defer bus.Stop(context.Background())

	ctx := context.Background()
	err = bus.Start(ctx)
	require.NoError(t, err)

	// Create a channel to receive events
	eventsReceived := make(chan Event, 10)

	// Subscribe to wildcard topic
	handler := func(ctx context.Context, event Event) error {
		eventsReceived <- event
		return nil
	}

	sub, err := bus.Subscribe(ctx, "user.*", handler)
	require.NoError(t, err)
	require.NotNil(t, sub)

	// Give subscription time to be established
	time.Sleep(100 * time.Millisecond)

	// Publish multiple events
	events := []string{"user.created", "user.updated", "user.deleted"}
	for _, topic := range events {
		event := newTestCloudEvent(topic, map[string]string{"topic": topic})
		err = bus.Publish(ctx, event)
		require.NoError(t, err)
	}

	// Wait for events to be received
	receivedCount := 0
	timeout := time.After(2 * time.Second)
	for receivedCount < len(events) {
		select {
		case <-eventsReceived:
			receivedCount++
		case <-timeout:
			t.Fatalf("Timeout waiting for events, received %d/%d", receivedCount, len(events))
		}
	}

	assert.Equal(t, len(events), receivedCount)

	// Unsubscribe
	err = bus.Unsubscribe(ctx, sub)
	require.NoError(t, err)
}

// TestNatsEventBusAsyncSubscription tests asynchronous subscriptions
func TestNatsEventBusAsyncSubscription(t *testing.T) {
	url := startTestNATSServer(t)

	config := map[string]interface{}{
		"url": url,
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err)
	defer bus.Stop(context.Background())

	ctx := context.Background()
	err = bus.Start(ctx)
	require.NoError(t, err)

	// Create a channel to receive events
	eventReceived := make(chan Event, 1)

	// Subscribe asynchronously
	handler := func(ctx context.Context, event Event) error {
		// Simulate some processing time
		time.Sleep(50 * time.Millisecond)
		eventReceived <- event
		return nil
	}

	sub, err := bus.SubscribeAsync(ctx, "async.test", handler)
	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.True(t, sub.IsAsync())

	// Give subscription time to be established
	time.Sleep(100 * time.Millisecond)

	// Publish an event
	event := newTestCloudEvent("async.test", map[string]string{"message": "async test"})

	startTime := time.Now()
	err = bus.Publish(ctx, event)
	publishDuration := time.Since(startTime)
	require.NoError(t, err)

	// Publishing should not block for async subscriptions
	assert.Less(t, publishDuration, 100*time.Millisecond)

	// Wait for event to be received
	select {
	case <-eventReceived:
		// Event received successfully
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for async event")
	}

	// Unsubscribe
	err = bus.Unsubscribe(ctx, sub)
	require.NoError(t, err)
}

// TestNatsEventBusMultipleSubscribers tests multiple subscribers on the same topic
func TestNatsEventBusMultipleSubscribers(t *testing.T) {
	url := startTestNATSServer(t)

	config := map[string]interface{}{
		"url": url,
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err)
	defer bus.Stop(context.Background())

	ctx := context.Background()
	err = bus.Start(ctx)
	require.NoError(t, err)

	// Create channels for each subscriber
	events1 := make(chan Event, 1)
	events2 := make(chan Event, 1)

	// Subscribe with first handler
	handler1 := func(ctx context.Context, event Event) error {
		events1 <- event
		return nil
	}

	sub1, err := bus.Subscribe(ctx, "multi.test", handler1)
	require.NoError(t, err)

	// Subscribe with second handler
	handler2 := func(ctx context.Context, event Event) error {
		events2 <- event
		return nil
	}

	sub2, err := bus.Subscribe(ctx, "multi.test", handler2)
	require.NoError(t, err)

	// Check subscriber count
	count := bus.SubscriberCount("multi.test")
	assert.Equal(t, 2, count)

	// Give subscriptions time to be established
	time.Sleep(100 * time.Millisecond)

	// Publish an event
	event := newTestCloudEvent("multi.test", map[string]string{"message": "test"})

	err = bus.Publish(ctx, event)
	require.NoError(t, err)

	// Both handlers should receive the event
	receivedCount := 0
	timeout := time.After(2 * time.Second)

	for receivedCount < 2 {
		select {
		case <-events1:
			receivedCount++
		case <-events2:
			receivedCount++
		case <-timeout:
			t.Fatalf("Timeout waiting for events, received %d/2", receivedCount)
		}
	}

	assert.Equal(t, 2, receivedCount)

	// Clean up
	_ = bus.Unsubscribe(ctx, sub1)
	_ = bus.Unsubscribe(ctx, sub2)
}

// TestNatsEventBusTopics tests the Topics method
func TestNatsEventBusTopics(t *testing.T) {
	url := startTestNATSServer(t)

	config := map[string]interface{}{
		"url": url,
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err)
	defer bus.Stop(context.Background())

	ctx := context.Background()
	err = bus.Start(ctx)
	require.NoError(t, err)

	// Initially no topics
	topics := bus.Topics()
	assert.Empty(t, topics)

	// Subscribe to some topics
	handler := func(ctx context.Context, event Event) error {
		return nil
	}

	sub1, _ := bus.Subscribe(ctx, "topic1", handler)
	sub2, _ := bus.Subscribe(ctx, "topic2", handler)

	topics = bus.Topics()
	assert.Len(t, topics, 2)
	assert.Contains(t, topics, "topic1")
	assert.Contains(t, topics, "topic2")

	// Clean up
	_ = bus.Unsubscribe(ctx, sub1)
	_ = bus.Unsubscribe(ctx, sub2)
}

// TestNatsConfigDefaults tests that default configuration values are properly set
func TestNatsConfigDefaults(t *testing.T) {
	url := startTestNATSServer(t)

	config := map[string]interface{}{
		"url": url,
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err)
	defer bus.Stop(context.Background())

	natsBus, ok := bus.(*NatsEventBus)
	require.True(t, ok)

	assert.Equal(t, url, natsBus.config.URL)
	assert.Equal(t, "modular-eventbus", natsBus.config.ConnectionName)
	assert.Equal(t, 10, natsBus.config.MaxReconnects)
	assert.Equal(t, 2, natsBus.config.ReconnectWait)
	assert.Equal(t, true, natsBus.config.AllowReconnect)
	assert.Equal(t, 20, natsBus.config.PingInterval)
	assert.Equal(t, 2, natsBus.config.MaxPingsOut)
	assert.Equal(t, 5, natsBus.config.SubscribeTimeout)
}

// TestNatsSubscriptionMethods tests the subscription interface methods
func TestNatsSubscriptionMethods(t *testing.T) {
	t.Parallel()

	sub := &natsSubscription{
		id:      "test-id",
		topic:   "test.topic",
		isAsync: true,
		done:    make(chan struct{}),
	}

	assert.Equal(t, "test-id", sub.ID())
	assert.Equal(t, "test.topic", sub.Topic())
	assert.True(t, sub.IsAsync())

	// Test cancel
	err := sub.Cancel()
	assert.NoError(t, err)
	assert.True(t, sub.cancelled)

	// Test cancel is idempotent
	err = sub.Cancel()
	assert.NoError(t, err)
}

// TestNatsConfigurationParsing tests various configuration scenarios
func TestNatsConfigurationParsing(t *testing.T) {
	url := startTestNATSServer(t)

	tests := []struct {
		name           string
		config         map[string]interface{}
		expectedName   string
		expectedReconn int
	}{
		{
			name:           "minimal config",
			config:         map[string]interface{}{"url": url},
			expectedName:   "modular-eventbus",
			expectedReconn: 10,
		},
		{
			name: "custom name",
			config: map[string]interface{}{
				"url":            url,
				"connectionName": "my-app",
			},
			expectedName:   "my-app",
			expectedReconn: 10,
		},
		{
			name: "custom reconnect settings",
			config: map[string]interface{}{
				"url":           url,
				"maxReconnects": 5,
			},
			expectedName:   "modular-eventbus",
			expectedReconn: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus, err := NewNatsEventBus(tt.config)
			require.NoError(t, err)

			natsBus, ok := bus.(*NatsEventBus)
			require.True(t, ok)

			assert.Equal(t, url, natsBus.config.URL)
			assert.Equal(t, tt.expectedName, natsBus.config.ConnectionName)
			assert.Equal(t, tt.expectedReconn, natsBus.config.MaxReconnects)

			_ = bus.Stop(context.Background())
		})
	}
}

// TestNatsErrorCases tests error handling
func TestNatsErrorCases(t *testing.T) {
	url := startTestNATSServer(t)

	t.Run("operations fail when not started", func(t *testing.T) {
		config := map[string]interface{}{"url": url}
		bus, err := NewNatsEventBus(config)
		require.NoError(t, err)

		natsBus, ok := bus.(*NatsEventBus)
		require.True(t, ok)

		ctx := context.Background()

		// Don't start the bus

		// Publish should fail
		event := newTestCloudEvent("test", "data")
		err = natsBus.Publish(ctx, event)
		assert.Error(t, err)
		assert.Equal(t, ErrEventBusNotStarted, err)

		// Subscribe should fail
		handler := func(ctx context.Context, event Event) error { return nil }
		_, err = natsBus.Subscribe(ctx, "test", handler)
		assert.Error(t, err)
		assert.Equal(t, ErrEventBusNotStarted, err)

		// SubscribeAsync should fail
		_, err = natsBus.SubscribeAsync(ctx, "test", handler)
		assert.Error(t, err)
		assert.Equal(t, ErrEventBusNotStarted, err)

		// Unsubscribe should fail
		err = natsBus.Unsubscribe(ctx, &natsSubscription{})
		assert.Error(t, err)
		assert.Equal(t, ErrEventBusNotStarted, err)
	})

	t.Run("nil handler rejected", func(t *testing.T) {
		config := map[string]interface{}{"url": url}
		bus, err := NewNatsEventBus(config)
		require.NoError(t, err)
		defer bus.Stop(context.Background())

		err = bus.Start(context.Background())
		require.NoError(t, err)

		ctx := context.Background()

		// Subscribe with nil handler should fail
		_, err = bus.Subscribe(ctx, "test", nil)
		assert.Error(t, err)
		assert.Equal(t, ErrEventHandlerNil, err)

		// SubscribeAsync with nil handler should fail
		_, err = bus.SubscribeAsync(ctx, "test", nil)
		assert.Error(t, err)
		assert.Equal(t, ErrEventHandlerNil, err)
	})

	t.Run("connection to bad URL fails", func(t *testing.T) {
		config := map[string]interface{}{"url": "nats://127.0.0.1:1"}
		_, err := NewNatsEventBus(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to NATS")
	})
}
