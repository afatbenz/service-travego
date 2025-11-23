package model

import "time"

// UpdateProfileRequest represents update profile request payload
type UpdateProfileRequest struct {
	Name        string     `json:"name" validate:"required,min=2,max=100"`
	Phone       string     `json:"phone" validate:"required"`
	NPWP        string     `json:"npwp" validate:"omitempty"`
	Gender      string     `json:"gender" validate:"omitempty,oneof=M F"`
	DateOfBirth *time.Time `json:"date_of_birth" validate:"omitempty"`
	Address     string     `json:"address" validate:"required"`
	City        string     `json:"city" validate:"required"`
	Province    string     `json:"province" validate:"required"`
	PostalCode  string     `json:"postal_code" validate:"required"`
	Avatar      string     `json:"avatar" validate:"omitempty"`
}

// UpdateProfilePasswordRequest represents update password request payload for profile
type UpdateProfilePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" validate:"required,min=6"`
}
