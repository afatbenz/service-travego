package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

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

	user := &model.User{
		UserID:      userID,
		Name:        req.Name,
		Phone:       req.Phone,
		NPWP:        req.NPWP,
		Gender:      req.Gender,
		DateOfBirth: req.DateOfBirth,
		Address:     req.Address,
		City:        req.City,
		Province:    req.Province,
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

	var req model.UpdateProfilePasswordRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	// Validate new_password and confirm_password match
	if req.NewPassword != req.ConfirmPassword {
		return helper.BadRequestResponse(c, "new_password and confirm_password must match")
	}

	if err := h.userService.UpdatePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Password updated successfully", nil)
}
