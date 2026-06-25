package wagy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"service-travego/helper"
	"strings"
)

// SendMessageRequest represents the request body for sending a message
type SendMessageRequest struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

// SendDocumentRequest represents the request body for sending a document
type SendDocumentRequest struct {
	Phone     string `json:"phone"`
	Document  string `json:"document,omitempty"`  // base64 encoded file
	Filename  string `json:"filename,omitempty"`  // filename for document
	Caption   string `json:"caption,omitempty"`   // optional caption
	MediaType string `json:"media_type"`          // "document" or "image"
	MediaURL  string `json:"media_url,omitempty"` // URL to the media file
}

// SendImageRequest represents the request body for sending an image (optional, alias for SendDocumentRequest)
type SendImageRequest = SendDocumentRequest

// SendMessageResponse represents the response from Wagy
type SendMessageResponse struct {
	Status string `json:"status"`
	Data   struct {
		MessageID int64  `json:"message_id"`
		Timestamp string `json:"timestamp"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

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

// SendDocument sends a document (PDF, etc.) via Wagy API using base64
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

// SendDocumentWithURL sends a document via Wagy API using a public URL
func (wc *WagyClient) SendDocumentWithURL(phone, filename, mediaURL, caption string) (int64, error) {
	url := fmt.Sprintf("%s/%s/send", wc.baseURL, wc.deviceID)
	log.Printf("[Wagy][SendDocumentWithURL called for %s", filename)

	payload := SendDocumentRequest{
		Phone:     phone,
		Filename:  filename,
		Caption:   caption,
		MediaType: "document",
		MediaURL:  mediaURL,
	}

	log.Printf("[Wagy] payload: %+v", payload)

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

// SendImage sends an image via Wagy API using base64
func (wc *WagyClient) SendImage(phone, filename string, fileData []byte, caption string) (int64, error) {
	url := fmt.Sprintf("%s/%s/send", wc.baseURL, wc.deviceID)

	// Encode file to base64
	base64Data := base64.StdEncoding.EncodeToString(fileData)

	payload := SendDocumentRequest{
		Phone:     phone,
		Document:  base64Data,
		Filename:  filename,
		Caption:   caption,
		MediaType: "image",
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

// SendImageWithURL sends an image via Wagy API using a public URL
func (wc *WagyClient) SendImageWithURL(phone, mediaURL, caption string) (int64, error) {
	url := fmt.Sprintf("%s/%s/send", wc.baseURL, wc.deviceID)

	payload := SendDocumentRequest{
		Phone:     phone,
		Caption:   caption,
		MediaType: "image",
		MediaURL:  mediaURL,
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

// SendMediaWithTempFile saves file to temporary directory, sends via Wagy using URL, then deletes file
func (wc *WagyClient) SendMediaWithTempFile(phone, filename string, fileData []byte, mediaType, caption string) (int64, error) {
	// Create temporary directory if it doesn't exist
	tempDir := filepath.Join("assets", "tmp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Save file to temp directory
	tempFilePath := filepath.Join(tempDir, filename)
	if err := os.WriteFile(tempFilePath, fileData, 0644); err != nil {
		return 0, fmt.Errorf("failed to write temp file: %w", err)
	}

	// Cleanup temp file after we're done
	defer func() {
		_ = os.Remove(tempFilePath)
	}()

	// Generate public URL
	assetPath := "/assets/tmp/" + filename
	mediaURL := helper.GetAssetURL(assetPath)

	// Send via Wagy
	if mediaType == "document" {
		return wc.SendDocumentWithURL(phone, filename, mediaURL, caption)
	} else if mediaType == "image" {
		return wc.SendImageWithURL(phone, mediaURL, caption)
	}

	return 0, fmt.Errorf("unsupported media type: %s", mediaType)
}

func truncateForError(body string) string {
	const maxLen = 200
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "..."
}
