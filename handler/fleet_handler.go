package handler

import (
	"encoding/json"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strconv"
	"strings"
	"time"

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
		if v, ok := m["fuel_type"].(string); ok {
			req.FuelType = v
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
			req.Pickup = make([]model.FleetPickupRequest, 0, len(v))
			for _, it := range v {
				req.Pickup = append(req.Pickup, model.FleetPickupRequest{CityID: toInt(it)})
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
			req.Pricing = make([]model.FleetPriceRequest, 0, len(v))
			for _, it := range v {
				if mp, ok := it.(map[string]interface{}); ok {
					pr := model.FleetPriceRequest{}
					if dv, ok := mp["duration"]; ok {
						pr.Duration = toInt(dv)
					}
					if rv, ok := mp["rent_category"]; ok {
						pr.RentType = toInt(rv)
					}
					if pv, ok := mp["price"]; ok {
						pr.Price = toInt(pv)
					}
					if uom, ok := mp["uom"].(string); ok {
						pr.Uom = uom
					}
					req.Pricing = append(req.Pricing, pr)
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
						ad.AddonDesc = dv
					}
					if pv, ok := mp["price"]; ok {
						ad.AddonPrice = toInt(pv)
					}
					req.Addon = append(req.Addon, ad)
				}
			}
		}
		if v, ok := m["thumbnail"].(string); ok {
			req.Thumbnail = v
		}
		if imgs, ok := m["images"].([]interface{}); ok {
			req.Images = make([]model.FleetImageRequest, 0, len(imgs))
			for _, it := range imgs {
				if s, ok := it.(string); ok && s != "" {
					req.Images = append(req.Images, model.FleetImageRequest{PathFile: s})
				}
			}
		}
		if b, ok := m["body"].(map[string]interface{}); ok {
			if imgs, ok := b["images"].([]interface{}); ok {
				if req.Images == nil {
					req.Images = make([]model.FleetImageRequest, 0, len(imgs))
				}
				for _, it := range imgs {
					if s, ok := it.(string); ok && s != "" {
						req.Images = append(req.Images, model.FleetImageRequest{PathFile: s})
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
				if req.FuelType == "" {
					if v, ok := m["fuel_type"].(string); ok {
						req.FuelType = v
					}
				}
				if req.Thumbnail == "" {
					if v, ok := m["thumbnail"].(string); ok {
						req.Thumbnail = v
					}
				}
				if len(req.Pickup) == 0 {
					if v, ok := m["pickup_point"].([]interface{}); ok {
						req.Pickup = make([]model.FleetPickupRequest, 0, len(v))
						for _, it := range v {
							req.Pickup = append(req.Pickup, model.FleetPickupRequest{CityID: toInt(it)})
						}
					}
				}
				if len(req.Facilities) == 0 {
					if v, ok := m["fascilities"].([]interface{}); ok {
						req.Facilities = make([]string, 0, len(v))
						for _, it := range v {
							if s, ok := it.(string); ok {
								req.Facilities = append(req.Facilities, s)
							}
						}
					}
				}
				if len(req.Pricing) == 0 {
					if v, ok := m["prices"].([]interface{}); ok {
						req.Pricing = make([]model.FleetPriceRequest, 0, len(v))
						for _, it := range v {
							if mp, ok := it.(map[string]interface{}); ok {
								pr := model.FleetPriceRequest{}
								if dv, ok := mp["duration"]; ok {
									pr.Duration = toInt(dv)
								}
								if rv, ok := mp["rent_category"]; ok {
									pr.RentType = toInt(rv)
								}
								if pv, ok := mp["price"]; ok {
									pr.Price = toInt(pv)
								}
								if uom, ok := mp["uom"].(string); ok {
									pr.Uom = uom
								}
								req.Pricing = append(req.Pricing, pr)
							}
						}
					}
				}
				if b, ok := m["body"].(map[string]interface{}); ok {
					if imgs, ok := b["images"].([]interface{}); ok {
						req.Images = make([]model.FleetImageRequest, 0, len(imgs))
						for _, v := range imgs {
							if s, ok := v.(string); ok && s != "" {
								req.Images = append(req.Images, model.FleetImageRequest{PathFile: s})
							}
						}
					}
				}
				if imgs, ok := m["images"].([]interface{}); ok {
					req.Images = make([]model.FleetImageRequest, 0, len(imgs))
					for _, it := range imgs {
						if s, ok := it.(string); ok && s != "" {
							req.Images = append(req.Images, model.FleetImageRequest{PathFile: s})
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

func (h *FleetHandler) UpdateFleet(c *fiber.Ctx) error {
	var req model.UpdateFleetRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if req.FleetID == "" {
		return helper.BadRequestResponse(c, "fleet_id is required")
	}
	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("organization_id").(string)
	if userID == "" || orgID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}
	if err := h.service.UpdateFleet(userID, orgID, &req); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet updated", nil)
}

func (h *FleetHandler) ListFleets(c *fiber.Ctx) error {
	searchType := strings.ToLower(strings.TrimSpace(c.Query("search_type")))
	searchFor := strings.TrimSpace(c.Query("search_for"))

	var req model.ListFleetRequest
	req.FleetType = c.Query("fleet_type")
	req.FleetName = c.Query("fleet_name")
	req.FleetBody = c.Query("fleet_body")
	req.FleetEngine = c.Query("fleet_engine")
	if v := c.Query("pickup_location"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			req.PickupLocation = n
		}
	}

	if len(c.Body()) > 0 {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["fleet_type"].(string); ok {
				req.FleetType = v
			}
			if v, ok := m["fleet_name"].(string); ok {
				req.FleetName = v
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

	if searchType == "unit" {
		items, err := h.service.ListFleetsForUnit(orgID, searchFor)
		if err != nil {
			code := service.GetStatusCode(err)
			return helper.SendErrorResponse(c, code, err.Error())
		}
		return helper.SuccessResponse(c, fiber.StatusOK, "Fleet list loaded", items)
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

func (h *FleetHandler) DeleteFleet(c *fiber.Ctx) error {
	var req model.FleetDeleteRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if errs := helper.ValidateStruct(&req); len(errs) > 0 {
		return helper.SendValidationErrorResponse(c, errs)
	}

	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("organization_id").(string)
	if userID == "" || orgID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}

	if err := h.service.DeleteFleet(orgID, userID, req.FleetID); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet deleted", nil)
}

func (h *FleetHandler) GetPartnerOrderList(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	var filter model.PartnerOrderListFilter
	if v := strings.TrimSpace(c.Query("start_date")); v != "" {
		filter.StartDateFrom = v
	}
	if v := strings.TrimSpace(c.Query("end_date")); v != "" {
		filter.StartDateTo = v
	}
	if v := strings.TrimSpace(c.Query("order_date_start")); v != "" {
		filter.OrderDateFrom = v
	}
	if v := strings.TrimSpace(c.Query("order_date_end")); v != "" {
		filter.OrderDateTo = v
	}
	if v := strings.TrimSpace(c.Query("payment_status")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.PaymentStatus = n
			filter.HasPaymentStatus = true
		}
	}
	if strings.TrimSpace(filter.OrderDateFrom) == "" && strings.TrimSpace(filter.OrderDateTo) == "" {
		now := time.Now()
		from := now.AddDate(-1, 0, 0)
		filter.OrderDateFrom = from.Format("2006-01-02") + " 00:00:00"
		filter.OrderDateTo = now.Format("2006-01-02") + " 23:59:59"
	}
	show := strings.ToLower(strings.TrimSpace(c.Query("show")))

	res, err := h.service.GetPartnerOrdersWithSummary(orgID, &filter)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	if show == "summary" {
		return helper.SuccessResponse(c, fiber.StatusOK, "Order summary loaded", res.Summary)
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Order list loaded", res)
}

func (h *FleetHandler) GetPartnerOrderDetail(c *fiber.Ctx) error {
	orderID := c.Params("order_id")
	if orderID == "" {
		return helper.BadRequestResponse(c, "order_id is required")
	}
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	res, err := h.service.GetPartnerOrderDetail(orderID, orgID)
	if err != nil {
		code := fiber.StatusInternalServerError
		if err.Error() == "order not found or access denied" {
			code = fiber.StatusNotFound
		}
		return helper.SendErrorResponse(c, code, err.Error())
	}

	payment, err := h.service.GetPartnerOrderPaymentSummary(orderID, orgID, res.TotalAmount)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to load payment")
	}

	raw, _ := json.Marshal(res)
	var m map[string]interface{}
	_ = json.Unmarshal(raw, &m)
	m["payment"] = payment
	delete(m, "payment_status")

	return helper.SuccessResponse(c, fiber.StatusOK, "Order detail loaded", m)
}

func (h *FleetHandler) GetFleetPricesByFleetID(c *fiber.Ctx) error {
	fleetID := c.Params("fleetid")
	if fleetID == "" {
		return helper.BadRequestResponse(c, "fleetid is required")
	}
	typeID := c.Params("typeid")
	if typeID == "" {
		return helper.BadRequestResponse(c, "typeid is required")
	}

	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	items, err := h.service.GetFleetPricesByFleetID(orgID, fleetID, typeID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet prices loaded", items)
}

func (h *FleetHandler) GetFleetAddonList(c *fiber.Ctx) error {
	fleetID := c.Params("fleetid")
	if fleetID == "" {
		return helper.BadRequestResponse(c, "fleetid is required")
	}

	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	items, err := h.service.GetFleetAddonList(orgID, fleetID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet addon loaded", items)
}

func (h *FleetHandler) CreatePartnerOrder(c *fiber.Ctx) error {
	var req model.FleetOrderCreateRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 != nil {
			return helper.BadRequestResponse(c, "invalid payload")
		}
		if v, ok := m["fleet_id"].(string); ok {
			req.FleetID = v
		}
		if v, ok := m["customer_id"].(string); ok {
			req.CustomerID = v
		}
		if v, ok := m["pickup_datetime"].(string); ok {
			req.PickupDatetime = v
		}
		if v, ok := m["dropoff_datetime"].(string); ok {
			req.DropoffDatetime = v
		}
		if v, ok := m["pickup_address"].(string); ok {
			req.PickupAddress = v
		}
		if v, ok := m["pickup_city_id"]; ok {
			req.PickupCityID = strconv.Itoa(toInt(v))
		}
		if v, ok := m["pickup_location"].(string); ok {
			req.PickupLocation = v
		}
		if v, ok := m["quantity"]; ok {
			req.Quantity = toInt(v)
		}
		if v, ok := m["fleet_qty"]; ok {
			req.FleetQty = toInt(v)
		}
		if v, ok := m["price_id"].(string); ok {
			req.PriceID = v
		}
		if v, ok := m["price"]; ok {
			req.Price = float64(toInt(v))
		}
		if v, ok := m["discount_amount"]; ok {
			req.DiscountAmount = float64(toInt(v))
		}
		if v, ok := m["additional_request"].(string); ok {
			req.AdditionalRequest = v
		}
		if v, ok := m["addons"]; ok {
			if arr, ok := v.([]interface{}); ok {
				addons := make([]model.FleetOrderAddonItem, 0, len(arr))
				for _, rawItem := range arr {
					mm, ok := rawItem.(map[string]interface{})
					if !ok {
						continue
					}
					var it model.FleetOrderAddonItem
					if s, ok := mm["addon_id"].(string); ok {
						it.AddonID = s
					}
					if q, ok := mm["quantity"]; ok {
						it.Quantity = toInt(q)
					}
					if p, ok := mm["addon_price"]; ok {
						it.AddonPrice = float64(toInt(p))
					}
					addons = append(addons, it)
				}
				req.Addons = addons
			}
		}
		if v, ok := m["itinerary"]; ok {
			if arr, ok := v.([]interface{}); ok {
				items := make([]model.FleetOrderItineraryItem, 0, len(arr))
				for _, rawItem := range arr {
					mm, ok := rawItem.(map[string]interface{})
					if !ok {
						continue
					}
					var it model.FleetOrderItineraryItem
					if d, ok := mm["day"]; ok {
						it.Day = toInt(d)
					}
					if s, ok := mm["city_id"]; ok {
						switch vv := s.(type) {
						case string:
							it.CityID = vv
						default:
							it.CityID = strconv.Itoa(toInt(vv))
						}
					}
					if s, ok := mm["destination"].(string); ok {
						it.Destination = s
					}
					items = append(items, it)
				}
				req.Itinerary = items
			}
		}
	}

	orgID, _ := c.Locals("organization_id").(string)
	userID, _ := c.Locals("user_id").(string)
	if orgID == "" || userID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	orderID, err := h.service.CreatePartnerOrder(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Order created", fiber.Map{
		"order_id": orderID,
	})
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
