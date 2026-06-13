package waai

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifySignatureMatchesRawBody(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"event":"message.received","source":"whatsapp"}`)
	signature := signatureForBody(body, secret)

	if !VerifySignature(body, signature, secret) {
		t.Fatal("expected raw body signature to be valid")
	}
}

func TestVerifySignatureMatchesNormalizedJSONBody(t *testing.T) {
	secret := "test-secret"
	rawBody := []byte("{\n  \"event\": \"message.received\",\n  \"source\": \"whatsapp\"\n}")
	normalizedBody := []byte(`{"event":"message.received","source":"whatsapp"}`)
	signature := signatureForBody(normalizedBody, secret)

	if !VerifySignature(rawBody, signature, secret) {
		t.Fatal("expected normalized JSON signature to be valid")
	}
}

func TestVerifySignatureAcceptsSha256Prefix(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"event":"message.received"}`)
	signature := "sha256=" + signatureForBody(body, secret)

	if !VerifySignature(body, signature, secret) {
		t.Fatal("expected signature with sha256 prefix to be valid")
	}
}

func TestVerifySignatureRejectsInvalidSignature(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"event":"message.received"}`)

	if VerifySignature(body, "invalid-signature", secret) {
		t.Fatal("expected invalid signature to be rejected")
	}
}

func signatureForBody(body []byte, secret string) string {
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write(body)
	return hex.EncodeToString(hash.Sum(nil))
}
