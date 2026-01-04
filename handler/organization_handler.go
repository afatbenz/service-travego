package handler

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strconv"
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
		City:             strconv.Itoa(req.City),
		Province:         strconv.Itoa(req.Province),
		Phone:            req.Phone,
		Email:            req.Email,
		NPWPNumber:       req.NPWPNumber,
		OrganizationType: req.OrganizationType,
		PostalCode:       req.PostalCode,
	}

	createdOrg, err := h.orgService.CreateOrganization(userID, org)
	if err != nil {
		fmt.Println("Error creating organization:", err.Error())
		statusCode := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "profile") || strings.Contains(err.Error(), "complete") || strings.Contains(err.Error(), "invalid") || strings.Contains(strings.ToLower(err.Error()), "foreign key") {
			statusCode = fiber.StatusBadRequest
		}
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}
	responseData := map[string]interface{}{
		"organization_id":   createdOrg.ID,
		"organizationID":    createdOrg.ID,
		"organization_code": createdOrg.OrganizationCode,
		"OrganizationCode":  createdOrg.OrganizationCode,
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

	if h.orgJoinService == nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Join service not initialized")
	}

	err := h.orgJoinService.JoinOrganization(userID, req.OrganizationCode)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Join request submitted successfully", nil)
}

// GetAPIConfig handles GET /api/organization/api-config
func (h *OrganizationHandler) GetAPIConfig(c *fiber.Ctx) error {
	// Get user_id from locals (set by JWT middleware)
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}

	config, err := h.orgService.GetAPIConfig(userID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "API config retrieved successfully", config)
}

// UpdateDomainURL handles POST /api/organization/update/domain-url
func (h *OrganizationHandler) UpdateDomainURL(c *fiber.Ctx) error {
	// Get user_id from locals
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}

	// Get organization_id from locals
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	var req struct {
		DomainURL string `json:"domain_url"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	err := h.orgService.UpdateDomainURL(userID, orgID, req.DomainURL)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Domain URL updated successfully", nil)
}

// GetOrganizationTypes handles GET /api/organization/types
func (h *OrganizationHandler) GetOrganizationTypes(c *fiber.Ctx) error {
	if h.orgTypeService == nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Organization type service not initialized")
	}

	types, err := h.orgTypeService.GetAllOrganizationTypes()
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Organization types retrieved successfully", types)
}

// GetBankAccounts handles GET /api/organization/bank-accounts
func (h *OrganizationHandler) GetBankAccounts(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	accounts, err := h.orgService.GetBankAccounts(orgID)
	if err != nil {
		fmt.Println("Error fetching bank accounts:", err.Error())
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load bank accounts")
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Bank accounts loaded successfully", accounts)
}

// GetOrganizationDetail handles GET /api/organization/detail
func (h *OrganizationHandler) GetOrganizationDetail(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	res, err := h.orgService.GetOrganizationDetail(orgID)
	if err != nil {
		fmt.Println("Error fetching organization detail:", err.Error())
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Organization detail retrieved", res)
}

// UpdateOrganizationDetail handles POST /api/organization/update
func (h *OrganizationHandler) UpdateOrganizationDetail(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	var payload map[string]interface{}
	if err := c.BodyParser(&payload); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if payload["organization_name"] == nil || payload["company_name"] == nil || payload["phone"] == nil || payload["address"] == nil || payload["email"] == nil {
		return helper.BadRequestResponse(c, "organization_name, company_name, phone, address, email wajib")
	}

	if err := h.orgService.UpdateOrganizationDetail(orgID, payload); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Organization updated successfully", nil)
}

// UpdateOrganizationLogo handles POST /api/organization/update/logo
func (h *OrganizationHandler) UpdateOrganizationLogo(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	// Prefer multipart file upload if provided
	fileHeader, err := c.FormFile("file")
	if err != nil || fileHeader == nil {
		// try alternative field names
		fileHeader, err = c.FormFile("logo")
		if err != nil || fileHeader == nil {
			fileHeader, err = c.FormFile("filepath")
		}
	}

	if fileHeader != nil && err == nil {
		tempDir := os.TempDir()
		tempPath := filepath.Join(tempDir, fileHeader.Filename)
		if saveErr := c.SaveFile(fileHeader, tempPath); saveErr != nil {
			return helper.BadRequestResponse(c, "failed to save uploaded file")
		}
		defer os.Remove(tempPath)

		url, svcErr := h.orgService.UpdateOrganizationLogo(orgID, tempPath)
		if svcErr != nil {
			code := service.GetStatusCode(svcErr)
			return helper.SendErrorResponse(c, code, svcErr.Error())
		}
		return helper.SuccessResponse(c, fiber.StatusOK, "Logo updated successfully", map[string]string{"logo": url})
	}

	// Fallback: JSON body with file_path
	var payload struct {
		FilePath string `json:"file_path"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}
	if payload.FilePath == "" {
		return helper.BadRequestResponse(c, "file atau file_path wajib")
	}
	url, svcErr := h.orgService.UpdateOrganizationLogo(orgID, payload.FilePath)
	if svcErr != nil {
		code := service.GetStatusCode(svcErr)
		return helper.SendErrorResponse(c, code, svcErr.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Logo updated successfully", map[string]string{"logo": url})
}

// CreateBankAccount handles POST /api/organization/bank-account/create
func (h *OrganizationHandler) CreateBankAccount(c *fiber.Ctx) error {
	var req model.CreateOrganizationBankAccountRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	fmt.Printf("Received CreateBankAccount Request: %+v\n", req)

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	// Use X-Forwarded-For or X-Forwarded-Fot (as per user request typo?)
	// Assuming user meant X-Forwarded-For, but checking both just in case or sticking to standard.
	// User said "x-forwarded-fot".
	createdProxy := c.Get("X-Forwarded-For")
	if createdProxy == "" {
		createdProxy = c.Get("X-Forwarded-Fot")
	}

	createdIP := c.IP()

	err := h.orgService.CreateBankAccount(&req, orgID, userID, createdProxy, createdIP)
	if err != nil {
		fmt.Println("Error creating bank account:", err.Error())
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Bank account created successfully", nil)
}

// UpdateBankAccount handles POST/PUT /api/organization/bank-account/update
func (h *OrganizationHandler) UpdateBankAccount(c *fiber.Ctx) error {
	// Get organization_id from locals
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	var req model.UpdateOrganizationBankAccountRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Basic validation
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	updatedProxy := c.Get("X-Forwarded-For")
	if updatedProxy == "" {
		updatedProxy = c.Get("X-Forwarded-Fot")
	}
	updatedIP := c.IP()

	err := h.orgService.UpdateBankAccount(&req, orgID, updatedProxy, updatedIP)
	if err != nil {
		fmt.Println("Error updating bank account:", err.Error())
		if strings.Contains(err.Error(), "simultaneously") || strings.Contains(err.Error(), "required") {
			return helper.SendErrorResponse(c, fiber.StatusBadRequest, err.Error())
		}
		if err == sql.ErrNoRows {
			return helper.SendErrorResponse(c, fiber.StatusNotFound, "Bank account not found or unauthorized")
		}
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Bank account updated successfully", nil)
}

// DeleteBankAccount handles POST /api/organization/bank-account/delete
func (h *OrganizationHandler) DeleteBankAccount(c *fiber.Ctx) error {
	// Get organization_id from locals
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	var req model.DeleteOrganizationBankAccountRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Basic validation
	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	err := h.orgService.DeleteBankAccount(req.BankAccountID, orgID)
	if err != nil {
		fmt.Println("Error deleting bank account:", err.Error())
		if err == sql.ErrNoRows {
			return helper.SendErrorResponse(c, fiber.StatusNotFound, "Bank account not found or unauthorized")
		}
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Bank account deleted successfully", nil)
}
