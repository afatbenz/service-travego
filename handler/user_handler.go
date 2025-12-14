package handler

import (
    "service-travego/helper"
    "service-travego/model"
    "service-travego/service"
    "strconv"

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
	var req model.CreateUserRequest

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

	var req model.UpdateUserRequest
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
        City:     strconv.Itoa(req.City),
        Province: strconv.Itoa(req.Province),
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
