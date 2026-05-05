package handler

import (
	"encoding/json"
	"os"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func (h *OrganizationHandler) EmployeeAll(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	items, err := h.orgService.EmployeeAll(orgID, "")
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Employees loaded", items)
}

func (h *OrganizationHandler) EmployeeOperations(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	items, err := h.orgService.EmployeeAll(orgID, "operation")
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Employees loaded", items)
}

func (h *OrganizationHandler) EmployeeCreate(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req model.CreateEmployeeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if strings.TrimSpace(req.Avatar) == "" {
		req.Avatar = strings.TrimSpace(req.Photo)
	}
	if strings.TrimSpace(req.BirthDate) == "" {
		req.BirthDate = strings.TrimSpace(req.DateOfBirth)
	}
	if req.AddressCity == 0 && strings.TrimSpace(req.CityID) != "" {
		if v, err := strconv.Atoi(strings.TrimSpace(req.CityID)); err == nil {
			req.AddressCity = v
		} else {
			return helper.BadRequestResponse(c, "invalid city_id")
		}
	}
	if req.ContractStatus == nil && strings.TrimSpace(req.ContractTypeID) != "" {
		if v, err := strconv.Atoi(strings.TrimSpace(req.ContractTypeID)); err == nil {
			req.ContractStatus = &v
		} else {
			return helper.BadRequestResponse(c, "invalid contract_type_id")
		}
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	id, err := h.orgService.EmployeeCreate(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Employee created", fiber.Map{
		"uuid": id,
	})
}

func (h *OrganizationHandler) EmployeeUpdate(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req model.UpdateEmployeeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if strings.TrimSpace(req.Avatar) == "" {
		req.Avatar = strings.TrimSpace(req.Photo)
	}
	if strings.TrimSpace(req.BirthDate) == "" {
		req.BirthDate = strings.TrimSpace(req.DateOfBirth)
	}
	if req.AddressCity == 0 && strings.TrimSpace(req.CityID) != "" {
		if v, err := strconv.Atoi(strings.TrimSpace(req.CityID)); err == nil {
			req.AddressCity = v
		} else {
			return helper.BadRequestResponse(c, "invalid city_id")
		}
	}
	if req.ContractStatus == nil && strings.TrimSpace(req.ContractTypeID) != "" {
		if v, err := strconv.Atoi(strings.TrimSpace(req.ContractTypeID)); err == nil {
			req.ContractStatus = &v
		} else {
			return helper.BadRequestResponse(c, "invalid contract_type_id")
		}
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	if err := h.orgService.EmployeeUpdate(orgID, userID, &req); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Employee updated", nil)
}

func (h *OrganizationHandler) EmployeeDetail(c *fiber.Ctx) error {
	id := c.Params("uuid")
	if id == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "uuid is required")
	}
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	it, err := h.orgService.EmployeeDetail(orgID, id)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	if it.ContractStatus != nil {
		f, err := os.Open("config/common.json")
		if err == nil {
			defer f.Close()
			var cfg model.CommonConfig
			if err := json.NewDecoder(f).Decode(&cfg); err == nil {
				for _, ct := range cfg.ContractType {
					if ct.ID == *it.ContractStatus {
						it.ContractStatusLabel = ct.Label
						break
					}
				}
			}
		}
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Employee detail loaded", it)
}

func (h *OrganizationHandler) EmployeeDelete(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("uuid"))
	if id == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "ID is required")
	}
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	if err := h.orgService.EmployeeDelete(orgID, userID, id); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Employee deleted", nil)
}

func (h *OrganizationHandler) EmployeeShiftSchedule(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	var req model.EmployeeShiftScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	resp, err := h.orgService.EmployeeShiftSchedule(orgID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Employee shift schedule loaded", resp)
}

func (h *OrganizationHandler) EmployeeShiftSetSchedule(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req model.EmployeeShiftSetScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	typ := strings.ToLower(strings.TrimSpace(req.Type))
	if typ != "submit" && typ != "delete" {
		return helper.BadRequestResponse(c, "invalid type")
	}
	if typ == "delete" && strings.TrimSpace(req.ShiftID) == "" {
		return helper.BadRequestResponse(c, "shift_id is required")
	}
	if typ == "delete" && strings.TrimSpace(req.EmployeeID) == "" {
		return helper.BadRequestResponse(c, "employee_id is required")
	}
	if typ == "submit" {
		if len(req.Schedules) == 0 {
			if strings.TrimSpace(req.EmployeeID) == "" {
				return helper.BadRequestResponse(c, "employee_id is required")
			}
			if strings.TrimSpace(req.ShiftDate) == "" {
				return helper.BadRequestResponse(c, "shift_date is required")
			}
		} else {
			for _, it := range req.Schedules {
				if strings.TrimSpace(it.EmployeeID) == "" {
					return helper.BadRequestResponse(c, "employee_id is required")
				}
				if strings.TrimSpace(it.ShiftDate) == "" {
					return helper.BadRequestResponse(c, "shift_date is required")
				}
			}
		}
	}

	out, err := h.orgService.EmployeeShiftSetSchedule(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Employee shift schedule updated", out)
}
