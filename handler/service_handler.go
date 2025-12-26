package handler

import (
	"service-travego/helper"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type ServiceHandler struct {
	service *service.FleetService
}

func NewServiceHandler(s *service.FleetService) *ServiceHandler {
	return &ServiceHandler{service: s}
}

func (h *ServiceHandler) GetServiceFleets(c *fiber.Ctx) error {
	items, err := h.service.GetServiceFleets()
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Service fleets retrieved", items)
}
