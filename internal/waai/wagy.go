package waai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// WagyClient handles communication with Wagy API
type WagyClient struct {
	deviceID string
	token    string
	baseURL  string
}

// NewWagyClient creates a new Wagy client
func NewWagyClient(deviceID, token string) *WagyClient {
	return &WagyClient{
		deviceID: deviceID,
		token:    token,
		baseURL:  "https://api.wagy.web.id/api/v1",
	}
}

// SendMessage sends a message via Wagy API
func (wc *WagyClient) SendMessage(phone, message string) (int64, error) {
	url := fmt.Sprintf("%s/%s/send", wc.baseURL, wc.deviceID)
	fmt.Println("URL:", url)

	payload := SendMessageRequest{
		Phone:   phone,
		Message: message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+wc.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	trimmedBody := strings.TrimSpace(string(respBody))
	if trimmedBody == "" {
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			return 0, nil
		}
		return 0, fmt.Errorf("wagy returned status %d with empty response body", resp.StatusCode)
	}

	var result SendMessageResponse
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to parse wagy response (status %d): %s", resp.StatusCode, truncateForError(trimmedBody))
	}

	if result.Status != "success" {
		if result.Error != nil {
			return 0, fmt.Errorf("wagy error: %s - %s", result.Error.Code, result.Error.Message)
		}
		return 0, fmt.Errorf("wagy returned non-success status: %s", result.Status)
	}

	return result.Data.MessageID, nil
}

// SendDocument sends a document (PDF, etc.) via Wagy API
func (wc *WagyClient) SendDocument(phone, filename string, fileData []byte, caption string) (int64, error) {
	url := fmt.Sprintf("%s/%s/send", wc.baseURL, wc.deviceID)

	// Encode file to base64
	base64Data := base64.StdEncoding.EncodeToString(fileData)

	payload := SendDocumentRequest{
		Phone:     phone,
		Document:  base64Data,
		Filename:  filename,
		Caption:   caption,
		MediaType: "document",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+wc.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	trimmedBody := strings.TrimSpace(string(respBody))
	if trimmedBody == "" {
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			return 0, nil
		}
		return 0, fmt.Errorf("wagy returned status %d with empty response body", resp.StatusCode)
	}

	var result SendMessageResponse
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to parse wagy response (status %d): %s", resp.StatusCode, truncateForError(trimmedBody))
	}

	if result.Status != "success" {
		if result.Error != nil {
			return 0, fmt.Errorf("wagy error: %s - %s", result.Error.Code, result.Error.Message)
		}
		return 0, fmt.Errorf("wagy returned non-success status: %s", result.Status)
	}

	return result.Data.MessageID, nil
}

func truncateForError(body string) string {
	const maxLen = 200
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "..."
}
