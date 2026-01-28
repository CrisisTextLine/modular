package eventbus

import (
	"fmt"
	"time"
)

// ==============================================================================
// EVENT OBSERVATION
// ==============================================================================
// This file handles event observation functionality, including various
// event types emitted during eventbus operations.

// Event observation step implementations
func (ctx *EventBusBDDTestContext) aMessagePublishedEventShouldBeEmitted() error {
	time.Sleep(100 * time.Millisecond) // Allow time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeMessagePublished {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeMessagePublished, eventTypes)
}

func (ctx *EventBusBDDTestContext) aMessageReceivedEventShouldBeEmitted() error {
	time.Sleep(500 * time.Millisecond) // Allow more time for async message processing and event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeMessageReceived {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeMessageReceived, eventTypes)
}

func (ctx *EventBusBDDTestContext) aSubscriptionCreatedEventShouldBeEmitted() error {
	time.Sleep(100 * time.Millisecond) // Allow time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeSubscriptionCreated {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeSubscriptionCreated, eventTypes)
}

func (ctx *EventBusBDDTestContext) theEventbusModuleStarts() error {
	// Module should already be started in the background setup
	return nil
}

func (ctx *EventBusBDDTestContext) aConfigLoadedEventShouldBeEmitted() error {
	time.Sleep(100 * time.Millisecond) // Allow time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeConfigLoaded {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeConfigLoaded, eventTypes)
}

func (ctx *EventBusBDDTestContext) aBusStartedEventShouldBeEmitted() error {
	time.Sleep(100 * time.Millisecond) // Allow time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeBusStarted {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeBusStarted, eventTypes)
}

func (ctx *EventBusBDDTestContext) aBusStoppedEventShouldBeEmitted() error {
	time.Sleep(100 * time.Millisecond) // Allow time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeBusStopped {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeBusStopped, eventTypes)
}

func (ctx *EventBusBDDTestContext) aSubscriptionRemovedEventShouldBeEmitted() error {
	time.Sleep(100 * time.Millisecond) // Allow time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeSubscriptionRemoved {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}
	return fmt.Errorf("subscription removed event not found. Available events: %v", eventTypes)
}

func (ctx *EventBusBDDTestContext) aTopicCreatedEventShouldBeEmitted() error {
	time.Sleep(100 * time.Millisecond) // Allow time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeTopicCreated {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}
	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeTopicCreated, eventTypes)
}

func (ctx *EventBusBDDTestContext) aTopicDeletedEventShouldBeEmitted() error {
	time.Sleep(100 * time.Millisecond) // Allow time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeTopicDeleted {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}
	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeTopicDeleted, eventTypes)
}

func (ctx *EventBusBDDTestContext) aMessageFailedEventShouldBeEmitted() error {
	time.Sleep(500 * time.Millisecond) // Allow more time for handler processing and event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeMessageFailed {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}
	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeMessageFailed, eventTypes)
}
