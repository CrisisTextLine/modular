package eventbus

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// KMSClient is the subset of the AWS KMS API needed for envelope encryption.
// In production, use *kms.Client from github.com/aws/aws-sdk-go-v2/service/kms.
type KMSClient interface {
	// GenerateDataKey creates a new DEK, returning both the plaintext key
	// (for local encryption) and the KMS-wrapped ciphertext key (to attach
	// to the event so consumers can decrypt).
	GenerateDataKey(ctx context.Context, keyID string, encCtx map[string]string) (plaintext, ciphertextBlob []byte, err error)
}

// KMSFieldEncryptor implements FieldEncryptor using AWS KMS envelope
// encryption. For each call it generates a fresh data encryption key (DEK),
// encrypts the requested fields locally with AES-256-GCM, and returns the
// KMS-wrapped DEK so consumers can call KMS Decrypt to recover it.
//
// Usage with the eventbus module:
//
//	enc := &KMSFieldEncryptor{
//	    Client:      kmsClient,
//	    CMKArn:      "arn:aws:kms:us-east-1:123456789:key/abc-123",
//	    Environment: "prod",
//	    AffiliateID: "ctl",
//	}
//	err := bus.PublishEncrypted(ctx, "messaging.texter-message.received", payload, enc, []string{"messageBody"})
type KMSFieldEncryptor struct {
	Client      KMSClient
	CMKArn      string // KMS customer master key ARN
	Environment string // e.g. "prod", "staging"
	AffiliateID string // tenant/affiliate identifier
}

// EncryptFields implements FieldEncryptor.
//
// The flow is:
//  1. Build the encryption context (bound to the event's source and type via AAD).
//  2. Ask KMS to generate a 256-bit data encryption key (DEK).
//  3. For each requested field, encrypt the JSON-serialized value with AES-256-GCM
//     using the plaintext DEK, then replace the field with an EncryptedFieldValue.
//  4. Return the wrapped DEK and metadata so PublishEncrypted can set the
//     CloudEvents extensions.
func (e *KMSFieldEncryptor) EncryptFields(ctx context.Context, data map[string]interface{}, fields []string) (*EncryptionResult, error) {
	encCtx := map[string]string{
		"purpose":     "event-encryption",
		"eventSource": "messaging",
		"eventType":   "messaging.texter-message.received",
		"environment": e.Environment,
		"affiliateId": e.AffiliateID,
	}

	// Step 1: Generate a fresh DEK via KMS.
	plainDEK, wrappedDEK, err := e.Client.GenerateDataKey(ctx, e.CMKArn, encCtx)
	if err != nil {
		return nil, fmt.Errorf("KMS GenerateDataKey: %w", err)
	}

	// Step 2: Create an AES-256-GCM cipher from the plaintext DEK.
	block, err := aes.NewCipher(plainDEK)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	// Step 3: Encrypt each requested field.
	out := make(map[string]interface{}, len(data))
	for k, v := range data {
		out[k] = v
	}

	encrypted := make([]string, 0, len(fields))
	for _, f := range fields {
		val, ok := out[f]
		if !ok {
			continue
		}

		// Serialize the field value to JSON bytes for encryption.
		plaintext := []byte(fmt.Sprintf("%v", val))

		// Generate a random nonce (IV) for each field.
		nonce := make([]byte, gcm.NonceSize())
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return nil, fmt.Errorf("generating nonce for field %q: %w", f, err)
		}

		// Seal encrypts and authenticates plaintext, appending the auth tag.
		ciphertextWithTag := gcm.Seal(nil, nonce, plaintext, nil)

		// Split ciphertext and auth tag (GCM appends the 16-byte tag).
		tagSize := gcm.Overhead()
		ct := ciphertextWithTag[:len(ciphertextWithTag)-tagSize]
		tag := ciphertextWithTag[len(ciphertextWithTag)-tagSize:]

		out[f] = EncryptedFieldValue{
			IV:         base64.StdEncoding.EncodeToString(nonce),
			Ciphertext: base64.StdEncoding.EncodeToString(ct),
			AuthTag:    base64.StdEncoding.EncodeToString(tag),
		}
		encrypted = append(encrypted, f)
	}

	return &EncryptionResult{
		Data:            out,
		Algorithm:       "aes-256-gcm",
		KeyID:           e.CMKArn,
		WrappedDEK:      base64.StdEncoding.EncodeToString(wrappedDEK),
		EncryptedFields: encrypted,
		Context:         encCtx,
	}, nil
}

// --- Fake KMS client for testing (simulates GenerateDataKey) ---

type fakeKMSClient struct{}

func (f *fakeKMSClient) GenerateDataKey(_ context.Context, _ string, _ map[string]string) ([]byte, []byte, error) {
	// In production, AWS KMS returns a 32-byte plaintext DEK and its
	// KMS-wrapped ciphertext blob. Here we simulate both.
	plaintext := make([]byte, 32) // AES-256 = 32 bytes
	if _, err := io.ReadFull(rand.Reader, plaintext); err != nil {
		return nil, nil, err
	}
	// Simulate the wrapped DEK (in reality, this comes from KMS).
	wrapped := make([]byte, 64)
	if _, err := io.ReadFull(rand.Reader, wrapped); err != nil {
		return nil, nil, err
	}
	return plaintext, wrapped, nil
}

// TestPublishEncrypted_KMSEnvelopeEncryption demonstrates end-to-end usage of
// KMS envelope encryption with PublishEncrypted. It publishes an event that
// matches the ADR-010 shape:
//
//	{
//	  "type": "messaging.texter-message.received",
//	  "encryption": "aes-256-gcm",
//	  "keyid": "arn:aws:kms:us-east-1:123456789:key/abc-123",
//	  "encryptedfields": ["messageBody"],
//	  "encrypteddek": "<base64-wrapped-DEK>",
//	  "encryptioncontext": {"purpose":"event-encryption", ...},
//	  "data": {
//	    "messageId": "8665e36c-...",
//	    "messageBody": {"iv":"...","ciphertext":"...","auth_tag":"..."},
//	    "texterId": "d9b6fcf6-..."
//	  }
//	}
func TestPublishEncrypted_KMSEnvelopeEncryption(t *testing.T) {
	module, ctx := newTestModule(t)

	received := make(chan Event, 1)
	_, err := module.Subscribe(ctx, "messaging.texter-message.received", func(_ context.Context, event Event) error {
		received <- event
		return nil
	})
	require.NoError(t, err)

	// --- Build the encryptor (swap fakeKMSClient for real *kms.Client in prod) ---
	enc := &KMSFieldEncryptor{
		Client:      &fakeKMSClient{},
		CMKArn:      "arn:aws:kms:us-east-1:123456789:key/abc-123",
		Environment: "prod",
		AffiliateID: "ctl",
	}

	payload := map[string]interface{}{
		"messageId":   "8665e36c-2638-46ed-ae8f-bf97fb354133",
		"messageBody": "Hey, I need help with something personal...",
		"texterId":    "d9b6fcf6-89ff-48aa-8c9f-4e0287be31c8",
	}

	err = module.PublishEncrypted(ctx, "messaging.texter-message.received", payload, enc, []string{"messageBody"})
	require.NoError(t, err)

	select {
	case event := <-received:
		// --- CloudEvents extensions ---
		assert.Equal(t, "aes-256-gcm", event.Extensions()["encryption"])
		assert.Equal(t, "arn:aws:kms:us-east-1:123456789:key/abc-123", event.Extensions()["keyid"])
		assert.Equal(t, `["messageBody"]`, event.Extensions()["encryptedfields"])

		// Wrapped DEK is present and base64-decodable.
		dekStr, ok := event.Extensions()["encrypteddek"].(string)
		require.True(t, ok)
		dekBytes, err := base64.StdEncoding.DecodeString(dekStr)
		require.NoError(t, err)
		assert.Len(t, dekBytes, 64, "fake KMS returns 64-byte wrapped DEK")

		// Encryption context contains all expected keys.
		var encCtx map[string]string
		require.NoError(t, json.Unmarshal([]byte(event.Extensions()["encryptioncontext"].(string)), &encCtx))
		assert.Equal(t, "event-encryption", encCtx["purpose"])
		assert.Equal(t, "messaging", encCtx["eventSource"])
		assert.Equal(t, "messaging.texter-message.received", encCtx["eventType"])
		assert.Equal(t, "prod", encCtx["environment"])
		assert.Equal(t, "ctl", encCtx["affiliateId"])

		// --- Data payload ---
		var data map[string]interface{}
		require.NoError(t, event.DataAs(&data))

		// Unencrypted fields pass through unchanged.
		assert.Equal(t, "8665e36c-2638-46ed-ae8f-bf97fb354133", data["messageId"])
		assert.Equal(t, "d9b6fcf6-89ff-48aa-8c9f-4e0287be31c8", data["texterId"])

		// messageBody is now a structured {iv, ciphertext, auth_tag} object.
		body, ok := data["messageBody"].(map[string]interface{})
		require.True(t, ok, "messageBody should be a JSON object, got %T", data["messageBody"])

		// Each component is a non-empty base64 string.
		for _, key := range []string{"iv", "ciphertext", "auth_tag"} {
			val, exists := body[key]
			require.True(t, exists, "messageBody.%s must be present", key)
			str, ok := val.(string)
			require.True(t, ok, "messageBody.%s must be a string", key)
			_, err := base64.StdEncoding.DecodeString(str)
			assert.NoError(t, err, "messageBody.%s must be valid base64", key)
		}

	case <-time.After(2 * time.Second):
		t.Fatal("Event not received within timeout")
	}
}
