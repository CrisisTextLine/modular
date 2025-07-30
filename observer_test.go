package modular

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestObserverEvent(t *testing.T) {
	event := ObserverEvent{
		Type:      "test.event",
		Source:    "test.source",
		Data:      "test data",
		Metadata:  map[string]interface{}{"key": "value"},
		Timestamp: time.Now(),
	}

	if event.Type != "test.event" {
		t.Errorf("Expected Type to be 'test.event', got %s", event.Type)
	}
	if event.Source != "test.source" {
		t.Errorf("Expected Source to be 'test.source', got %s", event.Source)
	}
	if event.Data != "test data" {
		t.Errorf("Expected Data to be 'test data', got %v", event.Data)
	}
	if event.Metadata["key"] != "value" {
		t.Errorf("Expected Metadata['key'] to be 'value', got %v", event.Metadata["key"])
	}
}

func TestFunctionalObserver(t *testing.T) {
	called := false
	var receivedEvent ObserverEvent

	handler := func(ctx context.Context, event ObserverEvent) error {
		called = true
		receivedEvent = event
		return nil
	}

	observer := NewFunctionalObserver("test-observer", handler)

	// Test ObserverID
	if observer.ObserverID() != "test-observer" {
		t.Errorf("Expected ObserverID to be 'test-observer', got %s", observer.ObserverID())
	}

	// Test OnEvent
	testEvent := ObserverEvent{
		Type:      "test.event",
		Source:    "test",
		Data:      "test data",
		Timestamp: time.Now(),
	}

	err := observer.OnEvent(context.Background(), testEvent)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !called {
		t.Error("Expected handler to be called")
	}

	if receivedEvent.Type != testEvent.Type {
		t.Errorf("Expected received event type to be %s, got %s", testEvent.Type, receivedEvent.Type)
	}
}

func TestFunctionalObserverWithError(t *testing.T) {
	expectedErr := errors.New("test error")
	
	handler := func(ctx context.Context, event ObserverEvent) error {
		return expectedErr
	}

	observer := NewFunctionalObserver("test-observer", handler)

	testEvent := ObserverEvent{
		Type:   "test.event",
		Source: "test",
		Data:   "test data",
	}

	err := observer.OnEvent(context.Background(), testEvent)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestEventTypeConstants(t *testing.T) {
	// Test that our event type constants are properly defined
	expectedEventTypes := map[string]string{
		"EventTypeModuleRegistered":    "module.registered",
		"EventTypeModuleInitialized":   "module.initialized", 
		"EventTypeModuleStarted":       "module.started",
		"EventTypeModuleStopped":       "module.stopped",
		"EventTypeModuleFailed":        "module.failed",
		"EventTypeServiceRegistered":   "service.registered",
		"EventTypeServiceUnregistered": "service.unregistered",
		"EventTypeServiceRequested":    "service.requested",
		"EventTypeConfigLoaded":        "config.loaded",
		"EventTypeConfigValidated":     "config.validated",
		"EventTypeConfigChanged":       "config.changed",
		"EventTypeApplicationStarted":  "application.started",
		"EventTypeApplicationStopped":  "application.stopped",
		"EventTypeApplicationFailed":   "application.failed",
	}

	actualEventTypes := map[string]string{
		"EventTypeModuleRegistered":    EventTypeModuleRegistered,
		"EventTypeModuleInitialized":   EventTypeModuleInitialized,
		"EventTypeModuleStarted":       EventTypeModuleStarted,
		"EventTypeModuleStopped":       EventTypeModuleStopped,
		"EventTypeModuleFailed":        EventTypeModuleFailed,
		"EventTypeServiceRegistered":   EventTypeServiceRegistered,
		"EventTypeServiceUnregistered": EventTypeServiceUnregistered,
		"EventTypeServiceRequested":    EventTypeServiceRequested,
		"EventTypeConfigLoaded":        EventTypeConfigLoaded,
		"EventTypeConfigValidated":     EventTypeConfigValidated,
		"EventTypeConfigChanged":       EventTypeConfigChanged,
		"EventTypeApplicationStarted":  EventTypeApplicationStarted,
		"EventTypeApplicationStopped":  EventTypeApplicationStopped,
		"EventTypeApplicationFailed":   EventTypeApplicationFailed,
	}

	for name, expected := range expectedEventTypes {
		if actual, exists := actualEventTypes[name]; !exists {
			t.Errorf("Event type constant %s is not defined", name)
		} else if actual != expected {
			t.Errorf("Event type constant %s has value %s, expected %s", name, actual, expected)
		}
	}
}

// Mock implementation for testing Subject interface
type mockSubject struct {
	observers map[string]*mockObserverRegistration
	events    []ObserverEvent
}

type mockObserverRegistration struct {
	observer   Observer
	eventTypes []string
	registered time.Time
}

func newMockSubject() *mockSubject {
	return &mockSubject{
		observers: make(map[string]*mockObserverRegistration),
		events:    make([]ObserverEvent, 0),
	}
}

func (m *mockSubject) RegisterObserver(observer Observer, eventTypes ...string) error {
	m.observers[observer.ObserverID()] = &mockObserverRegistration{
		observer:   observer,
		eventTypes: eventTypes,
		registered: time.Now(),
	}
	return nil
}

func (m *mockSubject) UnregisterObserver(observer Observer) error {
	delete(m.observers, observer.ObserverID())
	return nil
}

func (m *mockSubject) NotifyObservers(ctx context.Context, event ObserverEvent) error {
	m.events = append(m.events, event)
	
	for _, registration := range m.observers {
		// Check if observer is interested in this event type
		if len(registration.eventTypes) == 0 {
			// No filter, observer gets all events
			registration.observer.OnEvent(ctx, event)
		} else {
			// Check if event type matches observer's interests
			for _, eventType := range registration.eventTypes {
				if eventType == event.Type {
					registration.observer.OnEvent(ctx, event)
					break
				}
			}
		}
	}
	return nil
}

func (m *mockSubject) GetObservers() []ObserverInfo {
	info := make([]ObserverInfo, 0, len(m.observers))
	for _, registration := range m.observers {
		info = append(info, ObserverInfo{
			ID:           registration.observer.ObserverID(),
			EventTypes:   registration.eventTypes,
			RegisteredAt: registration.registered,
		})
	}
	return info
}

func TestSubjectObserverInteraction(t *testing.T) {
	subject := newMockSubject()
	
	// Create observers
	events1 := make([]ObserverEvent, 0)
	observer1 := NewFunctionalObserver("observer1", func(ctx context.Context, event ObserverEvent) error {
		events1 = append(events1, event)
		return nil
	})

	events2 := make([]ObserverEvent, 0)
	observer2 := NewFunctionalObserver("observer2", func(ctx context.Context, event ObserverEvent) error {
		events2 = append(events2, event)
		return nil
	})

	// Register observers - observer1 gets all events, observer2 only gets "test.specific" events
	err := subject.RegisterObserver(observer1)
	if err != nil {
		t.Fatalf("Failed to register observer1: %v", err)
	}

	err = subject.RegisterObserver(observer2, "test.specific")
	if err != nil {
		t.Fatalf("Failed to register observer2: %v", err)
	}

	// Emit a general event
	generalEvent := ObserverEvent{
		Type:   "test.general",
		Source: "test",
		Data:   "general data",
	}
	err = subject.NotifyObservers(context.Background(), generalEvent)
	if err != nil {
		t.Fatalf("Failed to notify observers: %v", err)
	}

	// Emit a specific event
	specificEvent := ObserverEvent{
		Type:   "test.specific",
		Source: "test",
		Data:   "specific data",
	}
	err = subject.NotifyObservers(context.Background(), specificEvent)
	if err != nil {
		t.Fatalf("Failed to notify observers: %v", err)
	}

	// Check observer1 received both events
	if len(events1) != 2 {
		t.Errorf("Expected observer1 to receive 2 events, got %d", len(events1))
	}

	// Check observer2 received only the specific event
	if len(events2) != 1 {
		t.Errorf("Expected observer2 to receive 1 event, got %d", len(events2))
	}
	if len(events2) > 0 && events2[0].Type != "test.specific" {
		t.Errorf("Expected observer2 to receive 'test.specific' event, got %s", events2[0].Type)
	}

	// Test GetObservers
	observerInfos := subject.GetObservers()
	if len(observerInfos) != 2 {
		t.Errorf("Expected 2 observer infos, got %d", len(observerInfos))
	}

	// Test unregistration
	err = subject.UnregisterObserver(observer1)
	if err != nil {
		t.Fatalf("Failed to unregister observer1: %v", err)
	}

	observerInfos = subject.GetObservers()
	if len(observerInfos) != 1 {
		t.Errorf("Expected 1 observer info after unregistration, got %d", len(observerInfos))
	}
}