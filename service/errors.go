package service

import (
	"errors"
	"net/http"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrNotFound           = errors.New("not found")
	ErrEmailExists        = errors.New("email already exists")
	ErrUsernameExists     = errors.New("username already exists")
	ErrPhoneExists        = errors.New("phone already exists")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidOTP         = errors.New("invalid or expired OTP")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInternalServer     = errors.New("internal server error")
)

// ServiceError represents a service error with HTTP status code
type ServiceError struct {
	Err        error
	StatusCode int
	Message    string
}

func (e *ServiceError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

// NewServiceError creates a new service error
func NewServiceError(err error, statusCode int, message string) *ServiceError {
	return &ServiceError{
		Err:        err,
		StatusCode: statusCode,
		Message:    message,
	}
}

// GetStatusCode returns the HTTP status code for an error
func GetStatusCode(err error) int {
	var serviceErr *ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.StatusCode
	}

	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUserNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrEmailExists) || errors.Is(err, ErrUsernameExists) || errors.Is(err, ErrPhoneExists):
		return http.StatusBadRequest
	case errors.Is(err, ErrInvalidOTP) || errors.Is(err, ErrInvalidCredentials):
		return http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
