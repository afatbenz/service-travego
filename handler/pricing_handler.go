package handler

import (
	"service-travego/helper"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type PricingHandler struct {
	service *service.PricingService
}

func NewPricingHandler(s *service.PricingService) *PricingHandler {
	return &PricingHandler{service: s}
}

func (h *PricingHandler) GetPackages(c *fiber.Ctx) error {
	packages, err := h.service.GetPackages()
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Packages loaded", packages)
}

func (h *PricingHandler) GetReviews(c *fiber.Ctx) error {
	reviews, err := h.service.GetReviews()
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Reviews loaded", reviews)
}
