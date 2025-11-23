package model

// RegisterRequest represents registration request payload
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Fullname string `json:"fullname" validate:"required,min=3,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Phone    string `json:"phone" validate:"required"`
}

// VerifyOTPRequest represents verify OTP request payload
type VerifyOTPRequest struct {
	Token string `json:"token" validate:"required"`
	OTP   string `json:"otp" validate:"required"`
}

// ResendOTPRequest represents resend OTP request payload
type ResendOTPRequest struct {
	Email string `json:"email" validate:"omitempty,email"`
	Token string `json:"token" validate:"omitempty"`
}

// LoginRequest represents login request payload
type LoginRequest struct {
	Email    string `json:"email" validate:"omitempty,email"`
	Phone    string `json:"phone" validate:"omitempty"`
	Password string `json:"password" validate:"required"`
}

// RequestResetPasswordRequest represents request reset password payload
type RequestResetPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// UpdatePasswordRequest represents update password request payload
type UpdatePasswordRequest struct {
	Token           string `json:"token" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" validate:"required,min=6"`
}

