package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type OrganizationHandler struct {
	orgService     *service.OrganizationService
	orgJoinService *service.OrganizationJoinService
	orgTypeService *service.OrganizationTypeService
}

func NewOrganizationHandler(orgService *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{
		orgService: orgService,
	}
}

// SetJoinService sets the organization join service
func (h *OrganizationHandler) SetJoinService(orgJoinService *service.OrganizationJoinService) {
	h.orgJoinService = orgJoinService
}

// SetOrganizationTypeService sets the organization type service
func (h *OrganizationHandler) SetOrganizationTypeService(orgTypeService *service.OrganizationTypeService) {
	h.orgTypeService = orgTypeService
}

// CreateOrganization handles POST /api/organization/create
func (h *OrganizationHandler) CreateOrganization(c *fiber.Ctx) error {
	// Get user_id from locals (set by JWT middleware)
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}

	var req model.CreateOrganizationRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	// Create organization model
	org := &model.Organization{
		OrganizationCode: req.OrganizationCode,
		OrganizationName: req.OrganizationName,
		CompanyName:      req.CompanyName,
		Address:          req.Address,
		City:             req.City,
		Province:         req.Province,
		Phone:            req.Phone,
		Email:            req.Email,
		NPWPNumber:       req.NPWPNumber,
		OrganizationType: req.OrganizationType,
		PostalCode:       req.PostalCode,
	}

    createdOrg, err := h.orgService.CreateOrganization(userID, org)
    if err != nil {
        statusCode := fiber.StatusInternalServerError
        if strings.Contains(err.Error(), "profile") || strings.Contains(err.Error(), "complete") {
            statusCode = fiber.StatusBadRequest
        }
        return helper.SendErrorResponse(c, statusCode, err.Error())
    }
    responseData := map[string]interface{}{
        "organization_code": createdOrg.OrganizationCode,
        "organization":      createdOrg,
    }

    return helper.SuccessResponse(c, fiber.StatusCreated, "Organization created successfully", responseData)
}

func (h *OrganizationHandler) JoinOrganization(c *fiber.Ctx) error {
	// Get user_id from locals (set by JWT middleware)
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}

	var req model.JoinOrganizationRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	if err := h.orgJoinService.JoinOrganization(userID, req.OrganizationCode); err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Successfully joined organization. Waiting for approval.", nil)
}

// GetOrganizationTypes handles GET /api/organization/types
func (h *OrganizationHandler) GetOrganizationTypes(c *fiber.Ctx) error {
	if h.orgTypeService == nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Organization type service not initialized")
	}

	orgTypes, err := h.orgTypeService.GetAllOrganizationTypes()
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load organization types")
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Organization types loaded successfully", orgTypes)
}
