package handler

import (
	"service-travego/helper"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type GeneralHandler struct {
	generalService *service.GeneralService
}

func NewGeneralHandler(generalService *service.GeneralService) *GeneralHandler {
	return &GeneralHandler{
		generalService: generalService,
	}
}

func (h *GeneralHandler) GetGeneralConfig(c *fiber.Ctx) error {
	config, err := h.generalService.GetGeneralConfig()
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load general configuration")
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "General configuration loaded successfully", config)
}

func (h *GeneralHandler) GetWebMenu(c *fiber.Ctx) error {
	menu, err := h.generalService.GetWebMenu()
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load web menu")
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Web menu loaded successfully", menu)
}

func (h *GeneralHandler) GetProvinces(c *fiber.Ctx) error {
	provinces, err := h.generalService.GetProvinces()
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load provinces")
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Provinces loaded successfully", provinces)
}

func (h *GeneralHandler) GetCities(c *fiber.Ctx) error {
	// Get query parameters (optional)
	provinceID := c.Query("province_id", "")
	searchText := c.Query("search", "")

	cities, err := h.generalService.GetCities(provinceID, searchText)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load cities")
	}

	message := "Cities loaded successfully"
	if provinceID != "" && searchText != "" {
		message = "Cities filtered by province and search text loaded successfully"
	} else if provinceID != "" {
		message = "Cities filtered by province loaded successfully"
	} else if searchText != "" {
		message = "Cities filtered by search text loaded successfully"
	}

	return helper.SuccessResponse(c, fiber.StatusOK, message, cities)
}
