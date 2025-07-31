// Package modular provides Observer pattern interfaces for event-driven communication.
// These interfaces complement the existing eventbus module while providing traditional
// Observer pattern functionality directly in the core framework.
package modular

import (
	"context"
	"time"
)

// ObserverEvent represents an event in the Observer pattern.
// It provides a standardized structure for events emitted by Subjects
// and consumed by Observers throughout the application lifecycle.
type ObserverEvent struct {
	// Type identifies the kind of event (e.g., "module.registered", "service.added")
	Type string `json:"type"`

	// Source identifies what generated the event (e.g., "application", "module.auth")
	Source string `json:"source"`

	// Data contains the event payload - the actual data associated with the event
	Data interface{} `json:"data"`

	// Metadata contains additional contextual information about the event
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Timestamp indicates when the event was created
	Timestamp time.Time `json:"timestamp"`
}

// Observer defines the interface for objects that want to be notified of events.
// Observers register with Subjects to receive notifications when events occur.
// This follows the traditional Observer pattern where observers are notified
// of state changes or events in subjects they're watching.
type Observer interface {
	// OnEvent is called when an event occurs that the observer is interested in.
	// The context can be used for cancellation and timeouts.
	// Observers should handle events quickly to avoid blocking other observers.
	OnEvent(ctx context.Context, event ObserverEvent) error

	// ObserverID returns a unique identifier for this observer.
	// This ID is used for registration tracking and debugging.
	ObserverID() string
}

// Subject defines the interface for objects that can be observed.
// Subjects maintain a list of observers and notify them when events occur.
// This is the core interface that event emitters implement.
type Subject interface {
	// RegisterObserver adds an observer to receive notifications.
	// Observers can optionally filter events by type using the eventTypes parameter.
	// If eventTypes is empty, the observer receives all events.
	RegisterObserver(observer Observer, eventTypes ...string) error

	// UnregisterObserver removes an observer from receiving notifications.
	// This method should be idempotent and not error if the observer
	// wasn't registered.
	UnregisterObserver(observer Observer) error

	// NotifyObservers sends an event to all registered observers.
	// The notification process should be non-blocking for the caller
	// and handle observer errors gracefully.
	NotifyObservers(ctx context.Context, event ObserverEvent) error

	// GetObservers returns information about currently registered observers.
	// This is useful for debugging and monitoring.
	GetObservers() []ObserverInfo
}

// ObserverInfo provides information about a registered observer.
// This is used for debugging, monitoring, and administrative interfaces.
type ObserverInfo struct {
	// ID is the unique identifier of the observer
	ID string `json:"id"`

	// EventTypes are the event types this observer is subscribed to.
	// Empty slice means all events.
	EventTypes []string `json:"eventTypes"`

	// RegisteredAt indicates when the observer was registered
	RegisteredAt time.Time `json:"registeredAt"`
}

// EventType constants for common application events.
// These provide a standardized vocabulary for events emitted by the core framework.
const (
	// Module lifecycle events
	EventTypeModuleRegistered  = "module.registered"
	EventTypeModuleInitialized = "module.initialized"
	EventTypeModuleStarted     = "module.started"
	EventTypeModuleStopped     = "module.stopped"
	EventTypeModuleFailed      = "module.failed"

	// Service lifecycle events
	EventTypeServiceRegistered   = "service.registered"
	EventTypeServiceUnregistered = "service.unregistered"
	EventTypeServiceRequested    = "service.requested"

	// Configuration events
	EventTypeConfigLoaded    = "config.loaded"
	EventTypeConfigValidated = "config.validated"
	EventTypeConfigChanged   = "config.changed"

	// Application lifecycle events
	EventTypeApplicationStarted = "application.started"
	EventTypeApplicationStopped = "application.stopped"
	EventTypeApplicationFailed  = "application.failed"
)

// ObservableModule is an optional interface that modules can implement
// to participate in the observer pattern. Modules implementing this interface
// can emit their own events and register observers for events they're interested in.
type ObservableModule interface {
	Module

	// RegisterObservers is called during module initialization to allow
	// the module to register as an observer for events it's interested in.
	// The subject parameter is typically the application itself.
	RegisterObservers(subject Subject) error

	// EmitEvent allows modules to emit their own events.
	// This should typically delegate to the application's NotifyObservers method.
	EmitEvent(ctx context.Context, event ObserverEvent) error
}

// FunctionalObserver provides a simple way to create observers using functions.
// This is useful for quick observer creation without defining full structs.
type FunctionalObserver struct {
	id      string
	handler func(ctx context.Context, event ObserverEvent) error
}

// NewFunctionalObserver creates a new observer that uses the provided function
// to handle events. This is a convenience constructor for simple use cases.
func NewFunctionalObserver(id string, handler func(ctx context.Context, event ObserverEvent) error) Observer {
	return &FunctionalObserver{
		id:      id,
		handler: handler,
	}
}

// OnEvent implements the Observer interface by calling the handler function.
func (f *FunctionalObserver) OnEvent(ctx context.Context, event ObserverEvent) error {
	return f.handler(ctx, event)
}

// ObserverID implements the Observer interface by returning the observer ID.
func (f *FunctionalObserver) ObserverID() string {
	return f.id
}
