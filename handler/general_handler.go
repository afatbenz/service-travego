package handler

import (
	"service-travego/helper"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type GeneralHandler struct {
	generalService   *service.GeneralService
	fleetTypeService *service.FleetTypeService
	fleetMetaService *service.FleetMetaService
}

func NewGeneralHandler(generalService *service.GeneralService) *GeneralHandler {
	return &GeneralHandler{
		generalService: generalService,
	}
}

func (h *GeneralHandler) SetFleetTypeService(s *service.FleetTypeService) {
	h.fleetTypeService = s
}

func (h *GeneralHandler) SetFleetMetaService(s *service.FleetMetaService) {
	h.fleetMetaService = s
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
	searchText := c.Query("search", "")
	provinces, err := h.generalService.GetProvinces(searchText)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load provinces")
	}
	message := "Provinces loaded successfully"
	if searchText != "" {
		message = "Provinces filtered by search text loaded successfully"
	}
	return helper.SuccessResponse(c, fiber.StatusOK, message, provinces)
}

func (h *GeneralHandler) GetFleetTypes(c *fiber.Ctx) error {
	if h.fleetTypeService == nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Fleet type service not configured")
	}
	types, err := h.fleetTypeService.GetAllFleetTypes()
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load fleet types")
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet types loaded successfully", types)
}

func (h *GeneralHandler) GetFleetBodies(c *fiber.Ctx) error {
	if h.fleetMetaService == nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Fleet meta service not configured")
	}
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "missing organization context")
	}
	search := c.Query("search", "")
	list, err := h.fleetMetaService.GetBodies(orgID, search)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load fleet bodies")
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet bodies loaded successfully", list)
}

func (h *GeneralHandler) GetFleetEngines(c *fiber.Ctx) error {
	if h.fleetMetaService == nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Fleet meta service not configured")
	}
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "missing organization context")
	}
	search := c.Query("search", "")
	list, err := h.fleetMetaService.GetEngines(orgID, search)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load fleet engines")
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet engines loaded successfully", list)
}

func (h *GeneralHandler) GetCities(c *fiber.Ctx) error {
	// Get query parameters (optional)
	provinceID := c.Query("province_id", "")
	provinceName := c.Query("province", "")
	searchText := c.Query("search", "")

	// Support passing province as ID via `province` when `province_id` is empty
	if provinceID == "" && provinceName != "" {
		onlyDigits := true
		for i := 0; i < len(provinceName); i++ {
			ch := provinceName[i]
			if ch < '0' || ch > '9' {
				onlyDigits = false
				break
			}
		}
		if onlyDigits {
			provinceID = provinceName
			provinceName = ""
		}
	}

	cities, err := h.generalService.GetCities(provinceID, provinceName, searchText)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load cities")
	}

	message := "Cities loaded successfully"
	if (provinceID != "" || provinceName != "") && searchText != "" {
		message = "Cities filtered by province and search text loaded successfully"
	} else if provinceID != "" || provinceName != "" {
		message = "Cities filtered by province loaded successfully"
	} else if searchText != "" {
		message = "Cities filtered by search text loaded successfully"
	}

	return helper.SuccessResponse(c, fiber.StatusOK, message, cities)
}
