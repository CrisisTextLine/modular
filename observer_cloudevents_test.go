package modular

import (
	"context"
	"sync"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock types for testing
type mockConfigProvider struct {
	config interface{}
}

func (m *mockConfigProvider) GetConfig() interface{} {
	return m.config
}

func (m *mockConfigProvider) GetDefaultConfig() interface{} {
	return m.config
}

type mockLogger struct {
	entries []mockLogEntry
	mu      sync.Mutex
}

type mockLogEntry struct {
	Level   string
	Message string
	Args    []interface{}
}

func (l *mockLogger) Info(msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, mockLogEntry{Level: "INFO", Message: msg, Args: args})
}

func (l *mockLogger) Error(msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, mockLogEntry{Level: "ERROR", Message: msg, Args: args})
}

func (l *mockLogger) Debug(msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, mockLogEntry{Level: "DEBUG", Message: msg, Args: args})
}

func (l *mockLogger) Warn(msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, mockLogEntry{Level: "WARN", Message: msg, Args: args})
}

type mockModule struct {
	name string
}

func (m *mockModule) Name() string {
	return m.name
}

func (m *mockModule) RegisterConfig(app Application) error {
	return nil
}

func (m *mockModule) Init(app Application) error {
	return nil
}

func (m *mockModule) Start(ctx context.Context) error {
	return nil
}

func (m *mockModule) Stop(ctx context.Context) error {
	return nil
}

func (m *mockModule) Dependencies() []string {
	return nil
}

func TestToCloudEvent(t *testing.T) {
	timestamp := time.Now()
	observerEvent := ObserverEvent{
		Type:      "test.event",
		Source:    "test.source",
		Data:      "test data",
		Metadata:  map[string]interface{}{"key": "value"},
		Timestamp: timestamp,
	}

	cloudEvent := ToCloudEvent(observerEvent)

	assert.Equal(t, "test.event", cloudEvent.Type())
	assert.Equal(t, "test.source", cloudEvent.Source())
	assert.Equal(t, timestamp, cloudEvent.Time())
	assert.Equal(t, cloudevents.VersionV1, cloudEvent.SpecVersion())
	
	// Check data
	var data string
	err := cloudEvent.DataAs(&data)
	require.NoError(t, err)
	assert.Equal(t, "test data", data)
	
	// Check extensions (metadata)
	extensions := cloudEvent.Extensions()
	assert.Equal(t, "value", extensions["key"])
}

func TestFromCloudEvent(t *testing.T) {
	cloudEvent := cloudevents.NewEvent()
	cloudEvent.SetID("test-id")
	cloudEvent.SetSource("test.source")
	cloudEvent.SetType("test.event")
	cloudEvent.SetTime(time.Now())
	cloudEvent.SetData(cloudevents.ApplicationJSON, "test data")
	cloudEvent.SetExtension("key", "value")

	observerEvent := FromCloudEvent(cloudEvent)

	assert.Equal(t, "test.event", observerEvent.Type)
	assert.Equal(t, "test.source", observerEvent.Source)
	assert.Equal(t, cloudEvent.Time(), observerEvent.Timestamp)
	assert.Equal(t, "test data", observerEvent.Data)
	assert.Equal(t, "value", observerEvent.Metadata["key"])
}

func TestNewCloudEvent(t *testing.T) {
	data := map[string]interface{}{"test": "data"}
	metadata := map[string]interface{}{"key": "value"}
	
	event := NewCloudEvent("test.event", "test.source", data, metadata)

	assert.Equal(t, "test.event", event.Type())
	assert.Equal(t, "test.source", event.Source())
	assert.Equal(t, cloudevents.VersionV1, event.SpecVersion())
	assert.NotEmpty(t, event.ID())
	assert.False(t, event.Time().IsZero())
	
	// Check data
	var eventData map[string]interface{}
	err := event.DataAs(&eventData)
	require.NoError(t, err)
	assert.Equal(t, "data", eventData["test"])
	
	// Check extensions
	extensions := event.Extensions()
	assert.Equal(t, "value", extensions["key"])
}

func TestFunctionalCloudEventObserver(t *testing.T) {
	observerEventReceived := false
	cloudEventReceived := false
	var receivedObserverEvent ObserverEvent
	var receivedCloudEvent cloudevents.Event

	observerHandler := func(ctx context.Context, event ObserverEvent) error {
		observerEventReceived = true
		receivedObserverEvent = event
		return nil
	}

	cloudEventHandler := func(ctx context.Context, event cloudevents.Event) error {
		cloudEventReceived = true
		receivedCloudEvent = event
		return nil
	}

	observer := NewFunctionalCloudEventObserver("test-observer", observerHandler, cloudEventHandler)

	// Test observer ID
	assert.Equal(t, "test-observer", observer.ObserverID())

	// Test ObserverEvent handling
	observerEvent := ObserverEvent{
		Type:      "test.event",
		Source:    "test.source",
		Data:      "test data",
		Timestamp: time.Now(),
	}

	err := observer.OnEvent(context.Background(), observerEvent)
	require.NoError(t, err)
	assert.True(t, observerEventReceived)
	assert.Equal(t, "test.event", receivedObserverEvent.Type)

	// Test CloudEvent handling
	cloudEvent := NewCloudEvent("test.cloudevent", "test.source", "cloud data", nil)
	
	err = observer.OnCloudEvent(context.Background(), cloudEvent)
	require.NoError(t, err)
	assert.True(t, cloudEventReceived)
	assert.Equal(t, "test.cloudevent", receivedCloudEvent.Type())
}

func TestObservableApplicationCloudEvents(t *testing.T) {
	app := NewObservableApplication(&mockConfigProvider{}, &mockLogger{})

	// Test observer that handles both event types
	observerEvents := []ObserverEvent{}
	cloudEvents := []cloudevents.Event{}

	observer := NewFunctionalCloudEventObserver(
		"test-observer",
		func(ctx context.Context, event ObserverEvent) error {
			observerEvents = append(observerEvents, event)
			return nil
		},
		func(ctx context.Context, event cloudevents.Event) error {
			cloudEvents = append(cloudEvents, event)
			return nil
		},
	)

	// Register observer
	err := app.RegisterObserver(observer)
	require.NoError(t, err)

	// Test NotifyCloudEventObservers
	testEvent := NewCloudEvent("test.event", "test.source", "test data", nil)
	err = app.NotifyCloudEventObservers(context.Background(), testEvent)
	require.NoError(t, err)

	// Give time for async notification
	time.Sleep(100 * time.Millisecond)

	// Should have received CloudEvent
	require.Len(t, cloudEvents, 1)
	assert.Equal(t, "test.event", cloudEvents[0].Type())
	assert.Equal(t, "test.source", cloudEvents[0].Source())

	// Regular observer should also work via fallback
	regularObserver := NewFunctionalObserver("regular-observer", func(ctx context.Context, event ObserverEvent) error {
		observerEvents = append(observerEvents, event)
		return nil
	})

	err = app.RegisterObserver(regularObserver)
	require.NoError(t, err)

	// Send another CloudEvent
	testEvent2 := NewCloudEvent("test.event2", "test.source", "test data 2", nil)
	err = app.NotifyCloudEventObservers(context.Background(), testEvent2)
	require.NoError(t, err)

	// Give time for async notification
	time.Sleep(100 * time.Millisecond)

	// Both observers should have received events
	require.Len(t, cloudEvents, 2)
	require.Len(t, observerEvents, 1) // Regular observer gets converted event
	assert.Equal(t, "test.event2", observerEvents[0].Type)
}

func TestValidateCloudEvent(t *testing.T) {
	// Valid event
	validEvent := NewCloudEvent("test.event", "test.source", nil, nil)
	err := ValidateCloudEvent(validEvent)
	assert.NoError(t, err)

	// Invalid event - missing required fields
	invalidEvent := cloudevents.NewEvent()
	err = ValidateCloudEvent(invalidEvent)
	assert.Error(t, err)
}

func TestCloudEventConstants(t *testing.T) {
	// Test that CloudEvent constants follow proper naming convention
	assert.Equal(t, "com.modular.module.registered", CloudEventTypeModuleRegistered)
	assert.Equal(t, "com.modular.service.registered", CloudEventTypeServiceRegistered)
	assert.Equal(t, "com.modular.application.started", CloudEventTypeApplicationStarted)
	
	// All CloudEvent types should start with "com.modular."
	constants := []string{
		CloudEventTypeModuleRegistered,
		CloudEventTypeModuleInitialized,
		CloudEventTypeModuleStarted,
		CloudEventTypeModuleStopped,
		CloudEventTypeModuleFailed,
		CloudEventTypeServiceRegistered,
		CloudEventTypeServiceUnregistered,
		CloudEventTypeServiceRequested,
		CloudEventTypeConfigLoaded,
		CloudEventTypeConfigValidated,
		CloudEventTypeConfigChanged,
		CloudEventTypeApplicationStarted,
		CloudEventTypeApplicationStopped,
		CloudEventTypeApplicationFailed,
	}

	for _, constant := range constants {
		assert.Contains(t, constant, "com.modular.", "CloudEvent type should follow naming convention: %s", constant)
	}
}

func TestObservableApplicationLifecycleCloudEvents(t *testing.T) {
	app := NewObservableApplication(&mockConfigProvider{}, &mockLogger{})

	// Track all events
	allEvents := []cloudevents.Event{}
	observer := NewFunctionalCloudEventObserver(
		"lifecycle-observer",
		func(ctx context.Context, event ObserverEvent) error {
			return nil // Not testing traditional events here
		},
		func(ctx context.Context, event cloudevents.Event) error {
			allEvents = append(allEvents, event)
			return nil
		},
	)

	err := app.RegisterObserver(observer)
	require.NoError(t, err)

	// Test module registration
	module := &mockModule{name: "test-module"}
	app.RegisterModule(module)

	// Test service registration
	err = app.RegisterService("test-service", "test-value")
	require.NoError(t, err)

	// Test application lifecycle
	err = app.Init()
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	err = app.Stop()
	require.NoError(t, err)

	// Give time for async events
	time.Sleep(200 * time.Millisecond)

	// Should have received multiple CloudEvents
	assert.GreaterOrEqual(t, len(allEvents), 6) // module, service, init start, init complete, start, stop

	// Check specific events
	eventTypes := make([]string, len(allEvents))
	for i, event := range allEvents {
		eventTypes[i] = event.Type()
		assert.Equal(t, "application", event.Source())
		assert.Equal(t, cloudevents.VersionV1, event.SpecVersion())
		assert.NotEmpty(t, event.ID())
		assert.False(t, event.Time().IsZero())
	}

	assert.Contains(t, eventTypes, CloudEventTypeModuleRegistered)
	assert.Contains(t, eventTypes, CloudEventTypeServiceRegistered)
	assert.Contains(t, eventTypes, CloudEventTypeConfigLoaded)
	assert.Contains(t, eventTypes, CloudEventTypeConfigValidated)
	assert.Contains(t, eventTypes, CloudEventTypeApplicationStarted)
	assert.Contains(t, eventTypes, CloudEventTypeApplicationStopped)
}