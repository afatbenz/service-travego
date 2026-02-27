package handler

import (
	"service-travego/helper"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type TourPackageHandler struct {
	service *service.TourPackageService
}

func NewTourPackageHandler(service *service.TourPackageService) *TourPackageHandler {
	return &TourPackageHandler{
		service: service,
	}
}

func (h *TourPackageHandler) GetTourPackages(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	packages, err := h.service.GetTourPackages(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Tour packages retrieved successfully", packages)
}
