package helper

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
)

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := err.Error()

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	txID := GetTransactionID(c)
	errorLogMessage := fmt.Sprintf("TransactionID: %s - %s %s - Status: %d - Error: %v", txID, c.Method(), c.Path(), code, err)
	log.Printf("[ERROR] %s", errorLogMessage)

	env := os.Getenv("APP_ENV")
	if env == "" || env == "development" || env == "dev" || env == "local" {
		log.Printf("[ERROR] TransactionID: %s - Stack trace: %s", txID, string(debug.Stack()))
	}

	response := ErrorResponse{
		Success:       false,
		Message:       message,
		TransactionID: txID,
	}

	if err := LogErrorToFile(c, code, errorLogMessage, response); err != nil {
		log.Printf("[ERROR] Failed to write error log to file: %v", err)
	}

	return c.Status(code).JSON(response)
}
