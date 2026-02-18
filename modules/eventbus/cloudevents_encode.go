package eventbus

import (
	"encoding/json"
	"fmt"
)

// isCloudEventsPayload reports whether the event's Payload is a CloudEvents
// envelope, detected by the presence of a "specversion" key in the serialized
// JSON. This mirrors the read-side detection in parseRecord().
func isCloudEventsPayload(payload interface{}) bool {
	if payload == nil {
		return false
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return false
	}
	_, ok := m["specversion"]
	return ok
}

// marshalEventData serializes an Event for publishing.
// If event.Payload is already a CloudEvents v1.0 envelope (detected by the
// presence of a "specversion" key), it is serialized directly as flat JSON.
// Otherwise the full Event struct is serialized (legacy format).
func marshalEventData(event Event) ([]byte, error) {
	if event.Payload == nil {
		data, err := json.Marshal(event)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize event: %w", err)
		}
		return data, nil
	}

	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize event payload: %w", err)
	}

	// Probe for CloudEvents specversion key.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(payloadBytes, &m); err != nil {
		// Not a JSON object — use legacy wrapping.
		data, err := json.Marshal(event)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize event: %w", err)
		}
		return data, nil
	}

	if _, ok := m["specversion"]; ok {
		return payloadBytes, nil
	}

	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize event: %w", err)
	}
	return data, nil
}
