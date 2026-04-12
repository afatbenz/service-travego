package handler

import (
	"log"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type FleetUnitHandler struct {
	service *service.FleetUnitService
}

func NewFleetUnitHandler(s *service.FleetUnitService) *FleetUnitHandler {
	return &FleetUnitHandler{service: s}
}

func (h *FleetUnitHandler) List(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	items, err := h.service.List(orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet units loaded", items)
}

func (h *FleetUnitHandler) Create(c *fiber.Ctx) error {
	var req model.FleetUnitCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if errs := helper.ValidateStruct(&req); len(errs) > 0 {
		return helper.SendValidationErrorResponse(c, errs)
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.BadRequestResponse(c, "missing user context")
	}

	id, err := h.service.Create(orgID, userID, &req)
	if err != nil {
		log.Printf("[ERROR] TransactionID: %s - CreateFleetUnit - Error: %v", helper.GetTransactionID(c), err)
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet unit created", fiber.Map{
		"unit_id": id,
	})
}

func (h *FleetUnitHandler) Update(c *fiber.Ctx) error {
	var req model.FleetUnitUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if errs := helper.ValidateStruct(&req); len(errs) > 0 {
		return helper.SendValidationErrorResponse(c, errs)
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.BadRequestResponse(c, "missing user context")
	}

	if err := h.service.Update(orgID, userID, &req); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet unit updated", nil)
}

func (h *FleetUnitHandler) Detail(c *fiber.Ctx) error {
	id := c.Params("unit_id")
	if id == "" {
		return helper.BadRequestResponse(c, "unit_id is required")
	}
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	res, err := h.service.Detail(orgID, id)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet unit detail loaded", res)
}
