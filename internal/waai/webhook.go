package waai

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

// VerifySignature verifies the HMAC-SHA256 signature from Wagy
func VerifySignature(rawBody []byte, signature string, secret string) bool {
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write(rawBody)
	expectedSignature := hex.EncodeToString(hash.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// WebhookPayload represents the structure of a Wagy webhook event
type WebhookPayload struct {
	Event  string `json:"event"`
	Source string `json:"source"`
	Data   struct {
		ID       int64  `json:"id"`
		DeviceID string `json:"device_id"`
		OwnerJID string `json:"owner_jid"`
		Content  struct {
			PhoneJID  string `json:"pn_jid"`
			Message   string `json:"content"`
			MessageID string `json:"message_id"`
			Timestamp string `json:"timestamp"`
		} `json:"content"`
	} `json:"data"`
}

// ExtractPhoneNumber extracts the clean phone number from WhatsApp JID format
func ExtractPhoneNumber(jid string) string {
	// Format: "628123456789@s.whatsapp.net"
	phone := strings.TrimSuffix(jid, "@s.whatsapp.net")
	return strings.TrimPrefix(phone, "+")
}

// ReadAndVerifyWebhook reads the request body and verifies the signature
func ReadAndVerifyWebhook(body io.ReadCloser, signature string, secret string) ([]byte, error) {
	rawBody, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	if !VerifySignature(rawBody, signature, secret) {
		return nil, fmt.Errorf("invalid signature")
	}

	return rawBody, nil
}
