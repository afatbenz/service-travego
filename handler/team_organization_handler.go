package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func (h *OrganizationHandler) TeamListDivisions(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	items, err := h.orgService.ListDivisions(orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Divisions loaded", items)
}

func (h *OrganizationHandler) TeamCreateDivision(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req model.CreateOrganizationDivisionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	id, err := h.orgService.CreateDivision(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Division created", fiber.Map{
		"division_id": id,
	})
}

func (h *OrganizationHandler) TeamUpdateDivision(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req model.UpdateOrganizationDivisionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	if err := h.orgService.UpdateDivision(orgID, userID, &req); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Division updated", nil)
}

func (h *OrganizationHandler) TeamDeleteDivision(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req model.DeleteOrganizationDivisionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	if err := h.orgService.DeleteDivision(orgID, userID, req.DivisionID); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Division deleted", nil)
}

func (h *OrganizationHandler) TeamListRoles(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	items, err := h.orgService.ListRoles(orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Roles loaded", items)
}

func (h *OrganizationHandler) TeamCreateRole(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req model.CreateOrganizationRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	id, err := h.orgService.CreateRole(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Role created", fiber.Map{
		"role_id": id,
	})
}

func (h *OrganizationHandler) TeamUpdateRole(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req model.UpdateOrganizationRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	if err := h.orgService.UpdateRole(orgID, userID, &req); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Role updated", nil)
}

func (h *OrganizationHandler) TeamDeleteRole(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req model.DeleteOrganizationRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	if err := h.orgService.DeleteRole(orgID, userID, req.RoleID); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Role deleted", nil)
}
