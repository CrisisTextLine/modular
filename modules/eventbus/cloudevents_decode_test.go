package eventbus

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsCloudEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid CloudEvent with specversion",
			input:    `{"specversion":"1.0","type":"test","source":"test","id":"1"}`,
			expected: true,
		},
		{
			name:     "native Event without specversion",
			input:    `{"topic":"user.created","payload":{"id":"123"},"createdAt":"2026-01-01T00:00:00Z"}`,
			expected: false,
		},
		{
			name:     "empty JSON object",
			input:    `{}`,
			expected: false,
		},
		{
			name:     "invalid JSON",
			input:    `not json`,
			expected: false,
		},
		{
			name:     "array not object",
			input:    `[1,2,3]`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isCloudEvent(json.RawMessage(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseCloudEvent(t *testing.T) {
	t.Parallel()

	t.Run("full CloudEvent with all fields", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{
			"specversion": "1.0",
			"type": "conversations.conversation.started",
			"source": "platform.conversations",
			"id": "evt-ff9745302bb23718d9da693c",
			"time": "2026-02-06T23:02:35+00:00",
			"datacontenttype": "application/json",
			"data": {"id": "123", "texterId": "987", "keyword": "HELLO"}
		}`)

		event, err := parseCloudEvent(raw)
		require.NoError(t, err)

		assert.Equal(t, "conversations.conversation.started", event.Topic)
		assert.NotNil(t, event.Payload)
		assert.Equal(t, "1.0", event.Metadata["ce_specversion"])
		assert.Equal(t, "platform.conversations", event.Metadata["ce_source"])
		assert.Equal(t, "evt-ff9745302bb23718d9da693c", event.Metadata["ce_id"])
		assert.Equal(t, "application/json", event.Metadata["ce_datacontenttype"])

		expectedTime, _ := time.Parse(time.RFC3339, "2026-02-06T23:02:35+00:00")
		assert.Equal(t, expectedTime, event.CreatedAt)

		payloadMap, ok := event.Payload.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "123", payloadMap["id"])
		assert.Equal(t, "987", payloadMap["texterId"])
		assert.Equal(t, "HELLO", payloadMap["keyword"])
	})

	t.Run("CloudEvent with extension attributes", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{
			"specversion": "1.0",
			"type": "user.created",
			"source": "user-service",
			"id": "abc-123",
			"tenantid": "tenant-456",
			"traceparent": "00-abc-def-01"
		}`)

		event, err := parseCloudEvent(raw)
		require.NoError(t, err)

		assert.Equal(t, "user.created", event.Topic)
		assert.Equal(t, "tenant-456", event.Metadata["ce_tenantid"])
		assert.Equal(t, "00-abc-def-01", event.Metadata["ce_traceparent"])
	})

	t.Run("CloudEvent without time uses current time", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1"
		}`)

		before := time.Now()
		event, err := parseCloudEvent(raw)
		after := time.Now()
		require.NoError(t, err)

		assert.False(t, event.CreatedAt.Before(before))
		assert.False(t, event.CreatedAt.After(after))
	})

	t.Run("CloudEvent with unparseable time falls back to now", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"time": "not-a-timestamp"
		}`)

		before := time.Now()
		event, err := parseCloudEvent(raw)
		after := time.Now()
		require.NoError(t, err)

		assert.False(t, event.CreatedAt.IsZero())
		assert.False(t, event.CreatedAt.Before(before))
		assert.False(t, event.CreatedAt.After(after))
	})

	t.Run("CloudEvent with null data", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"data": null
		}`)

		event, err := parseCloudEvent(raw)
		require.NoError(t, err)
		assert.Nil(t, event.Payload)
	})

	t.Run("CloudEvent with no data field", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1"
		}`)

		event, err := parseCloudEvent(raw)
		require.NoError(t, err)
		assert.Nil(t, event.Payload)
	})

	t.Run("missing required type returns error", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{"specversion": "1.0", "source": "test", "id": "1"}`)
		_, err := parseCloudEvent(raw)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type")
	})

	t.Run("missing required source returns error", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{"specversion": "1.0", "type": "test", "id": "1"}`)
		_, err := parseCloudEvent(raw)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source")
	})

	t.Run("missing required id returns error", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{"specversion": "1.0", "type": "test", "source": "test"}`)
		_, err := parseCloudEvent(raw)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("missing required specversion returns error", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{"type": "test", "source": "test", "id": "1"}`)
		_, err := parseCloudEvent(raw)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "specversion")
	})

	t.Run("CloudEvent with subject attribute", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"subject": "resource-123"
		}`)

		event, err := parseCloudEvent(raw)
		require.NoError(t, err)
		assert.Equal(t, "resource-123", event.Metadata["ce_subject"])
	})

	t.Run("CloudEvent with string data payload", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"data": "plain text payload"
		}`)

		event, err := parseCloudEvent(raw)
		require.NoError(t, err)
		assert.Equal(t, "plain text payload", event.Payload)
	})

	t.Run("CloudEvent with array data payload", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"data": [1, 2, 3]
		}`)

		event, err := parseCloudEvent(raw)
		require.NoError(t, err)
		arr, ok := event.Payload.([]interface{})
		require.True(t, ok)
		assert.Len(t, arr, 3)
	})
}

func TestParseRecord(t *testing.T) {
	t.Parallel()

	t.Run("routes CloudEvent to parseCloudEvent", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`{
			"specversion": "1.0",
			"type": "order.placed",
			"source": "order-service",
			"id": "evt-123",
			"data": {"orderId": "456"}
		}`)

		event, err := parseRecord(raw)
		require.NoError(t, err)
		assert.Equal(t, "order.placed", event.Topic)
		assert.Equal(t, "1.0", event.Metadata["ce_specversion"])
	})

	t.Run("routes native Event to json.Unmarshal", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`{
			"topic": "user.created",
			"payload": {"userId": "789"},
			"metadata": {"source": "internal"},
			"createdAt": "2026-01-15T10:00:00Z"
		}`)

		event, err := parseRecord(raw)
		require.NoError(t, err)
		assert.Equal(t, "user.created", event.Topic)
		_, hasCeSpec := event.Metadata["ce_specversion"]
		assert.False(t, hasCeSpec)
		assert.Equal(t, "internal", event.Metadata["source"])
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		t.Parallel()
		_, err := parseRecord([]byte(`not json at all`))
		assert.Error(t, err)
	})

	t.Run("empty JSON object returns native Event", func(t *testing.T) {
		t.Parallel()
		event, err := parseRecord([]byte(`{}`))
		require.NoError(t, err)
		assert.Equal(t, "", event.Topic)
	})

	t.Run("native event with __topic metadata preserved", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`{
			"topic": "user.created",
			"payload": {"userId": "123"},
			"metadata": {"__topic": "user.created"},
			"createdAt": "2026-01-15T10:00:00Z"
		}`)

		event, err := parseRecord(raw)
		require.NoError(t, err)
		assert.Equal(t, "user.created", event.Topic)
		assert.Equal(t, "user.created", event.Metadata["__topic"])
	})
}
