package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required"`
	Phone    string `json:"phone" validate:"omitempty"`
	Address  string `json:"address" validate:"omitempty"`
}

type UpdateUserRequest struct {
	Name     string `json:"name" validate:"omitempty"`
	Phone    string `json:"phone" validate:"omitempty"`
	Address  string `json:"address" validate:"omitempty"`
	City     string `json:"city" validate:"omitempty"`
	Province string `json:"province" validate:"omitempty"`
}

func (h *UserHandler) GetAllUsers(c *fiber.Ctx) error {
	users, err := h.userService.GetAllUsers()
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, "Failed to fetch users")
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Users fetched successfully", users)
}

func (h *UserHandler) GetUserByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.BadRequestResponse(c, "User ID is required")
	}

	if _, err := uuid.Parse(id); err != nil {
		return helper.BadRequestResponse(c, "Invalid user ID format")
	}

	user, err := h.userService.GetUserByID(id)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "User fetched successfully", user)
}

func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	var req CreateUserRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Phone:    req.Phone,
		Address:  req.Address,
	}

	createdUser, err := h.userService.CreateUser(user)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusCreated, "User created successfully", createdUser)
}

func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.BadRequestResponse(c, "User ID is required")
	}

	if _, err := uuid.Parse(id); err != nil {
		return helper.BadRequestResponse(c, "Invalid user ID format")
	}

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	user := &model.User{
		Name:     req.Name,
		Phone:    req.Phone,
		Address:  req.Address,
		City:     req.City,
		Province: req.Province,
	}

	updatedUser, err := h.userService.UpdateUser(id, user)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "User updated successfully", updatedUser)
}

func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.BadRequestResponse(c, "User ID is required")
	}

	if _, err := uuid.Parse(id); err != nil {
		return helper.BadRequestResponse(c, "Invalid user ID format")
	}

	if err := h.userService.DeleteUser(id); err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "User deleted successfully", nil)
}
