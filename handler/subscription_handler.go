package handler

import (
	"service-travego/helper"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type SubscriptionHandler struct {
	subscriptionService *service.SubscriptionService
}

type SubmitSubscriptionRequest struct {
	PackageID string `json:"package_id"`
}

func NewSubscriptionHandler(subscriptionService *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionService: subscriptionService,
	}
}

func (h *SubscriptionHandler) GetSubscription(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)

	if !ok || orgID == "" {
		return helper.UnauthorizedResponse(c, "Organization not authenticated")
	}

	if _, err := uuid.Parse(orgID); err != nil {
		return helper.BadRequestResponse(c, "Invalid organization ID format")
	}

	subscription, err := h.subscriptionService.GetSubscription(orgID)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Subscription retrieved successfully", subscription)
}

func (h *SubscriptionHandler) GetSubscriptionHistory(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}
	if _, err := uuid.Parse(userID); err != nil {
		return helper.BadRequestResponse(c, "Invalid user ID format")
	}
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.UnauthorizedResponse(c, "Organization not authenticated")
	}
	if _, err := uuid.Parse(orgID); err != nil {
		return helper.BadRequestResponse(c, "Invalid organization ID format")
	}

	subscriptions, err := h.subscriptionService.GetSubscriptionHistory(userID, orgID)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Subscription history retrieved successfully", subscriptions)
}

func (h *SubscriptionHandler) SubmitSubscription(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}
	if _, err := uuid.Parse(userID); err != nil {
		return helper.BadRequestResponse(c, "Invalid user ID format")
	}
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.UnauthorizedResponse(c, "Organization not authenticated")
	}
	if _, err := uuid.Parse(orgID); err != nil {
		return helper.BadRequestResponse(c, "Invalid organization ID format")
	}

	var req SubmitSubscriptionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if req.PackageID == "" {
		return helper.BadRequestResponse(c, "Package ID is required")
	}

	// Decrypt package ID
	packageID, err := helper.DecryptString(req.PackageID)
	if err != nil {
		return helper.BadRequestResponse(c, "Invalid package ID")
	}

	result, err := h.subscriptionService.SubmitSubscriptionPayment(packageID, userID, orgID)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Subscription payment submitted successfully", result)
}

func (h *SubscriptionHandler) GetSubscriptionSummary(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.UnauthorizedResponse(c, "User not authenticated")
	}
	if _, err := uuid.Parse(userID); err != nil {
		return helper.BadRequestResponse(c, "Invalid user ID format")
	}
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.UnauthorizedResponse(c, "Organization not authenticated")
	}
	if _, err := uuid.Parse(orgID); err != nil {
		return helper.BadRequestResponse(c, "Invalid organization ID format")
	}

	var req SubmitSubscriptionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if req.PackageID == "" {
		return helper.BadRequestResponse(c, "Package ID is required")
	}

	// Decrypt package ID
	packageID, err := helper.DecryptString(req.PackageID)
	if err != nil {
		return helper.BadRequestResponse(c, "Invalid package ID")
	}

	result, err := h.subscriptionService.GetSubscriptionSummary(packageID, orgID)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Subscription summary retrieved successfully", result)
}

func (h *SubscriptionHandler) GetSubscriptionDetailByInvoice(c *fiber.Ctx) error {
	invoiceNumber := c.Params("invoicenumber")
	if invoiceNumber == "" {
		return helper.BadRequestResponse(c, "Invoice number is required")
	}

	result, err := h.subscriptionService.GetSubscriptionDetailByInvoice(invoiceNumber)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Subscription detail retrieved successfully", result)
}
