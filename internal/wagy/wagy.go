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
	"time"
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
	return wc.SendMessageWithHook(phone, message, nil)
}

// SendMessageWithHook sends a message via Wagy API and reports the result to the provided hook.
func (wc *WagyClient) SendMessageWithHook(phone, message string, onResult func(error)) (int64, error) {
	url := fmt.Sprintf("%s/%s/send", wc.baseURL, wc.deviceID)

	payload := SendMessageRequest{
		Phone:   phone,
		Message: message,
	}

	return wc.sendJSONWithHook(url, payload, onResult)
}

// SendDocument sends a document (PDF, etc.) via Wagy API using base64
func (wc *WagyClient) SendDocument(phone, filename string, fileData []byte, caption string) (int64, error) {
	return wc.SendDocumentWithHook(phone, filename, fileData, caption, nil)
}

// SendDocumentWithHook sends a document and reports the result to the provided hook.
func (wc *WagyClient) SendDocumentWithHook(phone, filename string, fileData []byte, caption string, onResult func(error)) (int64, error) {
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

	return wc.sendJSONWithHook(url, payload, onResult)
}

// SendDocumentWithURL sends a document via Wagy API using a public URL
func (wc *WagyClient) SendDocumentWithURL(phone, filename, mediaURL, caption string) (int64, error) {
	return wc.SendDocumentWithURLAndHook(phone, filename, mediaURL, caption, nil)
}

// SendDocumentWithURLAndHook sends a document via URL and reports the result to the provided hook.
func (wc *WagyClient) SendDocumentWithURLAndHook(phone, filename, mediaURL, caption string, onResult func(error)) (int64, error) {
	url := fmt.Sprintf("%s/%s/send", wc.baseURL, wc.deviceID)
	log.Printf("[Wagy] SendDocumentWithURL filename=%s media_url=%s", filename, mediaURL)

	payload := SendDocumentRequest{
		Phone:     phone,
		Filename:  filename,
		Caption:   caption,
		MediaType: "document",
		MediaURL:  mediaURL,
	}

	log.Printf("[Wagy] payload: %+v", payload)

	return wc.sendJSONWithHook(url, payload, onResult)
}

// SendImage sends an image via Wagy API using base64
func (wc *WagyClient) SendImage(phone, filename string, fileData []byte, caption string) (int64, error) {
	return wc.SendImageWithHook(phone, filename, fileData, caption, nil)
}

// SendImageWithHook sends an image and reports the result to the provided hook.
func (wc *WagyClient) SendImageWithHook(phone, filename string, fileData []byte, caption string, onResult func(error)) (int64, error) {
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

	return wc.sendJSONWithHook(url, payload, onResult)
}

// SendImageWithURL sends an image via Wagy API using a public URL
func (wc *WagyClient) SendImageWithURL(phone, mediaURL, caption string) (int64, error) {
	return wc.SendImageWithURLAndHook(phone, mediaURL, caption, nil)
}

// SendImageWithURLAndHook sends an image via URL and reports the result to the provided hook.
func (wc *WagyClient) SendImageWithURLAndHook(phone, mediaURL, caption string, onResult func(error)) (int64, error) {
	url := fmt.Sprintf("%s/%s/send", wc.baseURL, wc.deviceID)

	payload := SendDocumentRequest{
		Phone:     phone,
		Caption:   caption,
		MediaType: "image",
		MediaURL:  mediaURL,
	}

	return wc.sendJSONWithHook(url, payload, onResult)
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

func (wc *WagyClient) sendJSON(url string, payload interface{}) (int64, error) {
	return wc.sendJSONWithHook(url, payload, nil)
}

func (wc *WagyClient) sendJSONWithHook(url string, payload interface{}, onResult func(error)) (messageID int64, retErr error) {
	defer func() {
		if onResult != nil {
			onResult(retErr)
		}
	}()

	body, err := json.Marshal(payload)
	if err != nil {
		retErr = fmt.Errorf("failed to marshal request: %w", err)
		return 0, retErr
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		retErr = fmt.Errorf("failed to create request: %w", err)
		return 0, retErr
	}

	req.Header.Set("Authorization", "Bearer "+wc.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := wagyHTTPClient().Do(req)
	if err != nil {
		retErr = fmt.Errorf("failed to send request: %w", err)
		return 0, retErr
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		retErr = fmt.Errorf("failed to read response: %w", err)
		return 0, retErr
	}

	messageID, retErr = parseWagySendResponse(resp.StatusCode, respBody)
	return messageID, retErr
}

func parseWagySendResponse(statusCode int, respBody []byte) (int64, error) {
	trimmedBody := strings.TrimSpace(string(respBody))
	if trimmedBody == "" {
		if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
			return 0, nil
		}
		return 0, fmt.Errorf("wagy returned status %d with empty response body", statusCode)
	}

	var result SendMessageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to parse wagy response (status %d): %s", statusCode, truncateForError(trimmedBody))
	}

	if result.Status != "success" {
		if result.Error != nil {
			return 0, fmt.Errorf("wagy error: %s - %s", result.Error.Code, result.Error.Message)
		}
		return 0, fmt.Errorf("wagy returned non-success status: %s", result.Status)
	}

	return result.Data.MessageID, nil
}

func wagyHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}
