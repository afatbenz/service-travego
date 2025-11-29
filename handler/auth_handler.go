package handler

import (
    "fmt"
    "log"
    "os"
    "service-travego/helper"
    "service-travego/model"
    "service-travego/service"
    "strconv"
    "strings"

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

func (h *AuthHandler) Register(c *fiber.Ctx) error {
    var req model.RegisterRequest

	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] BodyParser failed - Path: %s, Error: %v", c.Path(), err)
		return helper.BadRequestResponse(c, "Invalid request body")
	}

    if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
        return helper.SendValidationErrorResponse(c, validationErrors)
    }

    // Default username from email local-part if not provided
    if req.Username == "" && req.Email != "" {
        if at := strings.Index(req.Email, "@"); at > 0 {
            req.Username = req.Email[:at]
        } else {
            req.Username = req.Email
        }
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

func (h *AuthHandler) VerifyOTP(c *fiber.Ctx) error {
	var req model.VerifyOTPRequest

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

func (h *AuthHandler) ResendOTP(c *fiber.Ctx) error {
	var req model.ResendOTPRequest

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

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest

	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] BodyParser failed - Path: %s, Error: %v", c.Path(), err)
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	// Validate: either email or phone must be provided
	if req.Email == "" && req.Phone == "" {
		return helper.BadRequestResponse(c, "email or phone is required")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	loginResponse, err := h.authService.Login(req.Email, req.Phone, req.Password)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		log.Printf("[ERROR] Login failed - Email: %s, Phone: %s, Status: %d, Error: %v", req.Email, req.Phone, statusCode, err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	// Store token in locals for middleware access
	c.Locals("auth_token", loginResponse.Token)

	// Create response data with token, username, fullname, and avatar
	responseData := map[string]interface{}{
		"token":    loginResponse.Token,
		"username": loginResponse.Username,
		"fullname": loginResponse.Fullname,
		"avatar":   loginResponse.Avatar,
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Login successful.", responseData)
}

func (h *AuthHandler) RequestResetPassword(c *fiber.Ctx) error {
	var req model.RequestResetPasswordRequest

	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] BodyParser failed - Path: %s, Error: %v", c.Path(), err)
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	// Get reset password URL from environment or use default
	resetPasswordURL := os.Getenv("RESET_PASSWORD_URL")
	if resetPasswordURL == "" {
		resetPasswordURL = "http://localhost:3000/reset-password" // Default URL
	}

	// Get expiry minutes from environment or use default (60 minutes)
	expiryMinutes := 60 // Default 60 minutes
	if envExpiry := os.Getenv("RESET_PASSWORD_EXPIRY"); envExpiry != "" {
		if expiry, err := strconv.Atoi(envExpiry); err == nil && expiry > 0 {
			expiryMinutes = expiry
		}
	}

	if err := h.authService.RequestResetPassword(req.Email, resetPasswordURL, expiryMinutes); err != nil {
		statusCode := service.GetStatusCode(err)
		log.Printf("[ERROR] RequestResetPassword failed - Email: %s, Status: %d, Error: %v", req.Email, statusCode, err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	// Create response data with email
	responseData := map[string]interface{}{
		"email": req.Email,
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Password reset link has been sent to your email.", responseData)
}

func (h *AuthHandler) UpdatePassword(c *fiber.Ctx) error {
	var req model.UpdatePasswordRequest

	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] BodyParser failed - Path: %s, Error: %v", c.Path(), err)
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	if err := h.authService.UpdatePassword(req.Token, req.NewPassword, req.ConfirmPassword); err != nil {
		statusCode := service.GetStatusCode(err)
		log.Printf("[ERROR] UpdatePassword failed - Status: %d, Error: %v", statusCode, err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Password updated successfully.", nil)
}
