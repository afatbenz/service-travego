package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type TourPackageHandler struct {
	service *service.TourPackageService
}

func NewTourPackageHandler(service *service.TourPackageService) *TourPackageHandler {
	return &TourPackageHandler{
		service: service,
	}
}

func (h *TourPackageHandler) GetTourPackages(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	packages, err := h.service.GetTourPackages(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Tour packages retrieved successfully", packages)
}

func (h *TourPackageHandler) CreateTourPackage(c *fiber.Ctx) error {
	var req model.CreateTourPackageRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] BodyParser failed - Path: %s, Error: %v", c.Path(), err)
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		log.Printf("[ERROR] Organization ID not found in context - Path: %s", c.Path())
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		log.Printf("[ERROR] User ID not found in context - Path: %s", c.Path())
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "User not found")
	}

	if err := h.service.CreateTourPackage(c.Context(), &req, orgID, userID); err != nil {
		log.Printf("[ERROR] CreateTourPackage failed - Path: %s, Error: %v", c.Path(), err)
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusCreated, "Tour package created successfully", nil)
}

func (h *TourPackageHandler) UpdateTourPackage(c *fiber.Ctx) error {
	var req model.UpdateTourPackageRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] BodyParser failed - Path: %s, Error: %v", c.Path(), err)
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 != nil {
			return helper.BadRequestResponse(c, "Invalid request body")
		}

		toInt := func(v interface{}) int {
			switch vv := v.(type) {
			case float64:
				return int(vv)
			case int:
				return vv
			case string:
				n, _ := strconv.Atoi(vv)
				return n
			default:
				return 0
			}
		}
		toStringID := func(v interface{}) string {
			switch vv := v.(type) {
			case string:
				return vv
			case float64:
				return strconv.Itoa(int(vv))
			case int:
				return strconv.Itoa(vv)
			default:
				return ""
			}
		}

		if v, ok := m["package_id"].(string); ok {
			req.PackageID = v
		}
		if v, ok := m["package_name"].(string); ok {
			req.PackageName = v
		}
		if v, ok := m["package_type"]; ok {
			req.PackageType = toStringID(v)
		}
		if v, ok := m["package_description"].(string); ok {
			req.PackageDescription = v
		}
		if v, ok := m["thumbnail"].(string); ok {
			req.Thumbnail = v
		}
		if v, ok := m["active"].(bool); ok {
			req.Active = v
		}

		if arr, ok := m["images"].([]interface{}); ok {
			req.Images = make([]model.TourPackageImageUpsertItem, 0, len(arr))
			for _, it := range arr {
				switch vv := it.(type) {
				case string:
					if vv != "" {
						req.Images = append(req.Images, model.TourPackageImageUpsertItem{ImagePath: vv})
					}
				case map[string]interface{}:
					item := model.TourPackageImageUpsertItem{}
					if s, ok := vv["uuid"].(string); ok {
						item.UUID = s
					}
					if s, ok := vv["image_path"].(string); ok {
						item.ImagePath = s
					}
					if s, ok := vv["path"].(string); ok && item.ImagePath == "" {
						item.ImagePath = s
					}
					if item.ImagePath != "" || item.UUID != "" {
						req.Images = append(req.Images, item)
					}
				}
			}
		}

		if arr, ok := m["facilities"].([]interface{}); ok {
			req.Facilities = make([]model.TourPackageFacilityUpsertItem, 0, len(arr))
			for _, it := range arr {
				switch vv := it.(type) {
				case string:
					if vv != "" {
						req.Facilities = append(req.Facilities, model.TourPackageFacilityUpsertItem{Facility: vv})
					}
				case map[string]interface{}:
					item := model.TourPackageFacilityUpsertItem{}
					if s, ok := vv["uuid"].(string); ok {
						item.UUID = s
					}
					if s, ok := vv["facility"].(string); ok {
						item.Facility = s
					}
					if item.Facility != "" || item.UUID != "" {
						req.Facilities = append(req.Facilities, item)
					}
				}
			}
		}

		if arr, ok := m["addons"].([]interface{}); ok {
			req.Addons = make([]model.TourPackageAddonUpsertItem, 0, len(arr))
			for _, it := range arr {
				if vv, ok := it.(map[string]interface{}); ok {
					item := model.TourPackageAddonUpsertItem{}
					if s, ok := vv["uuid"].(string); ok {
						item.UUID = s
					}
					if s, ok := vv["description"].(string); ok {
						item.Description = s
					}
					if p, ok := vv["price"].(float64); ok {
						item.Price = p
					}
					req.Addons = append(req.Addons, item)
				}
			}
		}

		if arr, ok := m["pricing"].([]interface{}); ok {
			req.Pricing = make([]model.TourPackagePricingUpsertItem, 0, len(arr))
			for _, it := range arr {
				if vv, ok := it.(map[string]interface{}); ok {
					item := model.TourPackagePricingUpsertItem{}
					if s, ok := vv["uuid"].(string); ok {
						item.UUID = s
					}
					if v, ok := vv["min_pax"]; ok {
						item.MinPax = toInt(v)
					}
					if v, ok := vv["max_pax"]; ok {
						item.MaxPax = toInt(v)
					}
					if p, ok := vv["price"].(float64); ok {
						item.Price = p
					}
					req.Pricing = append(req.Pricing, item)
				}
			}
		}

		if arr, ok := m["pickup_areas"].([]interface{}); ok {
			req.PickupAreas = make([]model.TourPackagePickupAreaUpsertItem, 0, len(arr))
			for _, it := range arr {
				switch vv := it.(type) {
				case map[string]interface{}:
					item := model.TourPackagePickupAreaUpsertItem{}
					if s, ok := vv["uuid"].(string); ok {
						item.UUID = s
					}
					if v, ok := vv["id"]; ok {
						item.ID = toStringID(v)
					}
					if item.ID != "" || item.UUID != "" {
						req.PickupAreas = append(req.PickupAreas, item)
					}
				case float64:
					req.PickupAreas = append(req.PickupAreas, model.TourPackagePickupAreaUpsertItem{ID: toStringID(vv)})
				case string:
					if vv != "" {
						req.PickupAreas = append(req.PickupAreas, model.TourPackagePickupAreaUpsertItem{ID: vv})
					}
				}
			}
		}

		if arr, ok := m["itineraries"].([]interface{}); ok {
			req.Itineraries = make([]model.TourPackageItineraryUpsert, 0, len(arr))
			for _, it := range arr {
				dayMap, ok := it.(map[string]interface{})
				if !ok {
					continue
				}
				day := model.TourPackageItineraryUpsert{}
				if v, ok := dayMap["day"]; ok {
					day.Day = toInt(v)
				}
				if acts, ok := dayMap["activities"].([]interface{}); ok {
					day.Activities = make([]model.TourPackageActivityUpsert, 0, len(acts))
					for _, a := range acts {
						actMap, ok := a.(map[string]interface{})
						if !ok {
							continue
						}
						act := model.TourPackageActivityUpsert{}
						if s, ok := actMap["uuid"].(string); ok {
							act.UUID = s
						}
						if s, ok := actMap["time"].(string); ok {
							act.Time = s
						}
						if s, ok := actMap["description"].(string); ok {
							act.Description = s
						}
						if s, ok := actMap["location"].(string); ok {
							act.Location = s
						}
						if city, ok := actMap["city"].(map[string]interface{}); ok {
							if v, ok := city["id"]; ok {
								act.City.ID = toStringID(v)
							}
							if s, ok := city["name"].(string); ok {
								act.City.Name = s
							}
						}
						day.Activities = append(day.Activities, act)
					}
				}
				req.Itineraries = append(req.Itineraries, day)
			}
		}
	}
	if req.PackageID == "" {
		return helper.BadRequestResponse(c, "package_id is required")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "User not found")
	}

	if err := h.service.UpdateTourPackage(c.Context(), &req, orgID, userID); err != nil {
		if err == sql.ErrNoRows {
			return helper.SendErrorResponse(c, fiber.StatusNotFound, "Tour package not found")
		}
		log.Printf("[ERROR] UpdateTourPackage failed - Path: %s, Error: %v", c.Path(), err)
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Tour package updated successfully", nil)
}

func (h *TourPackageHandler) DeleteTourPackage(c *fiber.Ctx) error {
	packageID := c.Params("packageid")
	if packageID == "" {
		return helper.BadRequestResponse(c, "package_id is required")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "User not found")
	}

	if err := h.service.DeleteTourPackage(c.Context(), orgID, userID, packageID); err != nil {
		if err == sql.ErrNoRows {
			return helper.SendErrorResponse(c, fiber.StatusNotFound, "Tour package not found")
		}
		log.Printf("[ERROR] DeleteTourPackage failed - Path: %s, Error: %v", c.Path(), err)
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Tour package deleted successfully", nil)
}

func (h *TourPackageHandler) TourPackageDetail(c *fiber.Ctx) error {
	var req model.TourPackageDetailRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] BodyParser failed - Path: %s, Error: %v", c.Path(), err)
		return helper.BadRequestResponse(c, "Invalid request body")
	}
	if req.PackageID == "" {
		return helper.BadRequestResponse(c, "package_id is required")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	res, err := h.service.GetTourPackageDetail(c.Context(), orgID, req.PackageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return helper.SendErrorResponse(c, fiber.StatusNotFound, "Tour package not found")
		}
		log.Printf("[ERROR] TourPackageDetail failed - Path: %s, Error: %v", c.Path(), err)
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Tour package detail loaded", res)
}
