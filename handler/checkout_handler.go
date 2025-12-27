package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type CheckoutHandler struct {
	service *service.CheckoutService
}

func NewCheckoutHandler(service *service.CheckoutService) *CheckoutHandler {
	return &CheckoutHandler{
		service: service,
	}
}

func (h *CheckoutHandler) GetFleetCheckoutSummary(c *fiber.Ctx) error {
	var req model.CheckoutFleetSummaryRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}

	if req.FleetID == "" || req.PriceID == "" {
		return helper.BadRequestResponse(c, "fleet_id and price_id are required")
	}

	res, err := h.service.GetFleetCheckoutSummary(&req)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Checkout summary retrieved", res)
}
