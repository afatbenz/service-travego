package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type FleetUnitHandler struct {
	service *service.FleetUnitService
}

func NewFleetUnitHandler(s *service.FleetUnitService) *FleetUnitHandler {
	return &FleetUnitHandler{service: s}
}

func (h *FleetUnitHandler) List(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	fleetId := strings.TrimSpace(c.Query("fleet_id"))

	items, err := h.service.List(orgID, fleetId)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet units loaded", items)
}

func (h *FleetUnitHandler) Create(c *fiber.Ctx) error {
	raw := c.Body()
	if len(raw) == 0 {
		return helper.BadRequestResponse(c, "invalid payload")
	}

	var batch model.FleetUnitBatchCreateRequest
	if err := json.Unmarshal(raw, &batch); err == nil && len(batch.Units) > 0 {
		if errs := helper.ValidateStruct(&batch); len(errs) > 0 {
			return helper.SendValidationErrorResponse(c, errs)
		}

		seenVehicle := map[string]struct{}{}
		seenPlate := map[string]struct{}{}
		for _, u := range batch.Units {
			vid := strings.ToUpper(strings.TrimSpace(u.VehicleID))
			if vid != "" {
				if _, ok := seenVehicle[vid]; ok {
					return helper.SendErrorResponse(c, fiber.StatusBadRequest, "DUPLICATE_VEHICLE_ID")
				}
				seenVehicle[vid] = struct{}{}
			}
			pn := strings.ToUpper(strings.TrimSpace(u.PlateNumber))
			if pn != "" {
				if _, ok := seenPlate[pn]; ok {
					return helper.SendErrorResponse(c, fiber.StatusBadRequest, "DUPLICATE_PLATE_NUMBER")
				}
				seenPlate[pn] = struct{}{}
			}
		}

		orgID, ok := c.Locals("organization_id").(string)
		if !ok || orgID == "" {
			return helper.BadRequestResponse(c, "missing organization context")
		}
		userID, ok := c.Locals("user_id").(string)
		if !ok || userID == "" {
			return helper.BadRequestResponse(c, "missing user context")
		}

		ids, err := h.service.CreateBatch(orgID, userID, batch.FleetID, batch.Units)
		if err != nil {
			log.Printf("[ERROR] TransactionID: %s - CreateFleetUnitBatch - Error: %v", helper.GetTransactionID(c), err)
			code := service.GetStatusCode(err)
			return helper.SendErrorResponse(c, code, err.Error())
		}
		return helper.SuccessResponse(c, fiber.StatusOK, "Fleet units created", fiber.Map{
			"unit_ids": ids,
		})
	}

	var req model.FleetUnitCreateRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if errs := helper.ValidateStruct(&req); len(errs) > 0 {
		return helper.SendValidationErrorResponse(c, errs)
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.BadRequestResponse(c, "missing user context")
	}

	id, err := h.service.Create(orgID, userID, &req)
	if err != nil {
		log.Printf("[ERROR] TransactionID: %s - CreateFleetUnit - Error: %v", helper.GetTransactionID(c), err)
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet unit created", fiber.Map{
		"unit_id": id,
	})
}

func (h *FleetUnitHandler) Update(c *fiber.Ctx) error {
	var req model.FleetUnitUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if errs := helper.ValidateStruct(&req); len(errs) > 0 {
		return helper.SendValidationErrorResponse(c, errs)
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.BadRequestResponse(c, "missing user context")
	}

	if err := h.service.Update(orgID, userID, &req); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet unit updated", nil)
}

func (h *FleetUnitHandler) Detail(c *fiber.Ctx) error {
	id := c.Params("unit_id")
	if id == "" {
		return helper.BadRequestResponse(c, "unit_id is required")
	}
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	res, err := h.service.Detail(orgID, id)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	rating, err := h.service.UnitRating(orgID, id)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	reviews, err := h.service.UnitReviews(orgID, id)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	raw, _ := json.Marshal(res)
	var m map[string]interface{}
	_ = json.Unmarshal(raw, &m)
	m["rating"] = rating
	m["reviews"] = reviews

	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet unit detail loaded", m)
}

func (h *FleetUnitHandler) OrderHistory(c *fiber.Ctx) error {
	var req model.FleetUnitOrderHistoryRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if errs := helper.ValidateStruct(&req); len(errs) > 0 {
		return helper.SendValidationErrorResponse(c, errs)
	}
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	items, err := h.service.UnitOrderHistory(orgID, strings.TrimSpace(req.UnitID), strings.TrimSpace(req.StartDate), strings.TrimSpace(req.EndDate))
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	totalSchedules, latestSchedule, upcomingSchedule, err := h.service.UnitScheduleStats(orgID, strings.TrimSpace(req.UnitID))
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet unit order history loaded", fiber.Map{
		"history":           items,
		"total_schedule":    totalSchedules,
		"latest_schedule":   latestSchedule,
		"upcoming_schedule": upcomingSchedule,
	})
}

func formatFleetUnitPeriodIndonesian(t time.Time) string {
	months := [...]string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	mm := ""
	if int(t.Month()) >= 1 && int(t.Month()) <= 12 {
		mm = months[int(t.Month())]
	}
	if mm == "" {
		mm = t.Month().String()
	}
	return fmt.Sprintf("%s %04d", mm, t.Year())
}

func (h *FleetUnitHandler) UnitRevenue(c *fiber.Ctx) error {
	var req model.FleetUnitRevenueRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	if req.Period != "" {
		t, err := time.Parse("2006-01", req.Period)
		if err != nil {
			return helper.BadRequestResponse(c, "Invalid period format. Use YYYY-MM")
		}

		currentStart := t.Format("2006-01-02")
		currentEnd := t.AddDate(0, 1, -1).Format("2006-01-02")

		prevT := t.AddDate(0, -1, 0)
		prevStart := prevT.Format("2006-01-02")
		prevEnd := prevT.AddDate(0, 1, -1).Format("2006-01-02")

		currRev, err := h.service.GetUnitRevenue(orgID, req.UnitID, currentStart, currentEnd)
		if err != nil {
			code := service.GetStatusCode(err)
			return helper.SendErrorResponse(c, code, err.Error())
		}
		currRev.Period = formatFleetUnitPeriodIndonesian(t)

		prevRev, err := h.service.GetUnitRevenue(orgID, req.UnitID, prevStart, prevEnd)
		if err != nil {
			code := service.GetStatusCode(err)
			return helper.SendErrorResponse(c, code, err.Error())
		}
		prevRev.Period = formatFleetUnitPeriodIndonesian(prevT)

		return helper.SuccessResponse(c, fiber.StatusOK, "Fleet unit revenue", []interface{}{currRev, prevRev})
	}

	return helper.BadRequestResponse(c, "period is required")
}
