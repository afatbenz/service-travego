package handler

import (
	"service-travego/helper"
	"service-travego/model"
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
	orgID, _ := c.Locals("organization_id").(string)
	userID, _ := c.Locals("user_id").(string)

	packages, err := h.service.GetPackages(orgID, userID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Packages loaded", packages)
}

func (h *PricingHandler) GetPackageDetail(c *fiber.Ctx) error {
	packageID := c.Params("package_id")
	orgID, _ := c.Locals("organization_id").(string)
	userID, _ := c.Locals("user_id").(string)

	packageDetail, err := h.service.GetPackageDetail(packageID, orgID, userID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Package detail loaded", packageDetail)
}

func (h *PricingHandler) GetReviews(c *fiber.Ctx) error {
	reviews, err := h.service.GetReviews()
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Reviews loaded", reviews)
}

func (h *PricingHandler) SubmitContact(c *fiber.Ctx) error {
	var contact model.ContactSubmission
	if err := c.BodyParser(&contact); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := h.service.SubmitContact(contact); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Contact submitted", nil)
}
