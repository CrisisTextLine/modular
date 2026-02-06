package eventbus

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// cloudEventEnvelope is a lightweight representation of a CloudEvents v1.0
// JSON envelope. Only the required and commonly-used attributes are given
// dedicated fields; extension attributes are captured separately.
type cloudEventEnvelope struct {
	SpecVersion     string          `json:"specversion"`
	Type            string          `json:"type"`
	Source          string          `json:"source"`
	ID              string          `json:"id"`
	Time            string          `json:"time,omitempty"`
	DataContentType string          `json:"datacontenttype,omitempty"`
	Data            json.RawMessage `json:"data,omitempty"`
	Subject         string          `json:"subject,omitempty"`
}

// knownCloudEventKeys are the CloudEvents spec-defined keys that have
// dedicated handling. Anything else is treated as an extension attribute.
var knownCloudEventKeys = map[string]bool{
	"specversion":     true,
	"type":            true,
	"source":          true,
	"id":              true,
	"time":            true,
	"datacontenttype": true,
	"data":            true,
	"data_base64":     true,
	"subject":         true,
}

// isCloudEvent checks whether raw JSON contains a CloudEvents envelope
// by probing for the required "specversion" key.
func isCloudEvent(raw json.RawMessage) bool {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	_, ok := probe["specversion"]
	return ok
}

// parseCloudEvent maps a CloudEvents JSON envelope to an eventbus.Event.
//
// Mapping:
//   - type         → Event.Topic
//   - data         → Event.Payload
//   - time         → Event.CreatedAt (RFC3339; falls back to time.Now())
//   - specversion, source, id, datacontenttype, subject, and all extension
//     attributes → Event.Metadata (prefixed with "ce_")
func parseCloudEvent(raw json.RawMessage) (Event, error) {
	var ce cloudEventEnvelope
	if err := json.Unmarshal(raw, &ce); err != nil {
		return Event{}, fmt.Errorf("failed to parse CloudEvent envelope: %w", err)
	}

	if ce.SpecVersion == "" {
		return Event{}, fmt.Errorf("CloudEvent missing required 'specversion' attribute")
	}
	if ce.Type == "" {
		return Event{}, fmt.Errorf("CloudEvent missing required 'type' attribute")
	}
	if ce.Source == "" {
		return Event{}, fmt.Errorf("CloudEvent missing required 'source' attribute")
	}
	if ce.ID == "" {
		return Event{}, fmt.Errorf("CloudEvent missing required 'id' attribute")
	}

	var createdAt time.Time
	if ce.Time != "" {
		var err error
		createdAt, err = time.Parse(time.RFC3339, ce.Time)
		if err != nil {
			slog.Warn("CloudEvent has unparseable 'time' attribute, using current time",
				"time", ce.Time, "error", err)
			createdAt = time.Now()
		}
	} else {
		createdAt = time.Now()
	}

	var payload interface{}
	if len(ce.Data) > 0 && string(ce.Data) != "null" {
		if err := json.Unmarshal(ce.Data, &payload); err != nil {
			return Event{}, fmt.Errorf("failed to parse CloudEvent 'data' field: %w", err)
		}
	}

	// Build metadata from known attributes and extension attributes.
	var fullMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fullMap); err != nil {
		return Event{}, fmt.Errorf("failed to parse CloudEvent for extensions: %w", err)
	}

	metadata := make(map[string]interface{})
	metadata["ce_specversion"] = ce.SpecVersion
	metadata["ce_source"] = ce.Source
	metadata["ce_id"] = ce.ID
	if ce.DataContentType != "" {
		metadata["ce_datacontenttype"] = ce.DataContentType
	}
	if ce.Subject != "" {
		metadata["ce_subject"] = ce.Subject
	}

	for key, val := range fullMap {
		if knownCloudEventKeys[key] {
			continue
		}
		var extVal interface{}
		if err := json.Unmarshal(val, &extVal); err != nil {
			metadata["ce_"+key] = string(val)
		} else {
			metadata["ce_"+key] = extVal
		}
	}

	return Event{
		Topic:     ce.Type,
		Payload:   payload,
		Metadata:  metadata,
		CreatedAt: createdAt,
	}, nil
}

// parseRecord attempts to parse raw JSON as either a CloudEvents envelope
// or a native eventbus.Event. This is the entry point used by engine
// deserialization paths.
func parseRecord(raw []byte) (Event, error) {
	if isCloudEvent(json.RawMessage(raw)) {
		return parseCloudEvent(json.RawMessage(raw))
	}

	var event Event
	if err := json.Unmarshal(raw, &event); err != nil {
		return Event{}, fmt.Errorf("failed to deserialize record: %w", err)
	}

	return event, nil
}
