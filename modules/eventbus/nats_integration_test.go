//go:build integration
// +build integration

package eventbus

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupNATS starts a NATS server using Docker for integration testing
func setupNATS(t *testing.T) (cleanup func()) {
	t.Helper()

	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available, skipping NATS integration tests")
	}

	// Start NATS container
	containerName := fmt.Sprintf("nats-test-%d", time.Now().Unix())
	cmd := exec.Command("docker", "run", "-d", "--name", containerName, "-p", "4222:4222", "nats:2.10-alpine")
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to start NATS container: %v", err)
	}

	// Wait for NATS to be ready
	time.Sleep(3 * time.Second)

	// Return cleanup function
	return func() {
		exec.Command("docker", "stop", containerName).Run()
		exec.Command("docker", "rm", containerName).Run()
	}
}

// TestNatsIntegrationPubSub tests NATS pub/sub with a real server
func TestNatsIntegrationPubSub(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run")
	}

	cleanup := setupNATS(t)
	defer cleanup()

	config := map[string]interface{}{
		"url": "nats://localhost:4222",
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err, "Failed to create NATS event bus")
	require.NotNil(t, bus)
	defer bus.Stop(context.Background())

	ctx := context.Background()
	err = bus.Start(ctx)
	require.NoError(t, err, "Failed to start NATS event bus")

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
	defer bus.Unsubscribe(ctx, sub)

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
		assert.NotNil(t, receivedEvent.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

// TestNatsIntegrationWildcards tests NATS wildcard subscriptions with a real server
func TestNatsIntegrationWildcards(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run")
	}

	cleanup := setupNATS(t)
	defer cleanup()

	config := map[string]interface{}{
		"url": "nats://localhost:4222",
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err)
	require.NotNil(t, bus)
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
	defer bus.Unsubscribe(ctx, sub)

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
}

// TestNatsIntegrationAsync tests NATS async subscriptions with a real server
func TestNatsIntegrationAsync(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run")
	}

	cleanup := setupNATS(t)
	defer cleanup()

	config := map[string]interface{}{
		"url": "nats://localhost:4222",
	}

	bus, err := NewNatsEventBus(config)
	require.NoError(t, err)
	require.NotNil(t, bus)
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
	defer bus.Unsubscribe(ctx, sub)

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
}
