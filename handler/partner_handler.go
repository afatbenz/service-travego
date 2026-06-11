package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type PartnerHandler struct {
	service *service.PartnerService
}

func NewPartnerHandler(s *service.PartnerService) *PartnerHandler {
	return &PartnerHandler{service: s}
}

func (h *PartnerHandler) List(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	partnerName := c.Query("partner_name")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	partners, err := h.service.List(orgID, partnerName, startDate, endDate)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	if partners == nil {
		partners = []model.OperationPartner{}
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Success fetch operation partners", partners)
}

func (h *PartnerHandler) Create(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	userID, okUser := c.Locals("user_id").(string)
	if !ok || !okUser || orgID == "" || userID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}

	var req model.CreateOperationPartnerRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}

	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.SendValidationErrorResponse(c, errs)
	}

	partner, err := h.service.Create(req, orgID, userID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Success create operation partner", partner)
}

func (h *PartnerHandler) Update(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	userID, okUser := c.Locals("user_id").(string)
	if !ok || !okUser || orgID == "" || userID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}

	var req model.UpdateOperationPartnerRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}

	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.SendValidationErrorResponse(c, errs)
	}

	partner, err := h.service.Update(req, orgID, userID)
	if err != nil {
		if err.Error() == "partner not found" {
			return helper.NotFoundResponse(c, "Partner not found")
		}
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Success update operation partner", partner)
}

func (h *PartnerHandler) Detail(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	var req model.OperationPartnerDetailRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}

	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.SendValidationErrorResponse(c, errs)
	}

	partner, err := h.service.Detail(&req, orgID)
	if err != nil {
		if err.Error() == "partner not found" {
			return helper.NotFoundResponse(c, "Partner not found")
		}
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Success fetch operation partner detail", partner)
}
