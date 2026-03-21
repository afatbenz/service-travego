package handler

import (
	"service-travego/helper"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type CustomersHandler struct {
	service *service.CustomersService
}

func NewCustomersHandler(s *service.CustomersService) *CustomersHandler {
	return &CustomersHandler{service: s}
}

func (h *CustomersHandler) ListCustomers(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	customerName := c.Query("customer_name")

	items, err := h.service.ListCustomers(orgID, customerName)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Customers loaded", items)
}

