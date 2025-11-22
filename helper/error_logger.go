package helper

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	logFilePath = "assets/log.json"
	maxLogLines = 1000
)

type ErrorLogEntry struct {
	Timestamp    string      `json:"timestamp"`
	ErrorCode    int         `json:"errorCode"`
	TMS          string      `json:"tms"`
	Host         string      `json:"host"`
	Method       string      `json:"method"`
	PathURL      string      `json:"pathURL"`
	Payload      interface{} `json:"payload"`
	Query        interface{} `json:"query"`
	Response     interface{} `json:"response"`
	ErrorMessage string      `json:"errorMessage"`
}

var (
	logMutex sync.Mutex
	logFile  *os.File
	logCount int
)

func init() {
	ensureLogFile()
	countLogLines()
}

func ensureLogFile() {
	if _, err := os.Stat("assets"); os.IsNotExist(err) {
		os.MkdirAll("assets", 0755)
	}
}

func countLogLines() {
	file, err := os.Open(logFilePath)
	if err != nil {
		logCount = 0
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	logCount = 0
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			logCount++
		}
	}
}

func resetLogFile() error {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}

	file, err := os.Create(logFilePath)
	if err != nil {
		return err
	}
	logFile = file
	logCount = 0
	return nil
}

// sanitizeCredentials replaces sensitive fields with "****"
func sanitizeCredentials(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	switch v := data.(type) {
	case map[string]interface{}:
		sanitized := make(map[string]interface{})
		for key, value := range v {
			lowerKey := strings.ToLower(key)
			// Check if key contains sensitive keywords
			if strings.Contains(lowerKey, "password") ||
				strings.Contains(lowerKey, "credential") ||
				strings.Contains(lowerKey, "secret") ||
				strings.Contains(lowerKey, "token") ||
				strings.Contains(lowerKey, "otp") ||
				strings.Contains(lowerKey, "api_key") ||
				strings.Contains(lowerKey, "apikey") ||
				strings.Contains(lowerKey, "access_token") ||
				strings.Contains(lowerKey, "refresh_token") ||
				strings.Contains(lowerKey, "auth_token") ||
				lowerKey == "pwd" {
				sanitized[key] = "****"
			} else {
				// Recursively sanitize nested structures
				sanitized[key] = sanitizeCredentials(value)
			}
		}
		return sanitized
	case map[string]string:
		sanitized := make(map[string]string)
		for key, value := range v {
			lowerKey := strings.ToLower(key)
			// Check if key contains sensitive keywords
			if strings.Contains(lowerKey, "password") ||
				strings.Contains(lowerKey, "credential") ||
				strings.Contains(lowerKey, "secret") ||
				strings.Contains(lowerKey, "token") ||
				strings.Contains(lowerKey, "otp") ||
				strings.Contains(lowerKey, "api_key") ||
				strings.Contains(lowerKey, "apikey") ||
				strings.Contains(lowerKey, "access_token") ||
				strings.Contains(lowerKey, "refresh_token") ||
				strings.Contains(lowerKey, "auth_token") ||
				lowerKey == "pwd" {
				sanitized[key] = "****"
			} else {
				sanitized[key] = value
			}
		}
		return sanitized
	case []interface{}:
		sanitized := make([]interface{}, len(v))
		for i, item := range v {
			sanitized[i] = sanitizeCredentials(item)
		}
		return sanitized
	default:
		return v
	}
}

func LogErrorToFile(c *fiber.Ctx, errorCode int, errorMessage string, response interface{}) error {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logCount >= maxLogLines {
		if err := resetLogFile(); err != nil {
			return fmt.Errorf("failed to reset log file: %w", err)
		}
	}

	if logFile == nil {
		var err error
		logFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
	}

	txID := GetTransactionID(c)
	host := c.Hostname()
	if host == "" {
		host = c.IP()
	}

	var payload interface{}
	if c.Method() == "POST" || c.Method() == "PUT" || c.Method() == "PATCH" {
		if bodyData, ok := c.Locals("request_body").([]byte); ok && len(bodyData) > 0 {
			var jsonBody interface{}
			if err := json.Unmarshal(bodyData, &jsonBody); err == nil {
				payload = jsonBody
			} else {
				payload = string(bodyData)
			}
		} else {
			body := c.Body()
			if len(body) > 0 {
				var jsonBody interface{}
				if err := json.Unmarshal(body, &jsonBody); err == nil {
					payload = jsonBody
				} else {
					payload = string(body)
				}
			}
		}
	}

	queryParams := make(map[string]string)
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		queryParams[string(key)] = string(value)
	})
	if len(queryParams) == 0 {
		queryParams = nil
	}

	// Sanitize sensitive data before logging
	sanitizedPayload := sanitizeCredentials(payload)
	sanitizedQuery := sanitizeCredentials(queryParams)
	sanitizedResponse := sanitizeCredentials(response)

	entry := ErrorLogEntry{
		Timestamp:    time.Now().Format(time.RFC3339),
		ErrorCode:    errorCode,
		TMS:          txID,
		Host:         host,
		Method:       c.Method(),
		PathURL:      c.Path(),
		Payload:      sanitizedPayload,
		Query:        sanitizedQuery,
		Response:     sanitizedResponse,
		ErrorMessage: errorMessage,
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	if _, err := logFile.WriteString(string(jsonData) + "\n"); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	if err := logFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync log file: %w", err)
	}

	logCount++
	return nil
}
