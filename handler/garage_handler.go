package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type GarageHandler struct {
	garageService *service.GarageService
}

func NewGarageHandler(garageService *service.GarageService) *GarageHandler {
	return &GarageHandler{
		garageService: garageService,
	}
}

func (h *GarageHandler) GetGarages(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization_id")
	}

	itemID := c.Query("item_id", "")

	garages, err := h.garageService.GetGarages(orgID, itemID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load garages")
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Garages loaded successfully", garages)
}

func (h *GarageHandler) CreateGarage(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization_id")
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}

	var req model.CreateGarageRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	garage, err := h.garageService.CreateGarage(orgID, userID, &req)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Garage created successfully", garage)
}

func (h *GarageHandler) UpdateGarage(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization_id")
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}

	var req model.UpdateGarageRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if req.GarageID == "" {
		return helper.BadRequestResponse(c, "garage_id is required")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	garage, err := h.garageService.UpdateGarage(req.GarageID, orgID, userID, &req)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Garage updated successfully", garage)
}

func (h *GarageHandler) DeleteGarage(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization_id")
	}

	var req model.DeleteGarageRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if req.GarageID == "" {
		return helper.BadRequestResponse(c, "garage_id is required")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	if err := h.garageService.DeleteGarage(req.GarageID, orgID); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Garage deleted successfully", nil)
}
