package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strings"

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
	c.Set("Content-Disposition", "inline; filename=fleet-order-"+req.OrderID+".pdf")
	return c.Send(pdf)
}

func (h *PrintManagementHandler) GenerateFleetInvoiceDocument(c *fiber.Ctx) error {
	var req model.PrintFleetInvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	pdf, err := h.service.GenerateFleetInvoicePDF(orgID, req.OrderID, req.InvoiceNumber)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "inline; filename=fleet-invoice-"+req.OrderID+".pdf")
	return c.Send(pdf)
}

func (h *PrintManagementHandler) GenerateFleetTripsDocument(c *fiber.Ctx) error {
	scheduleNumber := strings.TrimSpace(c.Params("schedule_number"))
	if scheduleNumber == "" {
		return helper.BadRequestResponse(c, "schedule_number is required")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	pdf, err := h.service.GenerateFleetTripsPDF(orgID, scheduleNumber)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "inline; filename=fleet-trips-"+scheduleNumber+".pdf")
	return c.Send(pdf)
}
