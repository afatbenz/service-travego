package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type ScheduleHandler struct {
	service *service.ScheduleService
}

func NewScheduleHandler(s *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{service: s}
}

func (h *ScheduleHandler) Create(c *fiber.Ctx) error {
	var req model.ScheduleCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if validationErrors := helper.ValidateStruct(&req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	departureTime := strings.TrimSpace(req.DepartureTime)
	departureStart := strings.TrimSpace(req.DepartureStart)
	if departureTime == "" && departureStart == "" {
		return helper.BadRequestResponse(c, "departure_time is required")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.BadRequestResponse(c, "missing user context")
	}

	id, err := h.service.CreateSchedule(model.ScheduleCreateServiceInput{
		OrganizationID: orgID,
		UserID:         userID,
		Request:        &req,
	})
	if err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Schedule created", fiber.Map{
		"schedule_id": id,
	})
}
