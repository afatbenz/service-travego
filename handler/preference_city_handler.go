package handler

import (
	"service-travego/helper"
	"service-travego/service"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type PreferenceCityHandler struct {
	prefService *service.PreferenceCityService
}

func NewPreferenceCityHandler(prefService *service.PreferenceCityService) *PreferenceCityHandler {
	return &PreferenceCityHandler{
		prefService: prefService,
	}
}

func (h *PreferenceCityHandler) GetCities(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization_id")
	}

	cityIDStr := c.Query("city_id", "")

	var cityID *int

	if cityIDStr != "" {
		if id, err := strconv.Atoi(cityIDStr); err == nil {
			cityID = &id
		}
	}

	list, err := h.prefService.GetAll(orgID, cityID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load preference cities: "+err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Preference cities loaded successfully", list)
}

func (h *PreferenceCityHandler) CreateCity(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization_id")
	}

	userID, _ := c.Locals("user_id").(string)

	var req struct {
		CityID       []int `json:"city_id"`
		MinimalDay   int   `json:"minimal_day"`
		ServiceTypes []int `json:"service_types"`
		ServiceType  []int `json:"service_type"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	if len(req.CityID) == 0 {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing or empty city_id array")
	}

	if req.MinimalDay <= 0 {
		req.MinimalDay = 1
	}

	serviceTypes := req.ServiceTypes
	if len(serviceTypes) == 0 {
		serviceTypes = req.ServiceType
	}

	if err := h.prefService.Create(req.CityID, req.MinimalDay, orgID, userID, serviceTypes); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to create preference city: "+err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Preference city created successfully", nil)
}

func (h *PreferenceCityHandler) UpdateCity(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization_id")
	}

	var req struct {
		PreferenceID   string `json:"preference_id"`
		CityID         int    `json:"city_id"`
		MinimalDay     int    `json:"minimal_day"`
		ServiceTypeIDs []int  `json:"service_type_ids"`
		ServiceTypes   []int  `json:"service_types"`
		ServiceType    []int  `json:"service_type"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	if req.PreferenceID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing preference_id")
	}

	if req.CityID == 0 {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing city_id")
	}

	if req.MinimalDay <= 0 {
		req.MinimalDay = 1
	}

	serviceTypeIDs := req.ServiceTypeIDs
	if len(serviceTypeIDs) == 0 {
		serviceTypeIDs = req.ServiceTypes
	}
	if len(serviceTypeIDs) == 0 {
		serviceTypeIDs = req.ServiceType
	}

	if err := h.prefService.Update(req.PreferenceID, req.CityID, req.MinimalDay, orgID, serviceTypeIDs); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to update preference city: "+err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Preference city updated successfully", nil)
}

func (h *PreferenceCityHandler) DeleteCity(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization_id")
	}

	var req struct {
		PreferenceID string `json:"preference_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if err := h.prefService.Delete(req.PreferenceID, orgID); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to delete preference city")
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Preference city deleted successfully", nil)
}

func (h *PreferenceCityHandler) DeleteTypes(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization_id")
	}

	var req struct {
		CityID      int `json:"city_id"`
		ServiceType int `json:"service_type"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if err := h.prefService.DeleteByCityAndServiceType(req.CityID, req.ServiceType, orgID); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to delete preference city types")
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Preference city types deleted successfully", nil)
}
