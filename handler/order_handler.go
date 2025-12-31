package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type OrderHandler struct {
	service *service.OrderService
}

func NewOrderHandler(service *service.OrderService) *OrderHandler {
	return &OrderHandler{
		service: service,
	}
}

func (h *OrderHandler) GetFleetOrderSummary(c *fiber.Ctx) error {
	var req model.OrderFleetSummaryRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}

	if req.FleetID == "" || req.PriceID == "" {
		return helper.BadRequestResponse(c, "fleet_id and price_id are required")
	}

	res, err := h.service.GetFleetOrderSummary(&req)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Order summary retrieved", res)
}
