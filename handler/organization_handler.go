package handler

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"service-travego/helper"
	"service-travego/internal/wagy"
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
	authService    *service.AuthService
	wagyClient     *wagy.WagyClient
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

func (h *OrganizationHandler) SetAuthService(authService *service.AuthService) {
	h.authService = authService
}

// SetWagyClient sets the wagy client
func (h *OrganizationHandler) SetWagyClient(wagyClient *wagy.WagyClient) {
	h.wagyClient = wagyClient
}

// CreateOrganization handles POST /api/organization/create
func (h *OrganizationHandler) CreateOrganization(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		fmt.Println("User not authenticated")
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}

	var req model.CreateOrganizationRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	// // Create organization model
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

	if err := h.orgService.CreateOrganizationSubscription(createdOrg.OrganizationId); err != nil {
		fmt.Println("Error creating subscription:", err.Error())
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to create subscription")
	}

	assistantAccountID := ""
	if h.wagyClient != nil && strings.TrimSpace(req.Phone) != "" {
		message := "Selamat datang di TraveGO. Kini Anda bisa menikmati fitur TraveGO dengan chat AI Assistant dan dashboard web TraveGO."
		fmt.Printf("Sending welcome WhatsApp to %s\n", req.Phone)
		if _, err := h.wagyClient.SendMessage(req.Phone, message); err != nil {
			fmt.Println("Error sending welcome WhatsApp:", err.Error())
		} else {
			assistantAccountID, err = h.orgService.CreateDefaultAssistantAccount(createdOrg.OrganizationId, userID, req.Phone)
			if err != nil {
				fmt.Println("Error creating assistant account:", err.Error())
				return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to create assistant account")
			}
		}
	}

	if h.authService == nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Auth service not initialized")
	}
	loginResponse, err := h.authService.Login("", "", "", userID)
	if err != nil {
		fmt.Println("Error generating organization creation token:", err.Error())
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	responseData := map[string]interface{}{
		"organization_id":   createdOrg.OrganizationId,
		"organizationID":    createdOrg.OrganizationId,
		"organization_code": createdOrg.OrganizationCode,
		"OrganizationCode":  createdOrg.OrganizationCode,
		"organization":      createdOrg,
		"token":             loginResponse.Token,
		"refresh_token":     loginResponse.RefreshToken,
	}
	if assistantAccountID != "" {
		responseData["assistant_account_id"] = assistantAccountID
	}

	return helper.SuccessResponse(c, fiber.StatusCreated, "Organization created successfully", responseData)
}

func (h *OrganizationHandler) JoinOrganization(c *fiber.Ctx) error {
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

	var req model.UpdateOrganizationDetailRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	if err := h.orgService.UpdateOrganizationDetail(orgID, &req); err != nil {
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

// GetUsers retrieves users from the organization with optional status filter
func (h *OrganizationHandler) GetUsers(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	status := c.Query("status", "")

	users, err := h.orgService.GetOrganizationUsers(orgID, status)
	if err != nil {
		fmt.Println("Error fetching users:", err.Error())
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load users")
	}

	res := make([]map[string]interface{}, 0, len(users))
	for i := range users {
		res = append(res, map[string]interface{}{
			"user_id":    users[i].UserID,
			"username":   users[i].Username,
			"fullname":   users[i].Name,
			"email":      users[i].Email,
			"phone":      users[i].Phone,
			"address":    users[i].Address,
			"city":       users[i].City,
			"province":   users[i].Province,
			"avatar":     users[i].Avatar,
			"created_at": users[i].CreatedAt,
			"is_active":  users[i].IsActive,
		})
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Users loaded successfully", res)
}

// HandleJoinAction handles join request actions (approve, delete, reject)
func (h *OrganizationHandler) HandleJoinAction(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	action := c.Params("action")
	userID := c.Params("user_id")

	switch action {
	case "approve":
		if err := h.orgService.ApproveJoinRequest(orgID, userID); err != nil {
			fmt.Println("Error approving join request:", err.Error())
			return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to approve join request")
		}
		return helper.SuccessResponse(c, fiber.StatusOK, "Join request approved successfully", nil)
	case "reject":
		if err := h.orgService.RejectJoinRequest(orgID, userID); err != nil {
			fmt.Println("Error rejecting join request:", err.Error())
			return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to reject join request")
		}
		return helper.SuccessResponse(c, fiber.StatusOK, "Join request rejected successfully", nil)
	case "delete":
		if err := h.orgService.RejectJoinRequest(orgID, userID); err != nil {
			fmt.Println("Error deleting join request:", err.Error())
			return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to delete join request")
		}
		return helper.SuccessResponse(c, fiber.StatusOK, "Join request deleted successfully", nil)
	default:
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid action. Allowed actions: approve, delete, reject")
	}
}

// HandleUserAction handles user actions (enable, disable)
func (h *OrganizationHandler) HandleUserAction(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	action := c.Params("action")
	userID := c.Params("user_id")

	switch action {
	case "enable":
		if err := h.orgService.ToggleUserStatus(orgID, userID, true); err != nil {
			fmt.Println("Error enabling user:", err.Error())
			return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to enable user")
		}
		return helper.SuccessResponse(c, fiber.StatusOK, "User enabled successfully", nil)
	case "disable":
		if err := h.orgService.ToggleUserStatus(orgID, userID, false); err != nil {
			fmt.Println("Error disabling user:", err.Error())
			return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to disable user")
		}
		return helper.SuccessResponse(c, fiber.StatusOK, "User disabled successfully", nil)
	default:
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid action. Allowed actions: enable, disable")
	}
}
