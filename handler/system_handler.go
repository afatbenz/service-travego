package handler

import (
	"service-travego/helper"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type SystemHandler struct {
	service *service.SystemService
}

func NewSystemHandler(service *service.SystemService) *SystemHandler {
	return &SystemHandler{
		service: service,
	}
}

func (h *SystemHandler) GetSystemSummarymarize(c *fiber.Ctx) error {
	period := c.Query("period", "this_month")
	validPeriods := map[string]bool{
		"this_month": true,
		"last_month": true,
		"this_year":  true,
		"last_year":  true,
		"all_time":   true,
	}
	if !validPeriods[period] {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid period value")
	}

	res, err := h.service.GetSystemSummarize(period)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "System retrieved successfully", res)
}
