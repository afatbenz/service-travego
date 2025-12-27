package handler

import (
	"fmt"
	"service-travego/helper"
	"service-travego/model"
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
		fmt.Println("Error fetching service fleets:", err)
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Service fleets retrieved", items)
}

func (h *ServiceHandler) GetServiceFleetDetail(c *fiber.Ctx) error {
	var req model.ServiceFleetDetailRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}
	if req.FleetID == "" {
		return helper.BadRequestResponse(c, "fleet_id is required")
	}

	res, err := h.service.GetServiceFleetDetail(req.FleetID)
	if err != nil {
		code := fiber.StatusInternalServerError
		if err.Error() == "fleet not found" {
			code = fiber.StatusNotFound
		}
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet detail retrieved", res)
}
