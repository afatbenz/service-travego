package waai

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// VerifySignature verifies the HMAC-SHA256 signature from Wagy
func VerifySignature(rawBody []byte, signature string, secret string) bool {
	normalizedSignature := normalizeSignature(signature)
	if normalizedSignature == "" || secret == "" {
		return false
	}

	if verifyHMAC(rawBody, normalizedSignature, secret) {
		return true
	}

	normalizedBody, ok := normalizeJSONBody(rawBody)
	if !ok {
		return false
	}

	return verifyHMAC(normalizedBody, normalizedSignature, secret)
}

func verifyHMAC(body []byte, signature string, secret string) bool {
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write(body)
	expectedSignature := hex.EncodeToString(hash.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func normalizeSignature(signature string) string {
	signature = strings.TrimSpace(strings.ToLower(signature))
	signature = strings.TrimPrefix(signature, "sha256=")
	return signature
}

func normalizeJSONBody(rawBody []byte) ([]byte, bool) {
	var payload any
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return nil, false
	}

	normalizedBody, err := json.Marshal(payload)
	if err != nil {
		return nil, false
	}

	return normalizedBody, true
}

// ExtractPhoneNumber extracts the clean phone number from WhatsApp JID format
func ExtractPhoneNumber(jid string) string {
	// Format: "628123456789@s.whatsapp.net"
	phone := strings.TrimSuffix(jid, "@s.whatsapp.net")
	return strings.TrimPrefix(phone, "+")
}
