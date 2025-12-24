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
    "time"
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

	// Get nonce size
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt: %w", err)
	}

	// Unmarshal
	var data map[string]string
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return data["email"], data["user_id"], nil
}

// EncryptString encrypts a simple string
func EncryptString(text string) (string, error) {
	// Get secret key from JWT_SECRET or use default
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "your-secret-key-change-in-production"
	}

	// Ensure secret key is 32 bytes for AES-256
	key := make([]byte, 32)
	copy(key, []byte(secretKey))
	if len(secretKey) < 32 {
		for i := len(secretKey); i < 32; i++ {
			key[i] = 0
		}
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
	ciphertext := gcm.Seal(nonce, nonce, []byte(text), nil)

	// Encode to base64
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts a simple string
func DecryptString(cryptoText string) (string, error) {
	// Get secret key from JWT_SECRET or use default
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "your-secret-key-change-in-production"
	}

	// Ensure secret key is 32 bytes for AES-256
	key := make([]byte, 32)
	copy(key, []byte(secretKey))
	if len(secretKey) < 32 {
		for i := len(secretKey); i < 32; i++ {
			key[i] = 0
		}
	}

	// Decode from base64
	ciphertext, err := base64.URLEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", fmt.Errorf("failed to decode token: %w", err)
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

	// Get nonce size
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// ResetPasswordTokenData represents the data in a reset password token
type ResetPasswordTokenData struct {
	Email         string
	UserID        string
	ExpiryMinutes int64 // Unix timestamp in minutes
}

// EncryptResetPasswordToken encrypts email, user_id, and expiryMinutes into a token
func EncryptResetPasswordToken(email, userID string, expiryMinutes int) (string, error) {
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

	// Calculate expiry timestamp (current time + expiryMinutes)
	expiryTimestamp := time.Now().Unix() + int64(expiryMinutes*60)

	// Create data to encrypt
	data := map[string]interface{}{
		"email":          email,
		"user_id":        userID,
		"expiry_minutes": expiryTimestamp,
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

// DecryptResetPasswordToken decrypts token to get email, user_id, and expiryMinutes
// Returns error if token is expired
func DecryptResetPasswordToken(token string) (email, userID string, expiryMinutes int64, err error) {
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
		return "", "", 0, fmt.Errorf("failed to decode token: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", "", 0, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to decrypt: %w", err)
	}

	// Unmarshal JSON
	var data map[string]interface{}
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return "", "", 0, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	// Extract email
	emailVal, ok := data["email"].(string)
	if !ok {
		return "", "", 0, fmt.Errorf("invalid token data: email not found")
	}
	email = emailVal

	// Extract user_id
	userIDVal, ok := data["user_id"].(string)
	if !ok {
		return "", "", 0, fmt.Errorf("invalid token data: user_id not found")
	}
	userID = userIDVal

	// Extract expiry_minutes (as float64 from JSON, then convert to int64)
	expiryVal, ok := data["expiry_minutes"]
	if !ok {
		return "", "", 0, fmt.Errorf("invalid token data: expiry_minutes not found")
	}

	// Handle both float64 (from JSON) and int64
	var expiryTimestamp int64
	switch v := expiryVal.(type) {
	case float64:
		expiryTimestamp = int64(v)
	case int64:
		expiryTimestamp = v
	default:
		return "", "", 0, fmt.Errorf("invalid token data: expiry_minutes invalid type")
	}
	expiryMinutes = expiryTimestamp

	// Validate expiry
	currentTime := time.Now().Unix()
	if expiryTimestamp < currentTime {
		return "", "", 0, fmt.Errorf("token expired")
	}

	if email == "" || userID == "" {
		return "", "", 0, fmt.Errorf("invalid token data")
	}

	return email, userID, expiryMinutes, nil
}
// AuthSensitiveData represents sensitive auth data to be encrypted
type AuthSensitiveData struct {
    OrganizationID   string `json:"organization_id"`
    UserID           string `json:"user_id"`
    OrganizationRole int    `json:"organization_role"`
    IsAdmin          bool   `json:"is_admin"`
}

// EncryptAuthSensitiveData encrypts sensitive auth data into a token (AES-256-GCM, base64url)
func EncryptAuthSensitiveData(data AuthSensitiveData) (string, error) {
    secretKey := os.Getenv("JWT_SECRET")
    if secretKey == "" {
        secretKey = "your-secret-key-change-in-production"
    }

    key := make([]byte, 32)
    copy(key, []byte(secretKey))
    if len(secretKey) < 32 {
        for i := len(secretKey); i < 32; i++ {
            key[i] = 0
        }
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        return "", fmt.Errorf("failed to marshal data: %w", err)
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return "", fmt.Errorf("failed to create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("failed to create GCM: %w", err)
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", fmt.Errorf("failed to create nonce: %w", err)
    }

    ciphertext := gcm.Seal(nonce, nonce, jsonData, nil)
    token := base64.URLEncoding.EncodeToString(ciphertext)
    return token, nil
}

// DecryptAuthSensitiveData decrypts token to retrieve sensitive auth data
func DecryptAuthSensitiveData(token string) (AuthSensitiveData, error) {
    secretKey := os.Getenv("JWT_SECRET")
    if secretKey == "" {
        secretKey = "your-secret-key-change-in-production"
    }

    key := make([]byte, 32)
    copy(key, []byte(secretKey))
    if len(secretKey) < 32 {
        for i := len(secretKey); i < 32; i++ {
            key[i] = 0
        }
    }

    ciphertext, err := base64.URLEncoding.DecodeString(token)
    if err != nil {
        return AuthSensitiveData{}, fmt.Errorf("failed to decode token: %w", err)
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return AuthSensitiveData{}, fmt.Errorf("failed to create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return AuthSensitiveData{}, fmt.Errorf("failed to create GCM: %w", err)
    }

    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return AuthSensitiveData{}, fmt.Errorf("ciphertext too short")
    }

    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return AuthSensitiveData{}, fmt.Errorf("failed to decrypt: %w", err)
    }

    var data AuthSensitiveData
    if err := json.Unmarshal(plaintext, &data); err != nil {
        return AuthSensitiveData{}, fmt.Errorf("failed to unmarshal data: %w", err)
    }

    return data, nil
}
