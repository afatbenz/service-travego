package handler

import (
	"fmt"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func (h *OrganizationHandler) AssistantList(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	items, err := h.orgService.AssistantList(orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Assistant accounts loaded", items)
}

func (h *OrganizationHandler) AssistantSubmit(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req model.AssistantSubmitRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	res, err := h.orgService.AssistantSubmit(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	// Send WhatsApp notification
	if h.wagyClient != nil && req.AccountNumber != "" {
		normalized := service.NormalizeAssistantAccountNumber(req.AccountNumber)
		message := "Halo! Anda sudah bisa menikmati AI Assistant untuk memudahkan pekerjaan. Jika ada kendala harap informasikan ke administrator."
		go h.wagyClient.SendMessage(normalized, message)
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Assistant account created", res)
}

func (h *OrganizationHandler) AssistantUpdate(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	var req model.AssistantUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	// Fetch old data to check if account_number changed
	oldData, _ := h.orgService.GetAssistantAccountByID(orgID, req.AssistantID)

	if err := h.orgService.AssistantUpdate(orgID, &req); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	// Send WhatsApp notification if account_number changed
	if h.wagyClient != nil && req.AccountNumber != nil && *req.AccountNumber != "" {
		normalized := service.NormalizeAssistantAccountNumber(*req.AccountNumber)
		if oldData == nil || oldData.AccountNumber != normalized {
			message := "Halo! Anda sudah bisa menikmati AI Assistant untuk memudahkan pekerjaan. Jika ada kendala harap informasikan ke administrator."
			go h.wagyClient.SendMessage(normalized, message)
		}
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Assistant account updated", nil)
}

func (h *OrganizationHandler) AssistantDelete(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	var req model.AssistantDeleteRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	if err := h.orgService.AssistantDelete(orgID, req.EmployeeID); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Assistant account deleted", nil)
}

func (h *OrganizationHandler) AssistantWhatsAppBusinessList(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	accountNumber, err := h.orgService.AssistantWhatsAppBusinessList(orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Assistant WhatsApp business list loaded", accountNumber)
}

func (h *OrganizationHandler) AssistantWhatsAppBusinessUpdate(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	var req model.AssistantWhatsAppBusinessUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	// Validate that account number starts with "62"
	if !strings.HasPrefix(req.AccountNumber, "62") {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Account number must start with 62")
	}

	err := h.orgService.AssistantWhatsAppBusinessUpdate(orgID, req.AccountNumber)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Assistant WhatsApp business account updated", nil)
}

func (h *OrganizationHandler) EmployeeWhatsApp(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	employeeID := strings.TrimSpace(c.Params("employee_id"))
	if employeeID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "employee_id is required")
	}

	fmt.Println(employeeID, " - employeeID")
	res, err := h.orgService.EmployeeWhatsApp(orgID, employeeID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	message := "Employee WhatsApp loaded"
	if !res.HasPhone {
		message = "Employee WhatsApp is empty"
	}

	return helper.SuccessResponse(c, fiber.StatusOK, message, res)
}
