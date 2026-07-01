package handler

import (
	"service-travego/helper"
	"service-travego/service"
	"time"

	"github.com/gofiber/fiber/v2"
)

type DashboardHandler struct {
	service *service.DashboardService
}

func NewDashboardHandler(service *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		service: service,
	}
}

func (h *DashboardHandler) GetPartnerSummary(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Missing organization context")
	}

	summary, err := h.service.GetPartnerSummary(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Dashboard summary retrieved successfully", summary)
}

func (h *DashboardHandler) GetDashboard(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Missing organization context")
	}

	res, err := h.service.GetDashboard(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Dashboard retrieved successfully", res)
}

func (h *DashboardHandler) GetFinance(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Missing organization context")
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	if startDateStr == "" || endDateStr == "" {
		return helper.BadRequestResponse(c, "start_date and end_date are required")
	}

	startDate, err := time.ParseInLocation("2006-01-02", startDateStr, time.Local)
	if err != nil {
		return helper.BadRequestResponse(c, "Invalid start_date format")
	}
	endDate, err := time.ParseInLocation("2006-01-02", endDateStr, time.Local)
	if err != nil {
		return helper.BadRequestResponse(c, "Invalid end_date format")
	}

	if startDate.After(endDate) {
		return helper.BadRequestResponse(c, "start_date must be less than or equal to end_date")
	}

	diffDays := int(endDate.Sub(startDate).Hours() / 24)
	if diffDays > 365*2 {
		return helper.BadRequestResponse(c, "Date range must not exceed 2 years")
	}

	res, err := h.service.GetFinance(orgID, startDate, endDate)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Dashboard finance retrieved successfully", res)
}

func (h *DashboardHandler) GetTopDestinations(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Missing organization context")
	}

	res, err := h.service.GetTopDestinations(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Top destinations retrieved successfully", res)
}

func (h *DashboardHandler) GetTopPickupCity(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Missing organization context")
	}

	res, err := h.service.GetTopPickupCity(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Top pickup cities retrieved successfully", res)
}

func (h *DashboardHandler) GetTopFleets(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Missing organization context")
	}

	res, err := h.service.GetTopFleets(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Top fleets retrieved successfully", res)
}

func (h *DashboardHandler) GetTopTourPackages(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Missing organization context")
	}

	res, err := h.service.GetTopTourPackages(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Top tour packages retrieved successfully", res)
}

func (h *DashboardHandler) GetTopDrivers(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Missing organization context")
	}

	res, err := h.service.GetTopDrivers(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Top drivers retrieved successfully", res)
}

func (h *DashboardHandler) GetTopCustomers(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Missing organization context")
	}

	res, err := h.service.GetTopCustomers(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Top customers retrieved successfully", res)
}
