package handler

import (
	"encoding/json"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strconv"

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
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 != nil {
			return helper.BadRequestResponse(c, "invalid payload")
		}
		if v, ok := m["fleet_name"].(string); ok {
			req.FleetName = v
		}
		if v, ok := m["fleet_type"].(string); ok {
			req.FleetType = v
		}
		if v, ok := m["capacity"]; ok {
			req.Capacity = toInt(v)
		}
		if v, ok := m["production_year"]; ok {
			req.ProductionYear = toInt(v)
		}
		if v, ok := m["engine"].(string); ok {
			req.Engine = v
		}
		if v, ok := m["body"].(string); ok {
			req.Body = v
		}
		if v, ok := m["description"].(string); ok {
			req.Description = v
		}
		if v, ok := m["active"]; ok {
			switch vv := v.(type) {
			case bool:
				req.Active = vv
			case string:
				b, _ := strconv.ParseBool(vv)
				req.Active = b
			case float64:
				req.Active = vv != 0
			}
		}
		if v, ok := m["pickup_point"].([]interface{}); ok {
			req.PickupPoint = make([]int, 0, len(v))
			for _, it := range v {
				req.PickupPoint = append(req.PickupPoint, toInt(it))
			}
		}
		if v, ok := m["fascilities"].([]interface{}); ok {
			req.Facilities = make([]string, 0, len(v))
			for _, it := range v {
				if s, ok := it.(string); ok {
					req.Facilities = append(req.Facilities, s)
				}
			}
		}
		if v, ok := m["prices"].([]interface{}); ok {
			req.Prices = make([]model.FleetPriceRequest, 0, len(v))
			for _, it := range v {
				if mp, ok := it.(map[string]interface{}); ok {
					pr := model.FleetPriceRequest{}
					if dv, ok := mp["duration"]; ok {
						pr.Duration = toInt(dv)
					}
					if rv, ok := mp["rent_category"]; ok {
						pr.RentCategory = toInt(rv)
					}
					if pv, ok := mp["price"]; ok {
						pr.Price = toInt(pv)
					}
					req.Prices = append(req.Prices, pr)
				}
			}
		}
		if v, ok := m["addon"].([]interface{}); ok {
			req.Addon = make([]model.FleetAddonRequest, 0, len(v))
			for _, it := range v {
				if mp, ok := it.(map[string]interface{}); ok {
					ad := model.FleetAddonRequest{}
					if nv, ok := mp["addon_name"].(string); ok {
						ad.AddonName = nv
					}
					if dv, ok := mp["description"].(string); ok {
						ad.Description = dv
					}
					if pv, ok := mp["price"]; ok {
						ad.Price = toInt(pv)
					}
					req.Addon = append(req.Addon, ad)
				}
			}
		}
		if v, ok := m["thumbnail"].(string); ok {
			req.Thumbnail = v
		}
		if imgs, ok := m["images"].([]interface{}); ok {
			req.BodyImages = make([]string, 0, len(imgs))
			for _, it := range imgs {
				if s, ok := it.(string); ok && s != "" {
					req.BodyImages = append(req.BodyImages, s)
				}
			}
		}
		if b, ok := m["body"].(map[string]interface{}); ok {
			if imgs, ok := b["images"].([]interface{}); ok {
				if req.BodyImages == nil {
					req.BodyImages = make([]string, 0, len(imgs))
				}
				for _, it := range imgs {
					if s, ok := it.(string); ok && s != "" {
						req.BodyImages = append(req.BodyImages, s)
					}
				}
			}
			if v, ok := b["label"].(string); ok && req.Body == "" {
				req.Body = v
			}
		}
	} else {
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
				if imgs, ok := m["images"].([]interface{}); ok {
					req.BodyImages = make([]string, 0, len(imgs))
					for _, it := range imgs {
						if s, ok := it.(string); ok && s != "" {
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

func (h *FleetHandler) ListFleets(c *fiber.Ctx) error {
	var req model.ListFleetRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["fleet_type"].(string); ok {
				req.FleetType = v
			}
			if v, ok := m["fleet_body"].(string); ok {
				req.FleetBody = v
			}
			if v, ok := m["fleet_engine"].(string); ok {
				req.FleetEngine = v
			}
			if v, ok := m["pickup_location"]; ok {
				req.PickupLocation = toInt(v)
			}
		}
	}
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	req.OrganizationID = orgID
	items, err := h.service.ListFleets(&req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet list loaded", items)
}

func (h *FleetHandler) FleetDetail(c *fiber.Ctx) error {
	var req model.FleetDetailRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["fleet_id"].(string); ok {
				req.FleetID = v
			}
		}
	}
	if req.FleetID == "" {
		return helper.BadRequestResponse(c, "fleet_id is required")
	}
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	res, err := h.service.GetFleetDetail(orgID, req.FleetID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet detail loaded", res)
}

func toInt(v interface{}) int {
	switch vv := v.(type) {
	case float64:
		return int(vv)
	case string:
		n, _ := strconv.Atoi(vv)
		return n
	case json.Number:
		n, _ := vv.Int64()
		return int(n)
	default:
		return 0
	}
}
