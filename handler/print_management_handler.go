package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type PrintManagementHandler struct {
	service *service.PrintManagementService
}

func NewPrintManagementHandler(service *service.PrintManagementService) *PrintManagementHandler {
	return &PrintManagementHandler{service: service}
}

func (h *PrintManagementHandler) GenerateOrderFleetDocument(c *fiber.Ctx) error {
	var req model.PrintOrderFleetRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	pdf, err := h.service.GenerateOrderFleetPDF(orgID, req.OrderID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "inline; filename=order-fleet-"+req.OrderID+".pdf")
	return c.Send(pdf)
}

