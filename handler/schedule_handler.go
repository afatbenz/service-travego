package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

var scheduleCitiesOnce sync.Once
var scheduleCitiesMap map[string]string

func getScheduleCitiesMap() map[string]string {
	scheduleCitiesOnce.Do(func() {
		scheduleCitiesMap = map[string]string{}
		f, err := os.Open("config/location.json")
		if err != nil {
			return
		}
		defer f.Close()
		var loc model.Location
		if err := json.NewDecoder(f).Decode(&loc); err != nil {
			return
		}
		for _, c := range loc.Cities {
			id := strings.TrimSpace(c.ID)
			if id == "" {
				continue
			}
			scheduleCitiesMap[id] = c.Name
		}
	})
	return scheduleCitiesMap
}

type ScheduleHandler struct {
	service *service.ScheduleService
	db      *sql.DB
	driver  string
}

func NewScheduleHandler(s *service.ScheduleService, db *sql.DB, driver string) *ScheduleHandler {
	return &ScheduleHandler{service: s, db: db, driver: driver}
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

	if err := h.validateOrderScheduleAvailability(c, req.OrderID, req.ScheduleUnits, ""); err != nil {
		return err
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

	currentScheduleID, _ := h.latestScheduleIDByOrderID(orgID, req.OrderID)
	if err := h.validateOrderScheduleAvailability(c, req.OrderID, req.ScheduleUnits, currentScheduleID); err != nil {
		return err
	}

	scheduleID, err := h.service.UpdateSchedule(model.ScheduleUpdateServiceInput{
		OrganizationID: orgID,
		UserID:         userID,
		Request:        &req,
	})
	if err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Schedule updated", fiber.Map{
		"schedule_id": scheduleID,
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

	citiesMap := getScheduleCitiesMap()
	if result != nil && len(result.Schedules) > 0 && len(citiesMap) > 0 {
		for i := range result.Schedules {
			if strings.TrimSpace(result.Schedules[i].Destinations) != "" {
				continue
			}
			raw := strings.TrimSpace(result.Schedules[i].DestinationIDs)
			if raw == "" {
				continue
			}
			parts := strings.Split(raw, ",")
			names := make([]string, 0, len(parts))
			seen := map[string]struct{}{}
			for _, p := range parts {
				id := strings.TrimSpace(p)
				if id == "" {
					continue
				}
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}
				if name := strings.TrimSpace(citiesMap[id]); name != "" {
					names = append(names, name)
				}
			}
			result.Schedules[i].Destinations = strings.Join(names, ", ")
		}
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Schedule fleets loaded", result)
}

func (h *ScheduleHandler) GetFleetTripDetail(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	scheduleNumber := strings.TrimSpace(c.Params("schedule_number"))
	if scheduleNumber == "" {
		return helper.BadRequestResponse(c, "schedule_number is required")
	}

	res, err := h.service.GetFleetTripDetail(model.ScheduleFleetTripDetailServiceInput{
		OrganizationID: orgID,
		ScheduleNumber: scheduleNumber,
	})
	if err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "OK", res)
}

func (h *ScheduleHandler) UpdateFleetTrip(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.BadRequestResponse(c, "missing user context")
	}

	var req model.ScheduleFleetTripUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if validationErrors := helper.ValidateStruct(&req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	scheduleFleetID := strings.TrimSpace(req.ScheduleFleetID)
	driverID := strings.TrimSpace(req.DriverID)
	crewID := strings.TrimSpace(req.CrewID)

	placeholder := func(position int) string {
		if h.driver == "postgres" || h.driver == "pgx" {
			return fmt.Sprintf("$%d", position)
		}
		return "?"
	}

	sfExpr := "sf.uuid = " + placeholder(1)
	orgExpr := "sf.organization_id = " + placeholder(2)
	if h.driver == "postgres" || h.driver == "pgx" {
		sfExpr = "sf.uuid::text = " + placeholder(1)
		orgExpr = "sf.organization_id::text = " + placeholder(2)
	}

	periodQuery := `
		SELECT fo.start_date, fo.end_date
		FROM schedule_fleets sf
		INNER JOIN schedules s ON s.schedule_id = sf.schedule_id AND s.organization_id = sf.organization_id
		INNER JOIN fleet_orders fo ON fo.order_id = s.order_id AND fo.organization_id = s.organization_id
		WHERE ` + sfExpr + ` AND ` + orgExpr + `
		LIMIT 1
	`
	if h.driver == "postgres" || h.driver == "pgx" {
		periodQuery = `
			SELECT fo.start_date, fo.end_date
			FROM schedule_fleets sf
			INNER JOIN schedules s ON s.schedule_id::text = sf.schedule_id::text AND s.organization_id::text = sf.organization_id::text
			INNER JOIN fleet_orders fo ON fo.order_id::text = s.order_id::text AND fo.organization_id::text = s.organization_id::text
			WHERE ` + sfExpr + ` AND ` + orgExpr + `
			LIMIT 1
		`
	}

	var startDate sql.NullTime
	var endDate sql.NullTime
	if err := h.db.QueryRow(periodQuery, scheduleFleetID, orgID).Scan(&startDate, &endDate); err != nil {
		if err == sql.ErrNoRows {
			return helper.SendErrorResponse(c, fiber.StatusNotFound, "SCHEDULE_FLEET_NOT_FOUND")
		}
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to validate schedule")
	}
	if !startDate.Valid || !endDate.Valid {
		return helper.BadRequestResponse(c, "invalid schedule period")
	}

	sftOrgExpr := "sft.organization_id = " + placeholder(1)
	employeeExpr := "(sft.driver_id = " + placeholder(2) + " OR sft.crew_id = " + placeholder(2) + ")"
	excludeExpr := "sft.schedule_fleet_id <> " + placeholder(5)
	if h.driver == "postgres" || h.driver == "pgx" {
		sftOrgExpr = "sft.organization_id::text = " + placeholder(1)
		employeeExpr = "(sft.driver_id::text = " + placeholder(2) + " OR sft.crew_id::text = " + placeholder(2) + ")"
		excludeExpr = "sft.schedule_fleet_id::text <> " + placeholder(5)
	}

	conflictQuery := `
		SELECT 1
		FROM schedule_fleet_teams sft
		INNER JOIN schedules s ON s.schedule_id = sft.schedule_id AND s.organization_id = sft.organization_id
		INNER JOIN fleet_orders fo ON fo.order_id = s.order_id AND fo.organization_id = s.organization_id
		WHERE ` + sftOrgExpr + `
		  AND ` + employeeExpr + `
		  AND COALESCE(sft.status, 0) = 1
		  AND fo.start_date <= ` + placeholder(4) + `
		  AND fo.end_date >= ` + placeholder(3) + `
		  AND ` + excludeExpr + `
		LIMIT 1
	`
	if h.driver == "postgres" || h.driver == "pgx" {
		conflictQuery = `
			SELECT 1
			FROM schedule_fleet_teams sft
			INNER JOIN schedules s ON s.schedule_id::text = sft.schedule_id::text AND s.organization_id::text = sft.organization_id::text
			INNER JOIN fleet_orders fo ON fo.order_id::text = s.order_id::text AND fo.organization_id::text = s.organization_id::text
			WHERE ` + sftOrgExpr + `
			  AND ` + employeeExpr + `
			  AND COALESCE(sft.status, 0) = 1
			  AND fo.start_date <= ` + placeholder(4) + `
			  AND fo.end_date >= ` + placeholder(3) + `
			  AND ` + excludeExpr + `
			LIMIT 1
		`
	}

	checkConflict := func(employeeID string) (bool, error) {
		if strings.TrimSpace(employeeID) == "" {
			return false, nil
		}
		var one int
		err := h.db.QueryRow(conflictQuery, orgID, employeeID, startDate.Time, endDate.Time, scheduleFleetID).Scan(&one)
		if err == nil {
			return true, nil
		}
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	if ok, err := checkConflict(driverID); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to validate driver")
	} else if ok {
		return helper.BadRequestResponse(c, "DRIVER_NOT_AVAILABLE")
	}
	if ok, err := checkConflict(crewID); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to validate crew")
	} else if ok {
		return helper.BadRequestResponse(c, "CREW_NOT_AVAILABLE")
	}

	updateSftExpr := "schedule_fleet_id = " + placeholder(5)
	updateOrgExpr := "organization_id = " + placeholder(6)
	if h.driver == "postgres" || h.driver == "pgx" {
		updateSftExpr = "schedule_fleet_id::text = " + placeholder(5)
		updateOrgExpr = "organization_id::text = " + placeholder(6)
	}

	updateQuery := `
		UPDATE schedule_fleet_teams
		SET driver_id = ` + placeholder(1) + `,
			crew_id = ` + placeholder(2) + `,
			updated_at = ` + placeholder(3) + `,
			updated_by = ` + placeholder(4) + `
		WHERE ` + updateSftExpr + ` AND ` + updateOrgExpr + ` AND COALESCE(status, 0) = 1
	`

	res, err := h.db.Exec(updateQuery, driverID, crewID, time.Now(), userID, scheduleFleetID, orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to update")
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return helper.SendErrorResponse(c, fiber.StatusNotFound, "SCHEDULE_FLEET_TEAM_NOT_FOUND")
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "OK", fiber.Map{
		"schedule_fleet_id": scheduleFleetID,
	})
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

func (h *ScheduleHandler) GetDailyAvailabilityFleet(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	var req model.DailyAvailabilityFleetRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if validationErrors := helper.ValidateStruct(&req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	result, err := h.service.GetDailyAvailabilityFleet(model.DailyAvailabilityFleetServiceInput{
		OrganizationID: orgID,
		FleetID:        strings.TrimSpace(req.FleetID),
		StartDate:      strings.TrimSpace(req.StartDate),
		EndDate:        strings.TrimSpace(req.EndDate),
	})
	if err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "OK", fiber.Map{
		"schedules": result,
	})
}

func (h *ScheduleHandler) GetDailyAvailabilityFleetUnit(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	var req model.DailyAvailabilityFleetUnitRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if validationErrors := helper.ValidateStruct(&req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	result, err := h.service.GetDailyAvailabilityFleetUnit(model.DailyAvailabilityFleetUnitServiceInput{
		OrganizationID: orgID,
		UnitID:         strings.TrimSpace(req.UnitID),
		StartDate:      strings.TrimSpace(req.StartDate),
		EndDate:        strings.TrimSpace(req.EndDate),
	})
	if err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "OK", result)
}

func (h *ScheduleHandler) GetScheduleDetail(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	orderID := strings.TrimSpace(c.Params("order_id"))
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

func (h *ScheduleHandler) GetScheduleDetailByDate(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	date := strings.TrimSpace(c.Query("date"))
	if date == "" {
		return helper.BadRequestResponse(c, "date is required")
	}

	result, err := h.service.GetScheduleDetailByDate(model.ScheduleDetailByDateServiceInput{
		OrganizationID: orgID,
		Date:           date,
	})
	if err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Schedule detail loaded", fiber.Map{
		"items": result,
	})
}

func (h *ScheduleHandler) GetScheduleOperationAvailability(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	startDate := strings.TrimSpace(c.Query("start_date"))
	endDate := strings.TrimSpace(c.Query("end_date"))
	employeeID := strings.TrimSpace(c.Query("employee_id"))
	if startDate == "" {
		return helper.BadRequestResponse(c, "start_date is required")
	}
	if endDate == "" {
		return helper.BadRequestResponse(c, "end_date is required")
	}

	result, err := h.service.GetScheduleOperationAvailability(orgID, startDate, endDate, employeeID)
	if err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Operations availability loaded", fiber.Map{
		"items": result,
	})
}

func (h *ScheduleHandler) GetScheduleFleetUnitAvailability(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	startDate := strings.TrimSpace(c.Query("start_date"))
	endDate := strings.TrimSpace(c.Query("end_date"))
	fleetID := strings.TrimSpace(c.Query("fleet_id"))
	if startDate == "" {
		return helper.BadRequestResponse(c, "start_date is required")
	}
	if endDate == "" {
		return helper.BadRequestResponse(c, "end_date is required")
	}
	if fleetID == "" {
		return helper.BadRequestResponse(c, "fleet_id is required")
	}

	result, err := h.service.GetScheduleFleetUnitAvailability(model.ScheduleFleetUnitAvailabilityServiceInput{
		OrganizationID: orgID,
		StartDate:      startDate,
		EndDate:        endDate,
		FleetID:        fleetID,
	})
	if err != nil {
		return helper.SendErrorResponse(c, service.GetStatusCode(err), err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet units availability loaded", fiber.Map{
		"items": result,
	})
}

func (h *ScheduleHandler) latestScheduleIDByOrderID(orgID, orderID string) (string, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" || strings.TrimSpace(orgID) == "" {
		return "", nil
	}

	placeholder := func(position int) string {
		if h.driver == "postgres" || h.driver == "pgx" {
			return fmt.Sprintf("$%d", position)
		}
		return "?"
	}

	orderExpr := "order_id = " + placeholder(1)
	orgExpr := "organization_id = " + placeholder(2)
	scheduleIDExpr := "schedule_id"
	if h.driver == "postgres" || h.driver == "pgx" {
		orderExpr = "order_id::text = " + placeholder(1)
		orgExpr = "organization_id::text = " + placeholder(2)
		scheduleIDExpr = "schedule_id::text"
	}

	query := "SELECT " + scheduleIDExpr + " FROM schedules WHERE " + orderExpr + " AND " + orgExpr + " ORDER BY created_at DESC LIMIT 1"
	var scheduleID string
	if err := h.db.QueryRow(query, orderID, orgID).Scan(&scheduleID); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(scheduleID), nil
}

func (h *ScheduleHandler) validateOrderScheduleAvailability(c *fiber.Ctx, orderID string, units []model.ScheduleUnitRequest, excludeScheduleID string) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return helper.BadRequestResponse(c, "order_id is required")
	}

	placeholder := func(position int) string {
		if h.driver == "postgres" || h.driver == "pgx" {
			return fmt.Sprintf("$%d", position)
		}
		return "?"
	}

	orderExpr := "order_id = " + placeholder(1)
	orgExpr := "organization_id = " + placeholder(2)
	if h.driver == "postgres" || h.driver == "pgx" {
		orderExpr = "order_id::text = " + placeholder(1)
		orgExpr = "organization_id::text = " + placeholder(2)
	}

	query := "SELECT start_date, end_date FROM fleet_orders WHERE " + orderExpr + " AND " + orgExpr + " LIMIT 1"
	var startDate sql.NullTime
	var endDate sql.NullTime
	if err := h.db.QueryRow(query, orderID, orgID).Scan(&startDate, &endDate); err != nil {
		if err == sql.ErrNoRows {
			return helper.BadRequestResponse(c, "ORDER_ID_NOT_FOUND")
		}
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to validate order")
	}

	if endDate.Valid {
		now := time.Now()
		nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDateOnly := time.Date(endDate.Time.Year(), endDate.Time.Month(), endDate.Time.Day(), 0, 0, 0, 0, endDate.Time.Location())
		if endDateOnly.Before(nowDate) {
			return helper.BadRequestResponse(c, "ORDER_EXPIRED")
		}
	}

	if !startDate.Valid || !endDate.Valid {
		return nil
	}

	scheduleOrgExpr := "sf.organization_id = " + placeholder(1)
	unitExpr := "sf.unit_id = " + placeholder(2)
	scheduleJoinExpr := "s.schedule_id = sf.schedule_id AND s.organization_id = sf.organization_id"
	orderJoinExpr2 := "fo.order_id = s.order_id AND fo.organization_id = s.organization_id"
	excludeExpr := ""
	if strings.TrimSpace(excludeScheduleID) != "" {
		excludeExpr = " AND s.schedule_id <> " + placeholder(5)
	}
	if h.driver == "postgres" || h.driver == "pgx" {
		scheduleOrgExpr = "sf.organization_id::text = " + placeholder(1)
		unitExpr = "sf.unit_id::text = " + placeholder(2)
		scheduleJoinExpr = "s.schedule_id::text = sf.schedule_id::text AND s.organization_id::text = sf.organization_id::text"
		orderJoinExpr2 = "fo.order_id::text = s.order_id::text AND fo.organization_id::text = s.organization_id::text"
		if strings.TrimSpace(excludeScheduleID) != "" {
			excludeExpr = " AND s.schedule_id::text <> " + placeholder(5)
		}
	}

	conflictQuery := `
		SELECT 1
		FROM schedule_fleets sf
		INNER JOIN schedules s ON ` + scheduleJoinExpr + `
		INNER JOIN fleet_orders fo ON ` + orderJoinExpr2 + `
		WHERE ` + scheduleOrgExpr + `
		  AND ` + unitExpr + `
		  AND COALESCE(sf.status, 0) = 1
		  AND fo.start_date <= ` + placeholder(4) + `
		  AND fo.end_date >= ` + placeholder(3) + `
	` + excludeExpr + `
		LIMIT 1
	`

	for _, u := range units {
		unitID := strings.TrimSpace(u.UnitID)
		if unitID == "" {
			continue
		}

		args := []interface{}{orgID, unitID, startDate.Time, endDate.Time}
		if strings.TrimSpace(excludeScheduleID) != "" {
			args = append(args, strings.TrimSpace(excludeScheduleID))
		}

		var one int
		if err := h.db.QueryRow(conflictQuery, args...).Scan(&one); err == nil {
			return helper.BadRequestResponse(c, "UNIT_NOT_AVAILABLE")
		} else if err != sql.ErrNoRows {
			return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to validate availability")
		}
	}

	return nil
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

func (h *ScheduleHandler) GetScheduleTypes(c *fiber.Ctx) error {
	f, err := os.Open("config/common.json")
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load common config")
	}
	defer f.Close()

	var cfg struct {
		ScheduleTypes []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"schedule-types"`
	}
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to parse common config")
	}

	return c.JSON(cfg.ScheduleTypes)
}
