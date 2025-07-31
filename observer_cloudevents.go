// Package modular provides CloudEvents integration for the Observer pattern.
// This file extends the existing Observer pattern with CloudEvents specification
// support for standardized event format and better interoperability.
package modular

import (
	"context"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

// CloudEvent is an alias for the CloudEvents Event type for convenience
type CloudEvent = cloudevents.Event

// CloudEventObserver extends the Observer interface to also handle CloudEvents.
// This allows observers to receive both traditional ObserverEvents and CloudEvents.
type CloudEventObserver interface {
	Observer

	// OnCloudEvent is called when a CloudEvent occurs that the observer is interested in.
	// The context can be used for cancellation and timeouts.
	// Observers should handle events quickly to avoid blocking other observers.
	OnCloudEvent(ctx context.Context, event cloudevents.Event) error
}

// CloudEventSubject extends the Subject interface to emit CloudEvents.
// This allows subjects to emit both traditional ObserverEvents and CloudEvents.
type CloudEventSubject interface {
	Subject

	// NotifyCloudEventObservers sends a CloudEvent to all registered observers.
	// The notification process should be non-blocking for the caller
	// and handle observer errors gracefully.
	NotifyCloudEventObservers(ctx context.Context, event cloudevents.Event) error
}

// ToCloudEvent converts an ObserverEvent to a CloudEvent.
// This provides backward compatibility while enabling CloudEvent features.
func ToCloudEvent(observerEvent ObserverEvent) cloudevents.Event {
	event := cloudevents.NewEvent()
	
	// Set required CloudEvent attributes
	event.SetID(generateEventID())
	event.SetSource(observerEvent.Source)
	event.SetType(observerEvent.Type)
	event.SetTime(observerEvent.Timestamp)
	
	// Set data
	if observerEvent.Data != nil {
		event.SetData(cloudevents.ApplicationJSON, observerEvent.Data)
	}
	
	// Set extensions for metadata
	if observerEvent.Metadata != nil {
		for key, value := range observerEvent.Metadata {
			event.SetExtension(key, value)
		}
	}
	
	// Set CloudEvent spec version
	event.SetSpecVersion(cloudevents.VersionV1)
	
	return event
}

// FromCloudEvent converts a CloudEvent to an ObserverEvent.
// This provides backward compatibility for existing observers.
func FromCloudEvent(cloudEvent cloudevents.Event) ObserverEvent {
	observerEvent := ObserverEvent{
		Type:      cloudEvent.Type(),
		Source:    cloudEvent.Source(),
		Timestamp: cloudEvent.Time(),
		Metadata:  make(map[string]interface{}),
	}
	
	// Extract data - handle JSON unmarshaling if needed
	if cloudEvent.Data() != nil {
		// Try to unmarshal JSON data
		var data interface{}
		if err := cloudEvent.DataAs(&data); err == nil {
			observerEvent.Data = data
		} else {
			// Fallback to raw data
			observerEvent.Data = cloudEvent.Data()
		}
	}
	
	// Extract extensions as metadata
	for key, value := range cloudEvent.Extensions() {
		observerEvent.Metadata[key] = value
	}
	
	return observerEvent
}

// NewCloudEvent creates a new CloudEvent with the specified parameters.
// This is a convenience function for creating properly formatted CloudEvents.
func NewCloudEvent(eventType, source string, data interface{}, metadata map[string]interface{}) cloudevents.Event {
	event := cloudevents.NewEvent()
	
	// Set required attributes
	event.SetID(generateEventID())
	event.SetSource(source)
	event.SetType(eventType)
	event.SetTime(time.Now())
	event.SetSpecVersion(cloudevents.VersionV1)
	
	// Set data if provided
	if data != nil {
		event.SetData(cloudevents.ApplicationJSON, data)
	}
	
	// Set extensions for metadata
	if metadata != nil {
		for key, value := range metadata {
			event.SetExtension(key, value)
		}
	}
	
	return event
}

// FunctionalCloudEventObserver provides a simple way to create CloudEvent observers using functions.
// This extends FunctionalObserver to also handle CloudEvents.
type FunctionalCloudEventObserver struct {
	*FunctionalObserver
	cloudEventHandler func(ctx context.Context, event cloudevents.Event) error
}

// NewFunctionalCloudEventObserver creates a new observer that can handle both ObserverEvents and CloudEvents.
func NewFunctionalCloudEventObserver(
	id string,
	observerHandler func(ctx context.Context, event ObserverEvent) error,
	cloudEventHandler func(ctx context.Context, event cloudevents.Event) error,
) CloudEventObserver {
	return &FunctionalCloudEventObserver{
		FunctionalObserver: NewFunctionalObserver(id, observerHandler).(*FunctionalObserver),
		cloudEventHandler:  cloudEventHandler,
	}
}

// OnCloudEvent implements the CloudEventObserver interface.
func (f *FunctionalCloudEventObserver) OnCloudEvent(ctx context.Context, event cloudevents.Event) error {
	if f.cloudEventHandler != nil {
		return f.cloudEventHandler(ctx, event)
	}
	
	// Fallback to ObserverEvent handler if CloudEvent handler is not provided
	observerEvent := FromCloudEvent(event)
	return f.OnEvent(ctx, observerEvent)
}

// CloudEventConstants defines CloudEvent type constants for framework events.
// These follow CloudEvent naming conventions and can be used consistently
// across the application.
const (
	// CloudEvent types for module lifecycle
	CloudEventTypeModuleRegistered  = "com.modular.module.registered"
	CloudEventTypeModuleInitialized = "com.modular.module.initialized"
	CloudEventTypeModuleStarted     = "com.modular.module.started"
	CloudEventTypeModuleStopped     = "com.modular.module.stopped"
	CloudEventTypeModuleFailed      = "com.modular.module.failed"

	// CloudEvent types for service lifecycle
	CloudEventTypeServiceRegistered   = "com.modular.service.registered"
	CloudEventTypeServiceUnregistered = "com.modular.service.unregistered"
	CloudEventTypeServiceRequested    = "com.modular.service.requested"

	// CloudEvent types for configuration
	CloudEventTypeConfigLoaded    = "com.modular.config.loaded"
	CloudEventTypeConfigValidated = "com.modular.config.validated"
	CloudEventTypeConfigChanged   = "com.modular.config.changed"

	// CloudEvent types for application lifecycle
	CloudEventTypeApplicationStarted = "com.modular.application.started"
	CloudEventTypeApplicationStopped = "com.modular.application.stopped"
	CloudEventTypeApplicationFailed  = "com.modular.application.failed"
)

// generateEventID generates a unique identifier for CloudEvents using UUIDv7.
// UUIDv7 includes timestamp information which provides time-ordered uniqueness.
func generateEventID() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Fallback to v4 if v7 fails for any reason
		id = uuid.New()
	}
	return id.String()
}

// ValidateCloudEvent validates that a CloudEvent conforms to the specification.
// This provides validation beyond the basic CloudEvent SDK validation.
func ValidateCloudEvent(event cloudevents.Event) error {
	// Use the CloudEvent SDK's built-in validation
	if err := event.Validate(); err != nil {
		return err
	}
	
	// Additional validation could be added here for application-specific requirements
	return nil
}