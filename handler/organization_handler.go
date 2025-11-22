package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type OrganizationHandler struct {
	orgService *service.OrganizationService
}

func NewOrganizationHandler(orgService *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{
		orgService: orgService,
	}
}

// CreateOrganizationRequest represents create organization request payload
type CreateOrganizationRequest struct {
	OrganizationName string `json:"organization_name" validate:"required"`
	CompanyName      string `json:"company_name" validate:"required"`
	Address          string `json:"address" validate:"required"`
	City             string `json:"city" validate:"required"`
	Province         string `json:"province" validate:"required"`
	Phone            string `json:"phone" validate:"required"`
	Email            string `json:"email" validate:"required,email"`
}

// CreateOrganization handles POST /api/organization
func (h *OrganizationHandler) CreateOrganization(c *fiber.Ctx) error {
	var req CreateOrganizationRequest

	userID := c.Get("user_id")
	if userID == "" {
		return helper.UnauthorizedResponse(c, "User ID is required")
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	// Create organization model
	org := &model.Organization{
		OrganizationName: req.OrganizationName,
		CompanyName:      req.CompanyName,
		Address:          req.Address,
		City:             req.City,
		Province:         req.Province,
		Phone:            req.Phone,
		Email:            req.Email,
	}

	createdOrg, err := h.orgService.CreateOrganization(userID, org)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "user must complete") {
			statusCode = fiber.StatusBadRequest
		}
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusCreated, "Organization created successfully", createdOrg)
}
