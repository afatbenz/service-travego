package handler

import (
	"fmt"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type ScheduleHandler struct {
	service *service.ScheduleService
}

func NewScheduleHandler(s *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{service: s}
}

func (h *ScheduleHandler) Create(c *fiber.Ctx) error {
	var req model.ScheduleCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if validationErrors := helper.ValidateStruct(&req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	departureTime := strings.TrimSpace(req.DepartureTime)
	departureStart := strings.TrimSpace(req.DepartureTime)
	if departureTime == "" && departureStart == "" {
		return helper.BadRequestResponse(c, "departure_time is required")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.BadRequestResponse(c, "missing user context")
	}

	id, err := h.service.CreateSchedule(model.ScheduleCreateServiceInput{
		OrganizationID: orgID,
		UserID:         userID,
		Request:        &req,
	})
	if err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Schedule created", fiber.Map{
		"schedule_id": id,
	})
}

func (h *ScheduleHandler) Update(c *fiber.Ctx) error {
	var req model.ScheduleUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if validationErrors := helper.ValidateStruct(&req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	departureTime := strings.TrimSpace(req.DepartureTime)
	if departureTime == "" {
		return helper.BadRequestResponse(c, "departure_time is required")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.BadRequestResponse(c, "missing user context")
	}

	if err := h.service.UpdateSchedule(model.ScheduleUpdateServiceInput{
		OrganizationID: orgID,
		UserID:         userID,
		Request:        &req,
	}); err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Schedule updated", fiber.Map{
		"schedule_id": req.ScheduleID,
	})
}

func (h *ScheduleHandler) GetFleetSchedule(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	query := model.ScheduleFleetListQuery{
		Period:         strings.TrimSpace(c.Query("period")),
		OrderID:        strings.TrimSpace(c.Query("order_id")),
		FleetID:        strings.TrimSpace(c.Query("fleet_id")),
		UnitID:         strings.TrimSpace(c.Query("unit_id")),
		FleetName:      strings.TrimSpace(c.Query("fleet_name")),
		PlateNumber:    strings.TrimSpace(c.Query("plate_number")),
		VehicleID:      strings.TrimSpace(c.Query("vehicle_id")),
		Engine:         strings.TrimSpace(c.Query("engine")),
		Capacity:       strings.TrimSpace(c.Query("capacity")),
		ProductionYear: strings.TrimSpace(c.Query("production_year")),
	}

	result, err := h.service.GetScheduleFleetList(model.ScheduleFleetListServiceInput{
		OrganizationID: orgID,
		Query:          query,
	})
	if err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Schedule fleets loaded", result)
}

func (h *ScheduleHandler) GetFleetAvailability(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	var req model.ScheduleFleetAvailabilityRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if validationErrors := helper.ValidateStruct(&req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	filter, err := h.buildFleetAvailabilityFilter(req)
	if err != nil {
		return helper.BadRequestResponse(c, err.Error())
	}

	result, getErr := h.service.GetFleetAvailability(model.ScheduleFleetAvailabilityServiceInput{
		OrganizationID: orgID,
		Filter:         filter,
	})
	if getErr != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(getErr), getErr.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet availability loaded", fiber.Map{
		"items": result,
	})
}

func (h *ScheduleHandler) GetScheduleDetail(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	orderID := strings.TrimSpace(c.Params("orderid"))
	if orderID == "" {
		return helper.BadRequestResponse(c, "order_id is required")
	}

	result, err := h.service.GetScheduleDetail(model.ScheduleDetailServiceInput{
		OrganizationID: orgID,
		OrderID:        orderID,
	})
	if err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Schedule detail loaded", result)
}

func (h *ScheduleHandler) buildFleetAvailabilityFilter(req model.ScheduleFleetAvailabilityRequest) (model.ScheduleFleetAvailabilityFilter, error) {
	filter := model.ScheduleFleetAvailabilityFilter{
		StartDate: strings.TrimSpace(req.StartDate),
		EndDate:   strings.TrimSpace(req.EndDate),
	}

	var err error
	filter.VehicleID, err = parseStringSliceField(req.VehicleID, "vehicle_id")
	if err != nil {
		return model.ScheduleFleetAvailabilityFilter{}, err
	}
	filter.FleetName, err = parseStringSliceField(req.FleetName, "fleet_name")
	if err != nil {
		return model.ScheduleFleetAvailabilityFilter{}, err
	}
	filter.PlateNumber, err = parseStringSliceField(req.PlateNumber, "plate_number")
	if err != nil {
		return model.ScheduleFleetAvailabilityFilter{}, err
	}
	filter.FleetType, err = parseStringSliceField(req.FleetType, "fleet_type")
	if err != nil {
		return model.ScheduleFleetAvailabilityFilter{}, err
	}
	filter.Engine, err = parseStringSliceField(req.Engine, "engine")
	if err != nil {
		return model.ScheduleFleetAvailabilityFilter{}, err
	}
	filter.Capacity, err = parseStringSliceField(req.Capacity, "capacity")
	if err != nil {
		return model.ScheduleFleetAvailabilityFilter{}, err
	}
	filter.ProductionYear, err = parseStringSliceField(req.ProductionYear, "production_year")
	if err != nil {
		return model.ScheduleFleetAvailabilityFilter{}, err
	}

	return filter, nil
}

func parseStringSliceField(value interface{}, fieldName string) ([]string, error) {
	if value == nil {
		return nil, nil
	}

	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil, nil
		}
		return []string{trimmed}, nil
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			itemText := strings.TrimSpace(fmt.Sprintf("%v", item))
			if itemText != "" {
				result = append(result, itemText)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("%s must be string or array", fieldName)
	}
}
