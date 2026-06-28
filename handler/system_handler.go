package handler

import (
	"database/sql"
	"service-travego/helper"
	"service-travego/model"
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

func (h *SystemHandler) GetDeviceList(c *fiber.Ctx) error {
	search := c.Query("search", "")
	status := c.Query("status", "")
	if status != "" && status != "verified" && status != "unverified" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid status value")
	}

	res, err := h.service.GetDeviceList(search, status)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Device list retrieved successfully", res)
}

func (h *SystemHandler) UpdateDevice(c *fiber.Ctx) error {
	action := c.Params("action")
	if action != "enable" && action != "disable" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid action. Use 'enable' or 'disable'")
	}

	if action == "disable" {
		var req model.DeviceDisableRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.BadRequestResponse(c, "Invalid request body")
		}
		if req.Account == "" {
			return helper.BadRequestResponse(c, "account is required")
		}

		if err := h.service.UpdateDevice(req.Account, "disable", nil); err != nil {
			if err == sql.ErrNoRows {
				return helper.NotFoundResponse(c, "Device not found")
			}
			return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
		}

		return helper.SuccessResponse(c, fiber.StatusOK, "Device disabled successfully", nil)
	}

	if action == "enable" {
		var req model.DeviceEnableRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.BadRequestResponse(c, "Invalid request body")
		}
		if req.Account == "" {
			return helper.BadRequestResponse(c, "account is required")
		}

		if err := h.service.UpdateDevice(req.Account, "enable", &req); err != nil {
			if err == sql.ErrNoRows {
				return helper.NotFoundResponse(c, "Device not found")
			}
			return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
		}

		return helper.SuccessResponse(c, fiber.StatusOK, "Device enabled successfully", nil)
	}

	return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid action")
}

func (h *SystemHandler) GetOrganizations(c *fiber.Ctx) error {
	search := c.Query("search", "")
	status := c.Query("status", "")
	if status != "" && status != "active" && status != "inactive" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid status value")
	}

	res, err := h.service.GetOrganizations(search, status)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Organizations retrieved successfully", res)
}

func (h *SystemHandler) GetUsers(c *fiber.Ctx) error {
	search := c.Query("search", "")
	isActive := c.Query("is_active", "")
	if isActive != "" && isActive != "true" && isActive != "false" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid is_active value")
	}

	res, err := h.service.GetUsers(search, isActive)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Users retrieved successfully", res)
}
