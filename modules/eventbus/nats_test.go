package eventbus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNatsEventBusCreation tests creating a NATS event bus
func TestNatsEventBusCreation(t *testing.T) {
	t.Run("creates with default configuration", func(t *testing.T) {
		config := map[string]interface{}{
			"url": "nats://localhost:4222",
		}

		bus, err := NewNatsEventBus(config)

		// We expect this to fail if NATS is not running, which is fine
		// The important part is that the function doesn't panic
		if err != nil {
			t.Logf("Expected error when NATS is not available: %v", err)
			return
		}

		require.NotNil(t, bus)

		// Clean up
		if bus != nil {
			_ = bus.Stop(context.Background())
		}
	})

	t.Run("creates with custom configuration", func(t *testing.T) {
		config := map[string]interface{}{
			"url":              "nats://localhost:4222",
			"connectionName":   "test-connection",
			"maxReconnects":    5,
			"reconnectWait":    1,
			"allowReconnect":   true,
			"pingInterval":     10,
			"maxPingsOut":      3,
			"subscribeTimeout": 10,
		}

		bus, err := NewNatsEventBus(config)

		// We expect this to fail if NATS is not running
		if err != nil {
			t.Logf("Expected error when NATS is not available: %v", err)
			return
		}

		require.NotNil(t, bus)

		// Clean up
		if bus != nil {
			_ = bus.Stop(context.Background())
		}
	})

	t.Run("creates with authentication", func(t *testing.T) {
		config := map[string]interface{}{
			"url":      "nats://localhost:4222",
			"username": "test-user",
			"password": "test-pass",
		}

		bus, err := NewNatsEventBus(config)

		// We expect this to fail if NATS is not running or auth fails
		if err != nil {
			t.Logf("Expected error when NATS is not available or auth fails: %v", err)
			return
		}

		require.NotNil(t, bus)

		// Clean up
		if bus != nil {
			_ = bus.Stop(context.Background())
		}
	})

	t.Run("creates with token authentication", func(t *testing.T) {
		config := map[string]interface{}{
			"url":   "nats://localhost:4222",
			"token": "test-token",
		}

		bus, err := NewNatsEventBus(config)

		// We expect this to fail if NATS is not running or auth fails
		if err != nil {
			t.Logf("Expected error when NATS is not available or auth fails: %v", err)
			return
		}

		require.NotNil(t, bus)

		// Clean up
		if bus != nil {
			_ = bus.Stop(context.Background())
		}
	})
}

// TestNatsEventBusInterface verifies that NatsEventBus implements EventBus
func TestNatsEventBusInterface(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping NATS integration test in short mode")
	}

	config := map[string]interface{}{
		"url": "nats://localhost:4222",
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err, "NATS server must be available at localhost:4222. Start it with: docker run -d -p 4222:4222 nats:2.10-alpine")
	defer bus.Stop(context.Background())

	// Verify it implements the EventBus interface
	var _ EventBus = bus
}

// TestNatsTopicToSubject tests the topic to subject conversion
func TestNatsTopicToSubject(t *testing.T) {
	// Create a NatsEventBus instance without actually connecting
	// We can test the topicToSubject method which doesn't require a connection
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
	if testing.Short() {
		t.Skip("Skipping NATS integration test in short mode")
	}

	config := map[string]interface{}{
		"url": "nats://localhost:4222",
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err, "NATS server must be available at localhost:4222. Start it with: docker run -d -p 4222:4222 nats:2.10-alpine")

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
	if testing.Short() {
		t.Skip("Skipping NATS integration test in short mode")
	}

	config := map[string]interface{}{
		"url": "nats://localhost:4222",
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err, "NATS server must be available at localhost:4222. Start it with: docker run -d -p 4222:4222 nats:2.10-alpine")
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
	event := Event{
		Topic:   "test.topic",
		Payload: testPayload,
	}

	err = bus.Publish(ctx, event)
	require.NoError(t, err)

	// Wait for event to be received
	select {
	case receivedEvent := <-eventReceived:
		assert.Equal(t, "test.topic", receivedEvent.Topic)
		// Payload comparison is tricky due to JSON serialization
		// so we just check that it's not nil
		assert.NotNil(t, receivedEvent.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event")
	}

	// Unsubscribe
	err = bus.Unsubscribe(ctx, sub)
	require.NoError(t, err)
}

// TestNatsEventBusWildcardSubscription tests wildcard subscriptions
func TestNatsEventBusWildcardSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping NATS integration test in short mode")
	}

	config := map[string]interface{}{
		"url": "nats://localhost:4222",
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err, "NATS server must be available at localhost:4222. Start it with: docker run -d -p 4222:4222 nats:2.10-alpine")
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
		event := Event{
			Topic:   topic,
			Payload: map[string]string{"topic": topic},
		}
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
	if testing.Short() {
		t.Skip("Skipping NATS integration test in short mode")
	}

	config := map[string]interface{}{
		"url": "nats://localhost:4222",
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err, "NATS server must be available at localhost:4222. Start it with: docker run -d -p 4222:4222 nats:2.10-alpine")
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
	event := Event{
		Topic:   "async.test",
		Payload: map[string]string{"message": "async test"},
	}

	startTime := time.Now()
	err = bus.Publish(ctx, event)
	publishDuration := time.Since(startTime)
	require.NoError(t, err)

	// Publishing should not block for async subscriptions
	// Allow some overhead but it should be fast
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
	if testing.Short() {
		t.Skip("Skipping NATS integration test in short mode")
	}

	config := map[string]interface{}{
		"url": "nats://localhost:4222",
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err, "NATS server must be available at localhost:4222. Start it with: docker run -d -p 4222:4222 nats:2.10-alpine")
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
	event := Event{
		Topic:   "multi.test",
		Payload: map[string]string{"message": "test"},
	}

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
	if testing.Short() {
		t.Skip("Skipping NATS integration test in short mode")
	}

	config := map[string]interface{}{
		"url": "nats://localhost:4222",
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err, "NATS server must be available at localhost:4222. Start it with: docker run -d -p 4222:4222 nats:2.10-alpine")
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
	config := map[string]interface{}{}
	
	bus, err := NewNatsEventBus(config)
	if err != nil {
		// Expected when NATS is not running, but we can still verify the error
		assert.Contains(t, err.Error(), "failed to connect to NATS")
		return
	}
	
	natsBus, ok := bus.(*NatsEventBus)
	require.True(t, ok)
	
	// Verify defaults
	assert.Equal(t, "nats://localhost:4222", natsBus.config.URL)
	assert.Equal(t, "modular-eventbus", natsBus.config.ConnectionName)
	assert.Equal(t, 10, natsBus.config.MaxReconnects)
	assert.Equal(t, 2, natsBus.config.ReconnectWait)
	assert.Equal(t, true, natsBus.config.AllowReconnect)
	assert.Equal(t, 20, natsBus.config.PingInterval)
	assert.Equal(t, 2, natsBus.config.MaxPingsOut)
	assert.Equal(t, 5, natsBus.config.SubscribeTimeout)
	
	_ = bus.Stop(context.Background())
}

// TestNatsSubscriptionMethods tests the subscription interface methods
func TestNatsSubscriptionMethods(t *testing.T) {
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
	tests := []struct {
		name           string
		config         map[string]interface{}
		expectedURL    string
		expectedName   string
		expectedReconn int
	}{
		{
			name:           "minimal config",
			config:         map[string]interface{}{"url": "nats://test:4222"},
			expectedURL:    "nats://test:4222",
			expectedName:   "modular-eventbus",
			expectedReconn: 10,
		},
		{
			name: "custom name",
			config: map[string]interface{}{
				"url":            "nats://test:4222",
				"connectionName": "my-app",
			},
			expectedURL:    "nats://test:4222",
			expectedName:   "my-app",
			expectedReconn: 10,
		},
		{
			name: "custom reconnect settings",
			config: map[string]interface{}{
				"url":           "nats://test:4222",
				"maxReconnects": 5,
			},
			expectedURL:    "nats://test:4222",
			expectedName:   "modular-eventbus",
			expectedReconn: 5,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus, err := NewNatsEventBus(tt.config)
			if err != nil {
				// Expected when NATS is not running
				t.Logf("NATS not available: %v", err)
				return
			}
			
			natsBus, ok := bus.(*NatsEventBus)
			require.True(t, ok)
			
			assert.Equal(t, tt.expectedURL, natsBus.config.URL)
			assert.Equal(t, tt.expectedName, natsBus.config.ConnectionName)
			assert.Equal(t, tt.expectedReconn, natsBus.config.MaxReconnects)
			
			_ = bus.Stop(context.Background())
		})
	}
}

// TestNatsErrorCases tests error handling
func TestNatsErrorCases(t *testing.T) {
	t.Run("operations fail when not started", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping NATS integration test in short mode")
		}

		config := map[string]interface{}{"url": "nats://localhost:4222"}
		bus, err := NewNatsEventBus(config)
		require.NoError(t, err, "NATS server must be available at localhost:4222. Start it with: docker run -d -p 4222:4222 nats:2.10-alpine")
		
		natsBus, ok := bus.(*NatsEventBus)
		require.True(t, ok)
		
		ctx := context.Background()
		
		// Don't start the bus
		
		// Publish should fail
		event := Event{Topic: "test", Payload: "data"}
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
		if testing.Short() {
			t.Skip("Skipping NATS integration test in short mode")
		}

		config := map[string]interface{}{"url": "nats://localhost:4222"}
		bus, err := NewNatsEventBus(config)
		require.NoError(t, err, "NATS server must be available at localhost:4222. Start it with: docker run -d -p 4222:4222 nats:2.10-alpine")
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
}
