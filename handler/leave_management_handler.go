package handler

import (
	"service-travego/helper"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type LeaveManagementHandler struct {
	service *service.LeaveManagementService
}

func NewLeaveManagementHandler(s *service.LeaveManagementService) *LeaveManagementHandler {
	return &LeaveManagementHandler{service: s}
}

func (h *LeaveManagementHandler) GetLeaveTypes(c *fiber.Ctx) error {
	data, err := h.service.GetLeaveTypes()
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Leave types loaded", data)
}

func (h *LeaveManagementHandler) GetLeaveList(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	month := c.Query("month")
	year := c.Query("year")

	data, err := h.service.ListLeaveManagement(orgID, month, year)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Leave management loaded", data)
}
