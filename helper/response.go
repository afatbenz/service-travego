package helper

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type Response struct {
	Success       bool        `json:"success"`
	Message       string      `json:"message"`
	TransactionID string      `json:"transactionid"`
	Data          interface{} `json:"data,omitempty"`
	Errors        interface{} `json:"errors,omitempty"`
}

type ErrorResponse struct {
	Success       bool        `json:"success"`
	Message       string      `json:"message"`
	TransactionID string      `json:"transactionid"`
	Errors        interface{} `json:"errors,omitempty"`
}

type ValidationErrorResponse struct {
	Success       bool                    `json:"success"`
	Message       string                  `json:"message"`
	TransactionID string                  `json:"transactionid"`
	Errors        []ValidationErrorDetail `json:"errors"`
}

type ValidationErrorDetail struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

func SuccessResponse(c *fiber.Ctx, statusCode int, message string, data interface{}) error {
	txID := GetTransactionID(c)
	return c.Status(statusCode).JSON(Response{
		Success:       true,
		Message:       message,
		TransactionID: txID,
		Data:          data,
	})
}

func SendErrorResponse(c *fiber.Ctx, statusCode int, message string) error {
	txID := GetTransactionID(c)
	errorLogMessage := fmt.Sprintf("TransactionID: %s - %s %s - Status: %d - Error: %s", txID, c.Method(), c.Path(), statusCode, message)

	response := ErrorResponse{
		Success:       false,
		Message:       message,
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
	errors := make([]ValidationErrorDetail, 0, len(validationErrors))
	for _, ve := range validationErrors {
		errors = append(errors, ValidationErrorDetail{
			Field:   ve.FailedField,
			Tag:     ve.Tag,
			Message: ve.Error,
		})
	}

	errorLogMessage := fmt.Sprintf("TransactionID: %s - %s %s - Status: %d - Validation Error: %d errors", txID, c.Method(), c.Path(), fiber.StatusBadRequest, len(validationErrors))

	response := ValidationErrorResponse{
		Success:       false,
		Message:       "Validation failed",
		TransactionID: txID,
		Errors:        errors,
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
