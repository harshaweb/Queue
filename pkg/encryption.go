package pkg

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// EncryptionConfig configures message encryption
type EncryptionConfig struct {
	Enabled     bool   `json:"enabled"`
	Key         []byte `json:"-"` // Never serialize the key
	Algorithm   string `json:"algorithm"`
	KeyRotation bool   `json:"key_rotation"`
}

// MessageEncryption handles encryption/decryption of queue messages
type MessageEncryption struct {
	gcm       cipher.AEAD
	keyHash   string
	algorithm string
}

// NewMessageEncryption creates a new message encryption handler
func NewMessageEncryption(config EncryptionConfig) (*MessageEncryption, error) {
	if !config.Enabled {
		return nil, nil
	}

	if len(config.Key) == 0 {
		return nil, fmt.Errorf("encryption key is required when encryption is enabled")
	}

	// Hash the key to get consistent 32 bytes for AES-256
	hasher := sha256.New()
	hasher.Write(config.Key)
	keyBytes := hasher.Sum(nil)

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	algorithm := config.Algorithm
	if algorithm == "" {
		algorithm = "AES-256-GCM"
	}

	return &MessageEncryption{
		gcm:       gcm,
		keyHash:   base64.StdEncoding.EncodeToString(keyBytes[:8]), // First 8 bytes for key identification
		algorithm: algorithm,
	}, nil
}

// Encrypt encrypts a message payload
func (me *MessageEncryption) Encrypt(plaintext []byte) ([]byte, error) {
	if me == nil {
		return plaintext, nil // No encryption configured
	}

	// Generate random nonce
	nonce := make([]byte, me.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %v", err)
	}

	// Encrypt the data
	ciphertext := me.gcm.Seal(nonce, nonce, plaintext, nil)

	// Prepend key hash and algorithm info for decryption
	encrypted := &EncryptedMessage{
		Algorithm: me.algorithm,
		KeyHash:   me.keyHash,
		Data:      base64.StdEncoding.EncodeToString(ciphertext),
	}

	return encrypted.Marshal()
}

// Decrypt decrypts a message payload
func (me *MessageEncryption) Decrypt(ciphertext []byte) ([]byte, error) {
	if me == nil {
		return ciphertext, nil // No encryption configured
	}

	// Parse encrypted message
	encrypted, err := UnmarshalEncryptedMessage(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to parse encrypted message: %v", err)
	}

	// Verify key hash
	if encrypted.KeyHash != me.keyHash {
		return nil, fmt.Errorf("key mismatch - message encrypted with different key")
	}

	// Verify algorithm
	if encrypted.Algorithm != me.algorithm {
		return nil, fmt.Errorf("algorithm mismatch - expected %s, got %s", me.algorithm, encrypted.Algorithm)
	}

	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(encrypted.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted data: %v", err)
	}

	// Extract nonce and ciphertext
	nonceSize := me.gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := me.gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %v", err)
	}

	return plaintext, nil
}

// EncryptedMessage represents an encrypted message with metadata
type EncryptedMessage struct {
	Algorithm string `json:"algorithm"`
	KeyHash   string `json:"key_hash"`
	Data      string `json:"data"`
}

// Marshal serializes encrypted message to JSON
func (em *EncryptedMessage) Marshal() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"algorithm":"%s","key_hash":"%s","data":"%s"}`,
		em.Algorithm, em.KeyHash, em.Data)), nil
}

// UnmarshalEncryptedMessage parses encrypted message from JSON
func UnmarshalEncryptedMessage(data []byte) (*EncryptedMessage, error) {
	// Simple JSON parsing for encrypted message
	str := string(data)

	// Extract algorithm
	algorithm, err := extractJSONField(str, "algorithm")
	if err != nil {
		return nil, err
	}

	// Extract key hash
	keyHash, err := extractJSONField(str, "key_hash")
	if err != nil {
		return nil, err
	}

	// Extract data
	encData, err := extractJSONField(str, "data")
	if err != nil {
		return nil, err
	}

	return &EncryptedMessage{
		Algorithm: algorithm,
		KeyHash:   keyHash,
		Data:      encData,
	}, nil
}

// extractJSONField extracts a field value from JSON string
func extractJSONField(jsonStr, field string) (string, error) {
	// Simple regex-like extraction (avoiding regex dependency)
	start := fmt.Sprintf(`"%s":"`, field)
	startIdx := findSubstring(jsonStr, start)
	if startIdx == -1 {
		return "", fmt.Errorf("field %s not found", field)
	}

	valueStart := startIdx + len(start)
	valueEnd := findSubstring(jsonStr[valueStart:], `"`)
	if valueEnd == -1 {
		return "", fmt.Errorf("malformed JSON for field %s", field)
	}

	return jsonStr[valueStart : valueStart+valueEnd], nil
}

// findSubstring finds substring index
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// GenerateEncryptionKey generates a secure random encryption key
func GenerateEncryptionKey() ([]byte, error) {
	key := make([]byte, 32) // 256-bit key
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %v", err)
	}
	return key, nil
}

// EncryptionStats provides encryption statistics
type EncryptionStats struct {
	Enabled   bool   `json:"enabled"`
	Algorithm string `json:"algorithm"`
	KeyHash   string `json:"key_hash"`
}

// Stats returns encryption statistics
func (me *MessageEncryption) Stats() EncryptionStats {
	if me == nil {
		return EncryptionStats{Enabled: false}
	}

	return EncryptionStats{
		Enabled:   true,
		Algorithm: me.algorithm,
		KeyHash:   me.keyHash,
	}
}
