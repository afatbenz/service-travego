package handler

import (
	"fmt"
	"log"
	"service-travego/helper"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Fullname string `json:"fullname" validate:"required,min=3,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Phone    string `json:"phone" validate:"required"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest

	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] BodyParser failed - Path: %s, Error: %v", c.Path(), err)
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	user, token, err := h.authService.Register(req.Username, req.Fullname, req.Email, req.Password, req.Phone)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		log.Printf("[ERROR] Register failed - Username: %s, Email: %s, Status: %d, Error: %v", req.Username, req.Email, statusCode, err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	// Create profile data
	profile := map[string]interface{}{
		"email":    user.Email,
		"username": user.Username,
		"user_id":  user.UserID,
		"phone":    user.Phone,
	}

	// Create response data with profile and token
	responseData := map[string]interface{}{
		"profile": profile,
		"token":   token,
	}

	return helper.SuccessResponse(c, fiber.StatusCreated, "Registration successful. Please check your email for OTP.", responseData)
}

type VerifyOTPRequest struct {
	Token string `json:"token" validate:"required"`
	OTP   string `json:"otp" validate:"required"`
}

func (h *AuthHandler) VerifyOTP(c *fiber.Ctx) error {
	var req VerifyOTPRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	// Validate OTP length
	otpLength := helper.GetOTPLength()
	if len(req.OTP) != otpLength {
		return helper.BadRequestResponse(c, fmt.Sprintf("OTP must be %d digits", otpLength))
	}

	if err := h.authService.VerifyOTP(req.Token, req.OTP); err != nil {
		statusCode := service.GetStatusCode(err)
		log.Printf("[ERROR] VerifyOTP failed - Status: %d, Error: %v", statusCode, err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Email verified successfully. Account activated.", nil)
}

type ResendOTPRequest struct {
	Email string `json:"email" validate:"omitempty,email"`
	Token string `json:"token" validate:"omitempty"`
}

func (h *AuthHandler) ResendOTP(c *fiber.Ctx) error {
	var req ResendOTPRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	// Validate: either email or token must be provided
	if req.Email == "" && req.Token == "" {
		return helper.BadRequestResponse(c, "Either email or token is required")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	token, err := h.authService.ResendOTP(req.Email, req.Token)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		log.Printf("[ERROR] ResendOTP failed - Email: %s, Status: %d, Error: %v", req.Email, statusCode, err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	// Create response data with token
	responseData := map[string]interface{}{
		"token": token,
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "OTP has been resent to your email.", responseData)
}
