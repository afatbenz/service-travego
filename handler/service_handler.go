package handler

import (
	"fmt"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type ServiceHandler struct {
	service         *service.FleetService
	tourService     *service.TourPackageService
	customerService *service.CustomersService
}

func NewServiceHandler(s *service.FleetService, ts *service.TourPackageService, cs *service.CustomersService) *ServiceHandler {
	return &ServiceHandler{
		service:         s,
		tourService:     ts,
		customerService: cs,
	}
}

func (h *ServiceHandler) GetServiceFleets(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 10)

	items, err := h.service.GetServiceFleets(page, perPage)
	if err != nil {
		fmt.Println("Error fetching service fleets:", err)
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Service fleets retrieved", items)
}

func (h *ServiceHandler) GetServiceFleetDetail(c *fiber.Ctx) error {
	fmt.Println("GetServiceFleetDetail")
	var req model.ServiceFleetDetailRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}
	if req.FleetID == "" {
		return helper.BadRequestResponse(c, "fleet_id is required")
	}

	res, err := h.service.GetServiceFleetDetail(req.FleetID)
	if err != nil {
		fmt.Println("Error fetching service fleet detail:", err)
		code := fiber.StatusInternalServerError
		if err.Error() == "fleet not found" {
			code = fiber.StatusNotFound
		}
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet detail retrieved", res)
}

func (h *ServiceHandler) GetServiceFleetAddons(c *fiber.Ctx) error {
	fleetID := c.Params("fleetid")
	if fleetID == "" {
		return helper.BadRequestResponse(c, "fleet_id is required")
	}
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "Invalid or missing organization_id")
	}

	items, err := h.service.GetServiceFleetAddons(orgID, fleetID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet addons retrieved", items)
}

func (h *ServiceHandler) GetServiceFleetAvailibility(c *fiber.Ctx) error {
	var req model.ServiceFleetAvailibilityRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}

	startStr := strings.TrimSpace(req.StartDate)
	endStr := strings.TrimSpace(req.EndDate)

	if startStr == "" {
		return helper.BadRequestResponse(c, "start_date is required")
	}
	if endStr == "" {
		return helper.BadRequestResponse(c, "end_date is required")
	}

	layout := "2006-01-02 15:04"
	startDate, err := time.ParseInLocation(layout, startStr, time.Local)
	if err != nil {
		return helper.BadRequestResponse(c, "invalid start_date")
	}
	endDate, err := time.ParseInLocation(layout, endStr, time.Local)
	if err != nil {
		return helper.BadRequestResponse(c, "invalid end_date")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "Invalid or missing organization_id")
	}

	available, fleets, err := h.service.GetFleetAvailibility(orgID, startDate, endDate, req.FleetID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "OK", fiber.Map{
		"available": available,
		"fleets":    fleets,
	})
}

func (h *ServiceHandler) GetAvailableCities(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "Invalid or missing organization_id")
	}

	cities, err := h.service.GetAvailableCities(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Available cities retrieved", cities)
}

func (h *ServiceHandler) GetPublicTourPackages(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "Invalid or missing organization_id")
	}

	items, err := h.tourService.GetPublicTourPackages(c.Context(), orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Tour packages retrieved", items)
}

func (h *ServiceHandler) CheckCustomerAvailibility(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "Invalid or missing organization_id")
	}

	data, err := h.customerService.CheckCustomerAvailibility(orgID, req.Email, req.Phone)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	if data == nil {
		return helper.SuccessResponse(c, fiber.StatusOK, "Customer not found", "")
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Customer found", data)
}
