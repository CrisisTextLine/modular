package eventbus

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to unmarshal a JSON string into the map that parseCloudEvent expects.
func ceMap(t *testing.T, raw string) map[string]json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(raw), &m))
	return m
}

func TestExtractString(t *testing.T) {
	t.Parallel()

	t.Run("key present and string", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{"k": json.RawMessage(`"hello"`)}
		v, ok := extractString(m, "k")
		assert.True(t, ok)
		assert.Equal(t, "hello", v)
	})

	t.Run("key absent", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{}
		v, ok := extractString(m, "k")
		assert.False(t, ok)
		assert.Equal(t, "", v)
	})

	t.Run("key present but not a string", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{"k": json.RawMessage(`123`)}
		v, ok := extractString(m, "k")
		assert.False(t, ok)
		assert.Equal(t, "", v)
	})
}

func TestIsJSONContentType(t *testing.T) {
	t.Parallel()

	t.Run("application/json", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{"datacontenttype": json.RawMessage(`"application/json"`)}
		assert.True(t, isJSONContentType(m))
	})

	t.Run("application/json with charset", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{"datacontenttype": json.RawMessage(`"application/json; charset=utf-8"`)}
		assert.True(t, isJSONContentType(m))
	})

	t.Run("non-JSON content type", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{"datacontenttype": json.RawMessage(`"text/plain"`)}
		assert.False(t, isJSONContentType(m))
	})

	t.Run("missing datacontenttype", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{}
		assert.False(t, isJSONContentType(m))
	})

	t.Run("invalid media type", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{"datacontenttype": json.RawMessage(`";;;invalid"`)}
		assert.False(t, isJSONContentType(m))
	})
}

func TestParseCloudEvent(t *testing.T) {
	t.Parallel()

	t.Run("full CloudEvent with all fields", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "conversations.conversation.started",
			"source": "platform.conversations",
			"id": "evt-ff9745302bb23718d9da693c",
			"time": "2026-02-06T23:02:35+00:00",
			"datacontenttype": "application/json",
			"data": {"id": "123", "texterId": "987", "keyword": "HELLO"}
		}`)

		event, err := parseCloudEvent(m)
		require.NoError(t, err)

		assert.Equal(t, "conversations.conversation.started", event.Topic)
		assert.NotNil(t, event.Payload)
		assert.Equal(t, "1.0", event.Metadata["ce_specversion"])
		assert.Equal(t, "conversations.conversation.started", event.Metadata["ce_type"])
		assert.Equal(t, "platform.conversations", event.Metadata["ce_source"])
		assert.Equal(t, "evt-ff9745302bb23718d9da693c", event.Metadata["ce_id"])
		assert.Equal(t, "application/json", event.Metadata["ce_datacontenttype"])

		expectedTime, err := time.Parse(time.RFC3339, "2026-02-06T23:02:35+00:00")
		require.NoError(t, err)
		assert.Equal(t, expectedTime, event.CreatedAt)

		payloadMap, ok := event.Payload.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "123", payloadMap["id"])
		assert.Equal(t, "987", payloadMap["texterId"])
		assert.Equal(t, "HELLO", payloadMap["keyword"])
	})

	t.Run("CloudEvent with extension attributes", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "user.created",
			"source": "user-service",
			"id": "abc-123",
			"tenantid": "tenant-456",
			"traceparent": "00-abc-def-01"
		}`)

		event, err := parseCloudEvent(m)
		require.NoError(t, err)

		assert.Equal(t, "user.created", event.Topic)
		assert.Equal(t, "tenant-456", event.Metadata["ce_tenantid"])
		assert.Equal(t, "00-abc-def-01", event.Metadata["ce_traceparent"])
	})

	t.Run("CloudEvent without time uses current time", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1"
		}`)

		before := time.Now()
		event, err := parseCloudEvent(m)
		after := time.Now()
		require.NoError(t, err)

		assert.False(t, event.CreatedAt.Before(before))
		assert.False(t, event.CreatedAt.After(after))
	})

	t.Run("CloudEvent with unparseable time falls back to now", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"time": "not-a-timestamp"
		}`)

		before := time.Now()
		event, err := parseCloudEvent(m)
		after := time.Now()
		require.NoError(t, err)

		assert.False(t, event.CreatedAt.IsZero())
		assert.False(t, event.CreatedAt.Before(before))
		assert.False(t, event.CreatedAt.After(after))
	})

	t.Run("CloudEvent with null data", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"data": null
		}`)

		event, err := parseCloudEvent(m)
		require.NoError(t, err)
		assert.Nil(t, event.Payload)
	})

	t.Run("CloudEvent with no data field", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1"
		}`)

		event, err := parseCloudEvent(m)
		require.NoError(t, err)
		assert.Nil(t, event.Payload)
	})

	t.Run("missing required type returns error", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{"specversion": "1.0", "source": "test", "id": "1"}`)
		_, err := parseCloudEvent(m)
		assert.ErrorIs(t, err, ErrCloudEventMissingType)
	})

	t.Run("missing required source returns error", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{"specversion": "1.0", "type": "test", "id": "1"}`)
		_, err := parseCloudEvent(m)
		assert.ErrorIs(t, err, ErrCloudEventMissingSource)
	})

	t.Run("missing required id returns error", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{"specversion": "1.0", "type": "test", "source": "test"}`)
		_, err := parseCloudEvent(m)
		assert.ErrorIs(t, err, ErrCloudEventMissingID)
	})

	t.Run("missing required specversion returns error", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{"type": "test", "source": "test", "id": "1"}`)
		_, err := parseCloudEvent(m)
		assert.ErrorIs(t, err, ErrCloudEventMissingSpecVersion)
	})

	t.Run("non-string specversion returns error", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{
			"specversion": json.RawMessage(`1.0`),
			"type":        json.RawMessage(`"test"`),
			"source":      json.RawMessage(`"src"`),
			"id":          json.RawMessage(`"1"`),
		}
		_, err := parseCloudEvent(m)
		assert.ErrorIs(t, err, ErrCloudEventMissingSpecVersion)
	})

	t.Run("non-string type returns error", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{
			"specversion": json.RawMessage(`"1.0"`),
			"type":        json.RawMessage(`42`),
			"source":      json.RawMessage(`"src"`),
			"id":          json.RawMessage(`"1"`),
		}
		_, err := parseCloudEvent(m)
		assert.ErrorIs(t, err, ErrCloudEventMissingType)
	})

	t.Run("non-string source returns error", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{
			"specversion": json.RawMessage(`"1.0"`),
			"type":        json.RawMessage(`"test"`),
			"source":      json.RawMessage(`true`),
			"id":          json.RawMessage(`"1"`),
		}
		_, err := parseCloudEvent(m)
		assert.ErrorIs(t, err, ErrCloudEventMissingSource)
	})

	t.Run("non-string id returns error", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{
			"specversion": json.RawMessage(`"1.0"`),
			"type":        json.RawMessage(`"test"`),
			"source":      json.RawMessage(`"src"`),
			"id":          json.RawMessage(`99`),
		}
		_, err := parseCloudEvent(m)
		assert.ErrorIs(t, err, ErrCloudEventMissingID)
	})

	t.Run("CloudEvent with subject attribute", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"subject": "resource-123"
		}`)

		event, err := parseCloudEvent(m)
		require.NoError(t, err)
		assert.Equal(t, "resource-123", event.Metadata["ce_subject"])
	})

	t.Run("CloudEvent with string data payload", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"data": "plain text payload"
		}`)

		event, err := parseCloudEvent(m)
		require.NoError(t, err)
		assert.Equal(t, "plain text payload", event.Payload)
	})

	t.Run("CloudEvent with array data payload", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"data": [1, 2, 3]
		}`)

		event, err := parseCloudEvent(m)
		require.NoError(t, err)
		arr, ok := event.Payload.([]interface{})
		require.True(t, ok)
		assert.Len(t, arr, 3)
	})

	t.Run("CloudEvent with data_base64 binary payload", func(t *testing.T) {
		t.Parallel()
		// "SGVsbG8gV29ybGQ=" is base64 for "Hello World"
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"data_base64": "SGVsbG8gV29ybGQ="
		}`)

		event, err := parseCloudEvent(m)
		require.NoError(t, err)
		assert.Equal(t, []byte("Hello World"), event.Payload)
	})

	t.Run("CloudEvent with data_base64 JSON payload", func(t *testing.T) {
		t.Parallel()
		// base64 of `{"key":"value"}`, with charset parameter in content type
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"datacontenttype": "application/json; charset=utf-8",
			"data_base64": "eyJrZXkiOiJ2YWx1ZSJ9"
		}`)

		event, err := parseCloudEvent(m)
		require.NoError(t, err)
		payloadMap, ok := event.Payload.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value", payloadMap["key"])
	})

	t.Run("CloudEvent with invalid data_base64 returns error", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"data_base64": "!!!not-base64!!!"
		}`)

		_, err := parseCloudEvent(m)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "data_base64")
	})

	t.Run("CloudEvent with non-string data_base64 returns error", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{
			"specversion": json.RawMessage(`"1.0"`),
			"type":        json.RawMessage(`"test"`),
			"source":      json.RawMessage(`"src"`),
			"id":          json.RawMessage(`"1"`),
			"data_base64":  json.RawMessage(`12345`),
		}
		_, err := parseCloudEvent(m)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "data_base64")
	})

	t.Run("CloudEvent with data_base64 invalid JSON content returns error", func(t *testing.T) {
		t.Parallel()
		// base64 of "not json" = "bm90IGpzb24="
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"datacontenttype": "application/json",
			"data_base64": "bm90IGpzb24="
		}`)

		_, err := parseCloudEvent(m)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "data_base64")
	})

	t.Run("data takes precedence over data_base64", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"data": {"from": "data"},
			"data_base64": "SGVsbG8="
		}`)

		event, err := parseCloudEvent(m)
		require.NoError(t, err)
		payloadMap, ok := event.Payload.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "data", payloadMap["from"])
	})

	t.Run("CloudEvent with null data_base64", func(t *testing.T) {
		t.Parallel()
		m := ceMap(t, `{
			"specversion": "1.0",
			"type": "test.event",
			"source": "test",
			"id": "1",
			"data_base64": null
		}`)

		event, err := parseCloudEvent(m)
		require.NoError(t, err)
		assert.Nil(t, event.Payload)
	})

	t.Run("invalid data field returns error", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{
			"specversion": json.RawMessage(`"1.0"`),
			"type":        json.RawMessage(`"test"`),
			"source":      json.RawMessage(`"src"`),
			"id":          json.RawMessage(`"1"`),
			"data":        json.RawMessage(`{invalid`),
		}
		_, err := parseCloudEvent(m)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "data")
	})

	t.Run("extension with non-JSON value falls back to string", func(t *testing.T) {
		t.Parallel()
		m := map[string]json.RawMessage{
			"specversion": json.RawMessage(`"1.0"`),
			"type":        json.RawMessage(`"test"`),
			"source":      json.RawMessage(`"src"`),
			"id":          json.RawMessage(`"1"`),
			"customext":   json.RawMessage(`{bad json`),
		}
		event, err := parseCloudEvent(m)
		require.NoError(t, err)
		assert.Equal(t, "{bad json", event.Metadata["ce_customext"])
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

	t.Run("valid JSON but invalid native Event returns error", func(t *testing.T) {
		t.Parallel()
		// createdAt must be a valid RFC3339 timestamp; a bare string triggers unmarshal error.
		raw := []byte(`{"topic":"t","createdAt":"not-a-time"}`)
		_, err := parseRecord(raw)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to deserialize record")
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
