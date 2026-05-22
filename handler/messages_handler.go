package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type MessagesHandler struct {
	service *service.MessagesService
}

func NewMessagesHandler(s *service.MessagesService) *MessagesHandler {
	return &MessagesHandler{service: s}
}

func (h *MessagesHandler) SubmitMessage(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if strings.TrimSpace(orgID) == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	var req model.MessageSubmitRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}

	messageID, err := h.service.SubmitMessage(orgID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Message submitted", fiber.Map{
		"message_id": messageID,
	})
}

func (h *MessagesHandler) ListMessages(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if strings.TrimSpace(orgID) == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	items, err := h.service.ListMessages(orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Messages loaded", items)
}

func (h *MessagesHandler) ReadMessage(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if strings.TrimSpace(orgID) == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	var req model.MessageReadRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}

	if err := h.service.ReadMessage(orgID, req.MessageID); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Message updated", nil)
}

