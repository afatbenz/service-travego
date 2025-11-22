package handler

import (
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
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
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

	user, err := h.authService.Register(req.Username, req.Email, req.Password)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		log.Printf("[ERROR] Register failed - Username: %s, Email: %s, Status: %d, Error: %v", req.Username, req.Email, statusCode, err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusCreated, "Registration successful. Please check your email for OTP.", user)
}

type VerifyOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=8"`
}

func (h *AuthHandler) VerifyOTP(c *fiber.Ctx) error {
	var req VerifyOTPRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	if err := h.authService.VerifyOTP(req.Email, req.OTP); err != nil {
		statusCode := service.GetStatusCode(err)
		log.Printf("[ERROR] VerifyOTP failed - Email: %s, Status: %d, Error: %v", req.Email, statusCode, err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Email verified successfully. Account activated.", nil)
}
