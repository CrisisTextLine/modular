package eventbus

import "context"

// FieldEncryptor defines the interface for encrypting specific fields within event data.
// Implementations handle key management, algorithm selection, and field-level encryption,
// allowing different encryption strategies to be used per-call when publishing events.
type FieldEncryptor interface {
	// EncryptFields encrypts the specified fields within the provided data map.
	// The fields parameter identifies which keys in data should be encrypted.
	// Returns an EncryptionResult containing the modified data and encryption metadata.
	EncryptFields(ctx context.Context, data map[string]interface{}, fields []string) (*EncryptionResult, error)
}

// EncryptedFieldValue represents the structured format for an encrypted field value
// as defined by ADR-010. Each encrypted field in the event data should contain an
// instance of this struct rather than an opaque string.
type EncryptedFieldValue struct {
	IV         string `json:"iv"`
	Ciphertext string `json:"ciphertext"`
	AuthTag    string `json:"auth_tag"`
}

// EncryptionResult holds the output of a field encryption operation, including the
// modified data and metadata needed by consumers to decrypt the fields.
type EncryptionResult struct {
	// Data is the payload with specified fields replaced by EncryptedFieldValue
	// structs for encrypted fields. Unencrypted fields retain their original types.
	Data map[string]interface{}

	// Algorithm identifies the encryption algorithm used (e.g., "AES-256-GCM").
	Algorithm string

	// KeyID identifies the key used for encryption, enabling key rotation.
	KeyID string

	// WrappedDEK is the wrapped (encrypted) data encryption key, base64-encoded.
	// Consumers unwrap this with the key identified by KeyID to decrypt fields.
	WrappedDEK string

	// EncryptedFields lists the field names that were encrypted.
	EncryptedFields []string

	// Context holds additional encryption context (e.g., AAD) as key-value pairs.
	// This metadata is required during decryption to verify authenticity.
	Context map[string]string
}
