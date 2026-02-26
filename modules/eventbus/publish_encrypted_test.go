package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/CrisisTextLine/modular"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubFieldEncryptor is a test FieldEncryptor that returns structured
// EncryptedFieldValue objects per the ADR-010 format.
type stubFieldEncryptor struct {
	algorithm string
	keyID     string
}

func (e *stubFieldEncryptor) EncryptFields(_ context.Context, data map[string]interface{}, fields []string) (*EncryptionResult, error) {
	out := make(map[string]interface{}, len(data))
	for k, v := range data {
		out[k] = v
	}
	encrypted := make([]string, 0, len(fields))
	for _, f := range fields {
		if _, ok := out[f]; ok {
			out[f] = EncryptedFieldValue{
				IV:         fmt.Sprintf("iv-%s", f),
				Ciphertext: fmt.Sprintf("ct-%v", data[f]),
				AuthTag:    fmt.Sprintf("tag-%s", f),
			}
			encrypted = append(encrypted, f)
		}
	}
	return &EncryptionResult{
		Data:            out,
		Algorithm:       e.algorithm,
		KeyID:           e.keyID,
		WrappedDEK:      "d2VrLWtleS1ieXRlcw==",
		EncryptedFields: encrypted,
		Context:         map[string]string{"tenant": "acme"},
	}, nil
}

// failingEncryptor always returns an error.
type failingEncryptor struct{}

func (e *failingEncryptor) EncryptFields(context.Context, map[string]interface{}, []string) (*EncryptionResult, error) {
	return nil, fmt.Errorf("encryption service unavailable")
}

func newTestModule(t *testing.T) (*EventBusModule, context.Context) {
	t.Helper()
	module := NewModule().(*EventBusModule)
	app := newMockApp()

	cfg := &EventBusConfig{
		Engine:                 "memory",
		MaxEventQueueSize:      100,
		DefaultEventBufferSize: 10,
		WorkerCount:            2,
	}
	app.RegisterConfigSection(ModuleName, modular.NewStdConfigProvider(cfg))

	err := module.Init(app)
	require.NoError(t, err)

	ctx := context.Background()
	err = module.Start(ctx)
	require.NoError(t, err)

	t.Cleanup(func() { _ = module.Stop(ctx) })
	return module, ctx
}

func TestPublishEncrypted(t *testing.T) {
	t.Run("encrypts specified fields and sets CloudEvents extensions", func(t *testing.T) {
		module, ctx := newTestModule(t)

		eventReceived := make(chan Event, 1)
		_, err := module.Subscribe(ctx, "user.created", func(ctx context.Context, event Event) error {
			eventReceived <- event
			return nil
		})
		require.NoError(t, err)

		payload := map[string]interface{}{
			"user_id": "u-123",
			"email":   "alice@example.com",
			"name":    "Alice",
		}
		enc := &stubFieldEncryptor{algorithm: "AES-256-GCM", keyID: "key-2024-01"}

		err = module.PublishEncrypted(ctx, "user.created", payload, enc, []string{"email"})
		require.NoError(t, err)

		select {
		case event := <-eventReceived:
			// Verify extensions
			assert.Equal(t, "AES-256-GCM", event.Extensions()["encryption"])
			assert.Equal(t, "key-2024-01", event.Extensions()["keyid"])
			assert.Equal(t, `["email"]`, event.Extensions()["encryptedfields"])
			assert.Equal(t, "d2VrLWtleS1ieXRlcw==", event.Extensions()["encrypteddek"])

			// Verify encryption context is always present
			var encCtx map[string]string
			require.NoError(t, json.Unmarshal([]byte(event.Extensions()["encryptioncontext"].(string)), &encCtx))
			assert.Equal(t, "acme", encCtx["tenant"])

			// Verify the email field is an EncryptedFieldValue in the data
			var data map[string]interface{}
			require.NoError(t, event.DataAs(&data))
			emailField, ok := data["email"].(map[string]interface{})
			require.True(t, ok, "encrypted field should be a JSON object")
			assert.Equal(t, "iv-email", emailField["iv"])
			assert.Equal(t, "ct-alice@example.com", emailField["ciphertext"])
			assert.Equal(t, "tag-email", emailField["auth_tag"])
			assert.Equal(t, "u-123", data["user_id"]) // unencrypted field unchanged
			assert.Equal(t, "Alice", data["name"])    // unencrypted field unchanged

		case <-time.After(2 * time.Second):
			t.Fatal("Event not received within timeout")
		}
	})

	t.Run("encryptor error is propagated", func(t *testing.T) {
		module, ctx := newTestModule(t)

		err := module.PublishEncrypted(ctx, "user.created",
			map[string]interface{}{"email": "bob@example.com"},
			&failingEncryptor{}, []string{"email"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "encrypting fields")
		assert.Contains(t, err.Error(), "encryption service unavailable")
	})

	t.Run("non-marshalable payload returns error", func(t *testing.T) {
		module, ctx := newTestModule(t)

		err := module.PublishEncrypted(ctx, "test.topic",
			make(chan int), // channels can't be marshaled
			&stubFieldEncryptor{algorithm: "AES-256-GCM", keyID: "k1"}, []string{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "marshaling payload for encryption")
	})

	t.Run("encryptioncontext is always set even when context is empty", func(t *testing.T) {
		module, ctx := newTestModule(t)

		eventReceived := make(chan Event, 1)
		_, err := module.Subscribe(ctx, "test.emptyctx", func(ctx context.Context, event Event) error {
			eventReceived <- event
			return nil
		})
		require.NoError(t, err)

		enc := &emptyContextEncryptor{}
		err = module.PublishEncrypted(ctx, "test.emptyctx",
			map[string]interface{}{"field": "value"}, enc, []string{"field"})
		require.NoError(t, err)

		select {
		case event := <-eventReceived:
			ctxVal, hasCtx := event.Extensions()["encryptioncontext"]
			assert.True(t, hasCtx, "encryptioncontext extension should always be set")
			// Empty map serializes to "{}" or "null" — either way it should be present
			assert.NotNil(t, ctxVal)
		case <-time.After(2 * time.Second):
			t.Fatal("Event not received within timeout")
		}
	})

	t.Run("multiple fields are a JSON array in extension", func(t *testing.T) {
		module, ctx := newTestModule(t)

		eventReceived := make(chan Event, 1)
		_, err := module.Subscribe(ctx, "pii.export", func(ctx context.Context, event Event) error {
			eventReceived <- event
			return nil
		})
		require.NoError(t, err)

		payload := map[string]interface{}{
			"ssn":   "123-45-6789",
			"email": "alice@example.com",
			"name":  "Alice",
		}
		enc := &stubFieldEncryptor{algorithm: "AES-256-GCM", keyID: "key-1"}

		err = module.PublishEncrypted(ctx, "pii.export", payload, enc, []string{"ssn", "email"})
		require.NoError(t, err)

		select {
		case event := <-eventReceived:
			fieldsRaw := event.Extensions()["encryptedfields"].(string)
			var fields []string
			require.NoError(t, json.Unmarshal([]byte(fieldsRaw), &fields))
			assert.ElementsMatch(t, []string{"ssn", "email"}, fields)
		case <-time.After(2 * time.Second):
			t.Fatal("Event not received within timeout")
		}
	})

	// Kinesis example: demonstrates using PublishEncrypted with a partition key,
	// which is the typical pattern for Kinesis-backed event buses where you need
	// both field-level encryption and deterministic shard routing.
	t.Run("works with partition key context for Kinesis routing", func(t *testing.T) {
		module, ctx := newTestModule(t)

		eventReceived := make(chan Event, 1)
		_, err := module.Subscribe(ctx, "order.payment", func(ctx context.Context, event Event) error {
			eventReceived <- event
			return nil
		})
		require.NoError(t, err)

		payload := map[string]interface{}{
			"order_id":    "ord-789",
			"card_number": "4111-1111-1111-1111",
			"amount":      99.99,
			"currency":    "USD",
		}
		enc := &stubFieldEncryptor{algorithm: "AES-256-GCM", keyID: "pci-key-01"}

		// Set a partition key for Kinesis shard routing, then publish encrypted.
		pubCtx := WithPartitionKey(ctx, "ord-789")
		err = module.PublishEncrypted(pubCtx, "order.payment", payload, enc, []string{"card_number"})
		require.NoError(t, err)

		select {
		case event := <-eventReceived:
			assert.Equal(t, "AES-256-GCM", event.Extensions()["encryption"])
			assert.Equal(t, "pci-key-01", event.Extensions()["keyid"])
			assert.Equal(t, `["card_number"]`, event.Extensions()["encryptedfields"])

			var data map[string]interface{}
			require.NoError(t, event.DataAs(&data))
			cardField, ok := data["card_number"].(map[string]interface{})
			require.True(t, ok, "encrypted field should be a JSON object")
			assert.Equal(t, "iv-card_number", cardField["iv"])
			assert.Equal(t, "ct-4111-1111-1111-1111", cardField["ciphertext"])
			assert.Equal(t, "tag-card_number", cardField["auth_tag"])
			assert.Equal(t, "ord-789", data["order_id"])
			assert.Equal(t, 99.99, data["amount"])

		case <-time.After(2 * time.Second):
			t.Fatal("Event not received within timeout")
		}
	})
}

func TestPublish_SetsEmptyEncryptedFieldsExtension(t *testing.T) {
	module, ctx := newTestModule(t)

	eventReceived := make(chan Event, 1)
	_, err := module.Subscribe(ctx, "user.logged_in", func(ctx context.Context, event Event) error {
		eventReceived <- event
		return nil
	})
	require.NoError(t, err)

	err = module.Publish(ctx, "user.logged_in", map[string]interface{}{"user_id": "u-1"})
	require.NoError(t, err)

	select {
	case event := <-eventReceived:
		fields, ok := event.Extensions()["encryptedfields"]
		require.True(t, ok, "encryptedfields extension must always be present")
		assert.Equal(t, "[]", fields, "unencrypted events should have encryptedfields set to empty JSON array")
	case <-time.After(2 * time.Second):
		t.Fatal("Event not received within timeout")
	}
}

// emptyContextEncryptor returns an EncryptionResult with no Context.
type emptyContextEncryptor struct{}

func (e *emptyContextEncryptor) EncryptFields(_ context.Context, data map[string]interface{}, fields []string) (*EncryptionResult, error) {
	out := make(map[string]interface{}, len(data))
	for k, v := range data {
		out[k] = v
	}
	for _, f := range fields {
		if _, ok := out[f]; ok {
			out[f] = EncryptedFieldValue{
				IV:         fmt.Sprintf("iv-%s", f),
				Ciphertext: fmt.Sprintf("ct-%v", data[f]),
				AuthTag:    fmt.Sprintf("tag-%s", f),
			}
		}
	}
	return &EncryptionResult{
		Data:            out,
		Algorithm:       "AES-256-GCM",
		KeyID:           "test-key",
		WrappedDEK:      "dGVzdC1kZWs=",
		EncryptedFields: fields,
	}, nil
}
