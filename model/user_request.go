package model

// CreateUserRequest represents create user request payload
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required"`
	Phone    string `json:"phone" validate:"omitempty"`
	Address  string `json:"address" validate:"omitempty"`
}

// UpdateUserRequest represents update user request payload
type UpdateUserRequest struct {
	Name     string `json:"name" validate:"omitempty"`
	Phone    string `json:"phone" validate:"omitempty"`
	Address  string `json:"address" validate:"omitempty"`
	City     string `json:"city" validate:"omitempty"`
	Province string `json:"province" validate:"omitempty"`
}

