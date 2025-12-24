package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type ContentHandler struct {
	service *service.ContentService
}

func NewContentHandler(service *service.ContentService) *ContentHandler {
	return &ContentHandler{
		service: service,
	}
}

// UpsertGeneralContent handles POST /content/general
func (h *ContentHandler) UpsertGeneralContent(c *fiber.Ctx) error {
	var req model.ContentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	// Get user info from context (set by JWT middleware)
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Invalid user context")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	if err := h.service.UpsertGeneralContent(req, orgID, userID); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Content saved successfully", nil)
}

// GetGeneralContent handles GET /content/general/detail/:section_tag
func (h *ContentHandler) GetGeneralContent(c *fiber.Ctx) error {
	sectionTag := c.Params("section_tag")

	if sectionTag == "" {
		return helper.BadRequestResponse(c, "section_tag is required")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	res, err := h.service.GetGeneralContent(sectionTag, orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	if res == nil {
		return helper.SuccessResponse(c, fiber.StatusOK, "Content loaded", "")
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Content loaded", res)
}
