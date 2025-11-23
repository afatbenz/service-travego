package helper

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// sanitizeErrorMessage converts database errors to user-friendly messages
func sanitizeErrorMessage(err error) string {
	errStr := err.Error()

	// Check for PostgreSQL errors
	if strings.Contains(errStr, "pq:") {
		// Return user-friendly message but keep original in log
		return "Database error occurred"
	}

	// Check for MySQL errors
	if strings.Contains(errStr, "Error") && strings.Contains(errStr, ":") {
		return "Database error occurred"
	}

	// Check for SQL errors
	if strings.Contains(strings.ToLower(errStr), "sql:") ||
		strings.Contains(strings.ToLower(errStr), "database") ||
		strings.Contains(strings.ToLower(errStr), "column") ||
		strings.Contains(strings.ToLower(errStr), "relation") ||
		strings.Contains(strings.ToLower(errStr), "table") {
		return "Database error occurred"
	}

	// For other errors, return as is (might be user-friendly already)
	return errStr
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	originalError := err.Error()
	userMessage := originalError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		userMessage = e.Message
		originalError = e.Error()
	} else {
		// Sanitize database errors for user response
		userMessage = sanitizeErrorMessage(err)
	}

	txID := GetTransactionID(c)
	// Use original error for logging
	errorLogMessage := fmt.Sprintf("TransactionID: %s - %s %s - Status: %d - Error: %v", txID, c.Method(), c.Path(), code, err)
	log.Printf("[ERROR] %s", errorLogMessage)

	env := os.Getenv("APP_ENV")
	if env == "" || env == "development" || env == "dev" || env == "local" {
		log.Printf("[ERROR] TransactionID: %s - Stack trace: %s", txID, string(debug.Stack()))
	}

	// Response with user-friendly message
	response := ErrorResponse{
		Status:        "error",
		Message:       userMessage,
		Data:          nil,
		TransactionID: txID,
	}

	// Log original error detail to file
	if err := LogErrorToFile(c, code, errorLogMessage, response); err != nil {
		log.Printf("[ERROR] Failed to write error log to file: %v", err)
	}

	return c.Status(code).JSON(response)
}
