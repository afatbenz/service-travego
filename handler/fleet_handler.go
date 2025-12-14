package handler

import (
	"encoding/json"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type FleetHandler struct {
	service *service.FleetService
}

func NewFleetHandler(s *service.FleetService) *FleetHandler {
	return &FleetHandler{service: s}
}

func (h *FleetHandler) CreateFleet(c *fiber.Ctx) error {
	var req model.CreateFleetRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	// Extract body.images from raw JSON payload if present
	if raw := c.Body(); len(raw) > 0 {
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err == nil {
			if b, ok := m["body"].(map[string]interface{}); ok {
				if imgs, ok := b["images"].([]interface{}); ok {
					req.BodyImages = make([]string, 0, len(imgs))
					for _, v := range imgs {
						if s, ok := v.(string); ok && s != "" {
							req.BodyImages = append(req.BodyImages, s)
						}
					}
				}
			}
		}
	}
	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("organization_id").(string)
	if userID == "" || orgID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}
	id, err := h.service.CreateFleet(userID, orgID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet created", fiber.Map{
		"fleet_id": id,
	})
}
