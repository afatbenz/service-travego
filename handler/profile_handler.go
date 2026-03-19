package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (h *UserHandler) UpdateProfile(c *fiber.Ctx) error {
	// Get user_id from locals (set by JWT middleware)
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}

	if _, err := uuid.Parse(userID); err != nil {
		return helper.BadRequestResponse(c, "Invalid user ID format")
	}

	var req model.UpdateProfileRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	var dob *time.Time
	if req.DateOfBirth != nil {
		dobStr := strings.TrimSpace(*req.DateOfBirth)
		if dobStr != "" {
			// Try YYYY-MM-DD first
			if t, err := time.Parse("2006-01-02", dobStr); err == nil {
				dob = &t
			} else if t2, err2 := time.Parse(time.RFC3339, dobStr); err2 == nil {
				dob = &t2
			}
		}
	}

	user := &model.User{
		UserID:      userID,
		Name:        req.Name,
		Phone:       req.Phone,
		NPWP:        req.NPWP,
		Gender:      req.Gender,
		DateOfBirth: dob,
		Address:     req.Address,
		City:        strconv.Itoa(req.City),
		Province:    strconv.Itoa(req.Province),
		PostalCode:  req.PostalCode,
		Avatar:      req.Avatar,
	}

	updatedUser, err := h.userService.UpdateProfile(user)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Profile updated successfully", updatedUser)
}

// GetProfile handles GET /api/profile/detail
func (h *UserHandler) GetProfile(c *fiber.Ctx) error {
	// Get user_id from locals (set by JWT middleware)
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}

	profile, err := h.userService.GetProfile(userID)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Profile retrieved successfully", profile)
}

func (h *UserHandler) CheckPassword(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}

	var req model.CheckPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	if err := h.userService.CheckPassword(userID, req.Password); err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Password is valid", nil)
}

func (h *UserHandler) ValidateUpdatePassword(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.UnauthorizedResponse(c, "Organization not found")
	}

	if err := h.userService.SendUpdatePasswordOTP(orgID, userID); err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "OTP sent", nil)
}

// UpdatePassword handles POST /api/profile/update-password
func (h *UserHandler) UpdatePassword(c *fiber.Ctx) error {
	// Get user_id from locals (set by JWT middleware)
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}

	if _, err := uuid.Parse(userID); err != nil {
		return helper.BadRequestResponse(c, "Invalid user ID format")
	}

	var req model.UpdatePasswordWithOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	if err := h.userService.UpdatePasswordWithOTP(userID, req.OTP, req.ExistingPassword, req.NewPassword, req.ConfirmPassword); err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Password updated successfully", nil)
}
