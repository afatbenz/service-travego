package helper

// CommonError represents a common error response
type CommonError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error returns the error message
func (e *CommonError) Error() string {
	return e.Message
}

// NewError creates a new error
func NewError(code int, message string) *CommonError {
	return &CommonError{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithDetails creates a new error with details
func NewErrorWithDetails(code int, message, details string) *CommonError {
	return &CommonError{
		Code:    code,
		Message: message,
		Details: details,
	}
}
