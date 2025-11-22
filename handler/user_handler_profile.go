package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

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
}

func (h *UserHandler) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Get("user_id")
	if userID == "" {
		return helper.UnauthorizedResponse(c, "User ID is required")
	}

	if _, err := uuid.Parse(userID); err != nil {
		return helper.BadRequestResponse(c, "Invalid user ID format")
	}

	var req UpdateProfileRequest

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
	}

	updatedUser, err := h.userService.UpdateProfile(user)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Profile updated successfully", updatedUser)
}
