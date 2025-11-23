package helper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// EncryptData encrypts data (email, user_id) into a token
func EncryptData(email, userID string) (string, error) {
	// Get secret key from JWT_SECRET or use default
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "your-secret-key-change-in-production"
	}

	// Ensure secret key is 32 bytes for AES-256
	key := make([]byte, 32)
	copy(key, []byte(secretKey))
	if len(secretKey) < 32 {
		// Pad with zeros if shorter
		for i := len(secretKey); i < 32; i++ {
			key[i] = 0
		}
	}

	// Create data to encrypt
	data := map[string]string{
		"email":   email,
		"user_id": userID,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to create nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, jsonData, nil)

	// Encode to base64
	token := base64.URLEncoding.EncodeToString(ciphertext)

	return token, nil
}

// DecryptData decrypts token to get email and user_id
func DecryptData(token string) (email, userID string, err error) {
	// Get secret key from JWT_SECRET or use default
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "your-secret-key-change-in-production"
	}

	// Ensure secret key is 32 bytes for AES-256
	key := make([]byte, 32)
	copy(key, []byte(secretKey))
	if len(secretKey) < 32 {
		// Pad with zeros if shorter
		for i := len(secretKey); i < 32; i++ {
			key[i] = 0
		}
	}

	// Decode from base64
	ciphertext, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode token: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt: %w", err)
	}

	// Unmarshal JSON
	var data map[string]string
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal data: %w", err)
	}

	email = data["email"]
	userID = data["user_id"]

	if email == "" || userID == "" {
		return "", "", fmt.Errorf("invalid token data")
	}

	return email, userID, nil
}
