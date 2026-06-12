package handler

import (
	"service-travego/helper"
	"service-travego/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type NotificationHandler struct {
	service *service.NotificationService
}

func NewNotificationHandler(s *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: s}
}

func (h *NotificationHandler) GetAllNotifications(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || strings.TrimSpace(orgID) == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "missing organization context")
	}

	items, err := h.service.GetNotifications(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Notifications loaded", items)
}

func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || strings.TrimSpace(orgID) == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "missing organization context")
	}

	notificationID := c.Params("notification_id")
	if err := h.service.MarkAsRead(orgID, notificationID); err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Notification updated", nil)
}
