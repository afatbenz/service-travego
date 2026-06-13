package waai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendMessageAcceptsPlainTextSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("Created"))
	}))
	defer server.Close()

	client := NewWagyClient("device-1", "token-1")
	client.baseURL = server.URL

	messageID, err := client.SendMessage("628123456789", "halo")
	if err != nil {
		t.Fatalf("expected plain text success response to be accepted, got error: %v", err)
	}
	if messageID != 0 {
		t.Fatalf("expected message ID 0 for non-JSON success response, got %d", messageID)
	}
}

func TestSendMessageParsesJSONSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"message_id":12345,"timestamp":"2026-06-12T21:00:00Z"}}`))
	}))
	defer server.Close()

	client := NewWagyClient("device-1", "token-1")
	client.baseURL = server.URL

	messageID, err := client.SendMessage("628123456789", "halo")
	if err != nil {
		t.Fatalf("expected JSON success response to be parsed, got error: %v", err)
	}
	if messageID != 12345 {
		t.Fatalf("expected message ID 12345, got %d", messageID)
	}
}

func TestSendMessageReturnsHelpfulErrorForNonJSONFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "Cannot send message")
	}))
	defer server.Close()

	client := NewWagyClient("device-1", "token-1")
	client.baseURL = server.URL

	_, err := client.SendMessage("628123456789", "halo")
	if err == nil {
		t.Fatal("expected error for non-JSON failure response")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected error to include status code, got: %v", err)
	}
}
