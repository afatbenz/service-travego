package handler

import (
	"fmt"
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
		fmt.Println(err)
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

// GetContentByParent handles GET /content/:parent
func (h *ContentHandler) GetContentByParent(c *fiber.Ctx) error {
	parent := c.Params("parent")

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	res, err := h.service.GetContentByParent(parent, orgID)
	if err != nil {
		fmt.Println(err)
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Content loaded", res)
}

func (h *ContentHandler) GetContentDetailByParentAndTag(c *fiber.Ctx) error {
	parent := c.Params("parent")
	sectionTag := c.Params("section_tag")
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	res, err := h.service.GetContentDetail(parent, sectionTag, orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
	if res == nil {
		return helper.SuccessResponse(c, fiber.StatusOK, "Content loaded", "")
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Content loaded", res)
}

func (h *ContentHandler) UploadContent(c *fiber.Ctx) error {
	sectionTag := c.FormValue("section_tag")
	parent := c.FormValue("parent")
	if sectionTag == "" || parent == "" {
		return helper.BadRequestResponse(c, "section_tag and parent are required")
	}

	file, err := c.FormFile("file_path")
	if err != nil {
		file, err = c.FormFile("file")
		if err != nil {
			return helper.BadRequestResponse(c, "file_path (file) is required")
		}
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Invalid user context")
	}
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	url, err := h.service.UploadContent(file, parent, sectionTag, orgID, userID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Content uploaded successfully", map[string]string{
		"url": url,
	})
}

// GetAllGeneralContent handles GET /content
func (h *ContentHandler) GetAllGeneralContent(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}

	res, err := h.service.GetAllGeneralContent(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Content loaded", res)
}

func (h *ContentHandler) DeleteListByUUID(c *fiber.Ctx) error {
	uuid := c.Params("uuid")
	if uuid == "" {
		return helper.BadRequestResponse(c, "uuid is required")
	}
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusBadRequest, "Missing organization context")
	}
	if err := h.service.DeleteContentByUUID(uuid, orgID); err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "sql: no rows in result set" {
			status = fiber.StatusNotFound
		}
		return helper.SendErrorResponse(c, status, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Content deleted", nil)
}
