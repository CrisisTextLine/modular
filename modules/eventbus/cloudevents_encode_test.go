package eventbus

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalEventData_CloudEventsPayload(t *testing.T) {
	ce := map[string]interface{}{
		"specversion":     "1.0",
		"type":            "messaging.texter-message.received",
		"source":          "/chimera/messaging",
		"id":              "test-id-123",
		"datacontenttype": "application/json",
		"data": map[string]interface{}{
			"messageId": "msg-456",
		},
	}

	event := Event{
		Topic:   "messaging.texter-message.received",
		Payload: ce,
	}

	data, err := marshalEventData(event)
	require.NoError(t, err)

	// Should be flat CloudEvents JSON, not wrapped in Event envelope.
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Contains(t, m, "specversion", "output should contain specversion at top level")
	assert.Contains(t, m, "type", "output should contain type at top level")
	assert.Contains(t, m, "source", "output should contain source at top level")
	assert.NotContains(t, m, "topic", "output should not contain Event.Topic wrapper")
	assert.NotContains(t, m, "payload", "output should not contain Event.Payload wrapper")
}

func TestMarshalEventData_NativePayload(t *testing.T) {
	event := Event{
		Topic: "user.created",
		Payload: map[string]interface{}{
			"username": "alice",
		},
	}

	data, err := marshalEventData(event)
	require.NoError(t, err)

	// Should be wrapped in Event envelope (legacy format).
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Contains(t, m, "topic", "output should contain Event.Topic")
	assert.Contains(t, m, "payload", "output should contain Event.Payload")
	assert.NotContains(t, m, "specversion", "output should not contain specversion at top level")
}

func TestMarshalEventData_NilPayload(t *testing.T) {
	event := Event{
		Topic:   "user.created",
		Payload: nil,
	}

	data, err := marshalEventData(event)
	require.NoError(t, err)

	// Should be wrapped in Event envelope (legacy format).
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Contains(t, m, "topic")
}

func TestMarshalEventData_StringPayload(t *testing.T) {
	event := Event{
		Topic:   "user.created",
		Payload: "just a string",
	}

	data, err := marshalEventData(event)
	require.NoError(t, err)

	// A string payload is not a JSON object, so it should use legacy wrapping.
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Contains(t, m, "topic", "string payload should produce legacy Event envelope")
	assert.Contains(t, m, "payload")
}

func TestIsCloudEventsPayload(t *testing.T) {
	tests := []struct {
		name     string
		payload  interface{}
		expected bool
	}{
		{
			name:     "nil payload",
			payload:  nil,
			expected: false,
		},
		{
			name:     "string payload",
			payload:  "hello",
			expected: false,
		},
		{
			name: "map without specversion",
			payload: map[string]interface{}{
				"username": "alice",
			},
			expected: false,
		},
		{
			name: "map with specversion",
			payload: map[string]interface{}{
				"specversion": "1.0",
				"type":        "test.event",
				"source":      "/test",
				"id":          "123",
			},
			expected: true,
		},
		{
			name: "struct with specversion json tag",
			payload: struct {
				SpecVersion string `json:"specversion"`
				Type        string `json:"type"`
			}{
				SpecVersion: "1.0",
				Type:        "test.event",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isCloudEventsPayload(tt.payload))
		})
	}
}
