package eventbus

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

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

// extractString extracts a JSON string value from a pre-parsed map.
// Returns ("", false) if the key is absent or the value is not a JSON string.
func extractString(m map[string]json.RawMessage, key string) (string, bool) {
	raw, ok := m[key]
	if !ok {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

// parseCloudEvent maps a pre-parsed CloudEvents JSON map to an eventbus.Event.
// The caller is expected to have already unmarshalled the raw bytes into the
// map, so this function performs no redundant decoding.
//
// Mapping:
//   - type         → Event.Topic
//   - data         → Event.Payload
//   - time         → Event.CreatedAt (RFC3339; falls back to time.Now())
//   - specversion, source, id, datacontenttype, subject, and all extension
//     attributes → Event.Metadata (prefixed with "ce_")
func parseCloudEvent(m map[string]json.RawMessage) (Event, error) {
	specversion, ok := extractString(m, "specversion")
	if !ok || specversion == "" {
		return Event{}, fmt.Errorf("CloudEvent missing required 'specversion' attribute")
	}
	ceType, ok := extractString(m, "type")
	if !ok || ceType == "" {
		return Event{}, fmt.Errorf("CloudEvent missing required 'type' attribute")
	}
	source, ok := extractString(m, "source")
	if !ok || source == "" {
		return Event{}, fmt.Errorf("CloudEvent missing required 'source' attribute")
	}
	id, ok := extractString(m, "id")
	if !ok || id == "" {
		return Event{}, fmt.Errorf("CloudEvent missing required 'id' attribute")
	}

	var createdAt time.Time
	if timeStr, hasTime := extractString(m, "time"); hasTime && timeStr != "" {
		var err error
		createdAt, err = time.Parse(time.RFC3339, timeStr)
		if err != nil {
			slog.Warn("CloudEvent has unparseable 'time' attribute, using current time",
				"time", timeStr, "error", err)
			createdAt = time.Now()
		}
	} else {
		createdAt = time.Now()
	}

	var payload interface{}
	if data, hasData := m["data"]; hasData && len(data) > 0 && string(data) != "null" {
		if err := json.Unmarshal(data, &payload); err != nil {
			return Event{}, fmt.Errorf("failed to parse CloudEvent 'data' field: %w", err)
		}
	}

	// Build metadata from known attributes and extension attributes.
	metadata := make(map[string]interface{})
	metadata["ce_specversion"] = specversion
	metadata["ce_source"] = source
	metadata["ce_id"] = id
	if dct, ok := extractString(m, "datacontenttype"); ok {
		metadata["ce_datacontenttype"] = dct
	}
	if subj, ok := extractString(m, "subject"); ok {
		metadata["ce_subject"] = subj
	}

	for key, val := range m {
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
		Topic:     ceType,
		Payload:   payload,
		Metadata:  metadata,
		CreatedAt: createdAt,
	}, nil
}

// parseRecord attempts to parse raw JSON as either a CloudEvents envelope
// or a native eventbus.Event. This is the entry point used by engine
// deserialization paths. It performs a single JSON unmarshal into a generic
// map; if the map contains "specversion" the record is treated as a
// CloudEvent, otherwise it falls back to native Event deserialization.
func parseRecord(raw []byte) (Event, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return Event{}, fmt.Errorf("failed to deserialize record: %w", err)
	}

	if _, ok := m["specversion"]; ok {
		return parseCloudEvent(m)
	}

	var event Event
	if err := json.Unmarshal(raw, &event); err != nil {
		return Event{}, fmt.Errorf("failed to deserialize record: %w", err)
	}

	return event, nil
}
