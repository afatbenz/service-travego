package helper

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type Response struct {
	Status        string      `json:"status"`
	Message       string      `json:"message"`
	Data          interface{} `json:"data,omitempty"`
	TransactionID string      `json:"transaction_id"`
}

type ErrorResponse struct {
	Status        string      `json:"status"`
	Message       string      `json:"message"`
	Data          interface{} `json:"data"`
	TransactionID string      `json:"transaction_id"`
}

type ValidationErrorDetail struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

func SuccessResponse(c *fiber.Ctx, statusCode int, message string, data interface{}) error {
	txID := GetTransactionID(c)
	// Default status code to 200 if not specified or 0
	if statusCode == 0 {
		statusCode = fiber.StatusOK
	}
	return c.Status(statusCode).JSON(Response{
		Status:        "success",
		Message:       message,
		Data:          data,
		TransactionID: txID,
	})
}

func SendErrorResponse(c *fiber.Ctx, statusCode int, message string) error {
	txID := GetTransactionID(c)
	errorLogMessage := fmt.Sprintf("TransactionID: %s - %s %s - Status: %d - Error: %s", txID, c.Method(), c.Path(), statusCode, message)

	response := ErrorResponse{
		Status:        "error",
		Message:       message,
		Data:          nil,
		TransactionID: txID,
	}

	if err := LogErrorToFile(c, statusCode, errorLogMessage, response); err != nil {
		// Log to console if file logging fails, but don't fail the request
		fmt.Printf("[WARNING] Failed to write error log to file: %v\n", err)
	}

	return c.Status(statusCode).JSON(response)
}

func SendValidationErrorResponse(c *fiber.Ctx, validationErrors []ValidationError) error {
	txID := GetTransactionID(c)

	// Get first error message
	var firstErrorMessage string
	if len(validationErrors) > 0 {
		firstErrorMessage = validationErrors[0].Error
	} else {
		firstErrorMessage = "Validation failed"
	}

	errorLogMessage := fmt.Sprintf("TransactionID: %s - %s %s - Status: %d - Validation Error: %d errors", txID, c.Method(), c.Path(), fiber.StatusBadRequest, len(validationErrors))

	response := ErrorResponse{
		Status:        "error",
		Message:       firstErrorMessage,
		Data:          nil,
		TransactionID: txID,
	}

	if err := LogErrorToFile(c, fiber.StatusBadRequest, errorLogMessage, response); err != nil {
		fmt.Printf("[WARNING] Failed to write error log to file: %v\n", err)
	}

	return c.Status(fiber.StatusBadRequest).JSON(response)
}

func BadRequestResponse(c *fiber.Ctx, message string) error {
	return SendErrorResponse(c, fiber.StatusBadRequest, message)
}

func UnauthorizedResponse(c *fiber.Ctx, message string) error {
	return SendErrorResponse(c, fiber.StatusUnauthorized, message)
}

func NotFoundResponse(c *fiber.Ctx, message string) error {
	return SendErrorResponse(c, fiber.StatusNotFound, message)
}

func InternalServerErrorResponse(c *fiber.Ctx, message string) error {
	return SendErrorResponse(c, fiber.StatusInternalServerError, message)
}
