package helper

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom validators if needed
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := fld.Tag.Get("json")
		if name == "-" {
			return ""
		}
		return name
	})
}

// ValidateStruct validates a struct based on its tags
func ValidateStruct(s interface{}) []ValidationError {
	var errors []ValidationError

	err := validate.Struct(s)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ValidationError
			element.FailedField = err.Field()
			element.Tag = err.Tag()
			element.Value = err.Value()
			element.Error = getErrorMessage(err)
			errors = append(errors, element)
		}
	}

	return errors
}

// ValidationError represents a validation error
type ValidationError struct {
	FailedField string
	Tag         string
	Value       interface{}
	Error       string
}

// getErrorMessage returns a user-friendly error message
func getErrorMessage(err validator.FieldError) string {
	// Get field name from JSON tag if available, otherwise use struct field name
	// Since we registered TagNameFunc to use JSON tag, err.Field() should already return JSON tag name
	fieldName := err.Field()

	// If field name is empty or contains comma, extract the first part
	if idx := strings.Index(fieldName, ","); idx != -1 {
		fieldName = fieldName[:idx]
	}

	// Convert to lowercase for consistency (e.g., "Phone" -> "phone")
	fieldName = strings.ToLower(fieldName)

	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fieldName)
	case "email":
		return fmt.Sprintf("%s must be a valid email", fieldName)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fieldName, err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", fieldName, err.Param())
	case "alphanum":
		return fmt.Sprintf("%s must contain only alphanumeric characters", fieldName)
	case "numeric":
		return fmt.Sprintf("%s must be numeric", fieldName)
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", fieldName, err.Param())
	default:
		return fmt.Sprintf("%s is invalid", fieldName)
	}
}
