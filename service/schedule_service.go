package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"service-travego/configs"
	"service-travego/model"
	"service-travego/repository"
	"strings"
	"time"
)

type ScheduleService struct {
	repo      *repository.ScheduleRepository
	citiesMap map[string]string
}

func NewScheduleService(repo *repository.ScheduleRepository) *ScheduleService {
	return &ScheduleService{repo: repo}
}

func (s *ScheduleService) CreateSchedule(input model.ScheduleCreateServiceInput) (string, error) {
	paymentStatus, exists, err := s.repo.OrderPaymentStatus(model.ScheduleOrderValidationInput{
		OrganizationID: input.OrganizationID,
		OrderID:        input.Request.OrderID,
	})
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, s.internalMessage("failed to create schedule", err))
	}
	if !exists {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "ORDER_ID_NOT_FOUND")
	}
	if paymentStatus == int(configs.PaymentStatusWaitingPayment) {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "ORDER_UNPAID")
	}
	if paymentStatus == int(configs.PaymentStatusCancelled) {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "ORDER_CANCELLED")
	}

	fleets := make([]model.ScheduleFleetInsertItem, 0, len(input.Request.AssignmentUnits))
	for _, unit := range input.Request.AssignmentUnits {
		itemExists, checkErr := s.repo.OrderItemExists(model.ScheduleOrderItemValidationInput{
			OrganizationID: input.OrganizationID,
			OrderID:        input.Request.OrderID,
			FleetID:        unit.FleetID,
		})
		if checkErr != nil {
			return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, s.internalMessage("failed to create schedule", checkErr))
		}
		if !itemExists {
			return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, fmt.Sprintf("fleet_id %s not found in order", unit.FleetID))
		}

		fleets = append(fleets, model.ScheduleFleetInsertItem{
			FleetID:  unit.FleetID,
			UnitID:   unit.UnitID,
			DriverID: unit.DriverID,
		})
	}

	departureStart := strings.TrimSpace(input.Request.DepartureTime)
	if departureStart == "" {
		departureStart = strings.TrimSpace(input.Request.DepartureTime)
	}

	// Parse departure_time to time.Time for timestamp with time zone
	departureTime, err := time.Parse(time.RFC3339, departureStart)
	if err != nil {
		// Try alternative formats
		if t, err := time.Parse("2006-01-02T15:04", departureStart); err == nil {
			departureTime = t
		} else if t, err := time.Parse("2006-01-02 15:04:05", departureStart); err == nil {
			departureTime = t
		} else if t, err := time.Parse("2006-01-02 15:04", departureStart); err == nil {
			departureTime = t
		} else {
			return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid departure_time format")
		}
	}

	scheduleID, createErr := s.repo.CreateSchedule(model.ScheduleCreateRepositoryInput{
		OrganizationID: input.OrganizationID,
		UserID:         input.UserID,
		OrderID:        input.Request.OrderID,
		DepartureTime:  departureTime,
		CreatedAt:      time.Now(),
		Fleets:         fleets,
	})
	if createErr != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, s.internalMessage("failed to create schedule", createErr))
	}

	return scheduleID, nil
}

func (s *ScheduleService) UpdateSchedule(input model.ScheduleUpdateServiceInput) error {
	if strings.TrimSpace(input.Request.ScheduleID) == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "schedule_id is required")
	}
	if strings.TrimSpace(input.Request.OrderID) == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "order_id is required")
	}

	fleets := make([]model.ScheduleFleetInsertItem, 0, len(input.Request.AssignmentUnits))
	for _, unit := range input.Request.AssignmentUnits {
		itemExists, checkErr := s.repo.OrderItemExists(model.ScheduleOrderItemValidationInput{
			OrganizationID: input.OrganizationID,
			OrderID:        input.Request.OrderID,
			FleetID:        unit.FleetID,
		})
		if checkErr != nil {
			return NewServiceError(ErrInternalServer, http.StatusInternalServerError, s.internalMessage("failed to update schedule", checkErr))
		}
		if !itemExists {
			return NewServiceError(ErrInvalidInput, http.StatusBadRequest, fmt.Sprintf("fleet_id %s not found in order", unit.FleetID))
		}

		fleets = append(fleets, model.ScheduleFleetInsertItem{
			FleetID:  unit.FleetID,
			UnitID:   unit.UnitID,
			DriverID: unit.DriverID,
			CrewID:   unit.CrewID,
		})
	}

	departureStart := strings.TrimSpace(input.Request.DepartureTime)
	departureTime, err := time.Parse(time.RFC3339, departureStart)
	if err != nil {
		if t, err := time.Parse("2006-01-02T15:04", departureStart); err == nil {
			departureTime = t
		} else if t, err := time.Parse("2006-01-02 15:04:05", departureStart); err == nil {
			departureTime = t
		} else if t, err := time.Parse("2006-01-02 15:04", departureStart); err == nil {
			departureTime = t
		} else {
			return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid departure_time format")
		}
	}

	var arrivalTime *time.Time
	arrivalStart := strings.TrimSpace(input.Request.ArrivalTime)
	if arrivalStart != "" {
		at, err := time.Parse(time.RFC3339, arrivalStart)
		if err != nil {
			if t, err := time.Parse("2006-01-02T15:04", arrivalStart); err == nil {
				at = t
			} else if t, err := time.Parse("2006-01-02 15:04:05", arrivalStart); err == nil {
				at = t
			} else if t, err := time.Parse("2006-01-02 15:04", arrivalStart); err == nil {
				at = t
			} else {
				return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid arrival_time format")
			}
		}
		arrivalTime = &at
	}

	updateErr := s.repo.UpdateSchedule(model.ScheduleUpdateRepositoryInput{
		OrganizationID: input.OrganizationID,
		UserID:         input.UserID,
		ScheduleID:     input.Request.ScheduleID,
		OrderID:        input.Request.OrderID,
		DepartureTime:  departureTime,
		ArrivalTime:    arrivalTime,
		UpdatedAt:      time.Now(),
		Fleets:         fleets,
	})
	if updateErr != nil {
		if errors.Is(updateErr, sql.ErrNoRows) {
			return NewServiceError(ErrNotFound, http.StatusNotFound, "SCHEDULE_NOT_FOUND")
		}
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, s.internalMessage("failed to update schedule", updateErr))
	}

	return nil
}

func (s *ScheduleService) GetScheduleFleetList(input model.ScheduleFleetListServiceInput) (*model.ScheduleFleetListResponse, error) {
	periodDate := time.Now()
	if strings.TrimSpace(input.Query.Period) != "" {
		parsedPeriod, err := time.Parse("2006-01", input.Query.Period)
		if err != nil {
			return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "period must be YYYY-MM")
		}
		periodDate = parsedPeriod
	}
	monthStart := time.Date(periodDate.Year(), periodDate.Month(), 1, 0, 0, 0, 0, periodDate.Location())
	monthEnd := monthStart.AddDate(0, 1, -1)

	rows, err := s.repo.ListScheduleFleetOrders(input.Query, input.OrganizationID, monthStart, monthEnd)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, s.internalMessage("failed to get schedule fleets", err))
	}

	s.ensureCitiesLoaded()
	response := &model.ScheduleFleetListResponse{
		Period:    "",
		Schedules: make([]model.ScheduleFleetListItem, 0, len(rows)),
	}

	for _, row := range rows {
		fleets, fleetErr := s.repo.ListScheduleFleets(row.ScheduleID, input.OrganizationID)
		if fleetErr != nil {
			return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, s.internalMessage("failed to get schedule fleets", fleetErr))
		}

		pickupLabel := s.citiesMap[row.PickupCityID]

		for _, fleet := range fleets {
			response.Schedules = append(response.Schedules, model.ScheduleFleetListItem{
				FleetID:         fleet.FleetID,
				FleetName:       fleet.FleetName,
				VehicleID:       fleet.VehicleID,
				PlateNumber:     fleet.PlateNumber,
				Engine:          fleet.Engine,
				Capacity:        fleet.Capacity,
				ScheduleID:      row.ScheduleID,
				OrderID:         row.OrderID,
				StartDate:       row.StartDate.Format("2006-01-02"),
				EndDate:         row.EndDate.Format("2006-01-02"),
				PickupCityLabel: pickupLabel,
			})
		}
	}

	response.Period = monthStart.Format("January 2006")

	return response, nil
}

func (s *ScheduleService) GetFleetAvailability(input model.ScheduleFleetAvailabilityServiceInput) ([]model.ScheduleFleetAvailabilityItem, error) {
	startDate, startErr := time.Parse("2006-01-02", input.Filter.StartDate)
	if startErr != nil {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "start_date must be YYYY-MM-DD")
	}
	endDate, endErr := time.Parse("2006-01-02", input.Filter.EndDate)
	if endErr != nil {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "end_date must be YYYY-MM-DD")
	}
	if endDate.Before(startDate) {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "end_date must be greater than or equal start_date")
	}

	rows, err := s.repo.GetFleetAvailability(input.Filter, input.OrganizationID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, s.internalMessage("failed to get fleet availability", err))
	}

	items := make([]model.ScheduleFleetAvailabilityItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.ScheduleFleetAvailabilityItem{
			ScheduleID:     row.ScheduleID,
			FleetType:      row.FleetType,
			FleetName:      row.FleetName,
			DepartureTime:  row.DepartureTime,
			ArrivalTime:    row.ArrivalTime,
			StartDate:      row.StartDate.Format("2006-01-02"),
			EndDate:        row.EndDate.Format("2006-01-02"),
			VehicleID:      row.VehicleID,
			PlateNumber:    row.PlateNumber,
			Engine:         row.Engine,
			Capacity:       row.Capacity,
			ProductionYear: row.ProductionYear,
			Transmission:   row.Transmission,
		})
	}
	return items, nil
}

func (s *ScheduleService) GetScheduleDetail(input model.ScheduleDetailServiceInput) (*model.ScheduleDetailResponse, error) {
	if strings.TrimSpace(input.OrderID) == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "order_id is required")
	}

	scheduleID, exists, err := s.repo.LatestScheduleIDByOrderID(input.OrganizationID, input.OrderID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, s.internalMessage("failed to get schedule detail", err))
	}
	if !exists || strings.TrimSpace(scheduleID) == "" {
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "SCHEDULE_NOT_FOUND")
	}

	rows, err := s.repo.GetScheduleDetailRows(scheduleID, input.OrganizationID, input.OrderID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, s.internalMessage("failed to get schedule detail", err))
	}
	if len(rows) == 0 {
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "SCHEDULE_NOT_FOUND")
	}

	response := &model.ScheduleDetailResponse{
		ScheduleID:    rows[0].ScheduleID,
		OrderID:       rows[0].OrderID,
		OrderType:     rows[0].OrderType,
		DepartureTime: rows[0].DepartureTime,
		ArrivalTime:   rows[0].ArrivalTime,
		Status:        rows[0].Status,
		Fleets:        make([]model.ScheduleDetailFleetItem, 0),
	}

	seen := map[string]model.ScheduleDetailFleetItem{}
	order := make([]string, 0)
	for _, row := range rows {
		key := strings.TrimSpace(row.UnitID)
		if key == "" {
			key = strings.TrimSpace(row.FleetID)
		}
		if key == "" {
			continue
		}

		existing, ok := seen[key]
		if !ok {
			seen[key] = model.ScheduleDetailFleetItem{
				FleetName:   row.FleetName,
				FleetType:   row.FleetType,
				UnitID:      row.UnitID,
				DriverID:    row.DriverID,
				VehicleID:   row.VehicleID,
				PlateNumber: row.PlateNumber,
			}
			order = append(order, key)
			continue
		}

		if strings.TrimSpace(existing.DriverID) == "" && strings.TrimSpace(row.DriverID) != "" {
			existing.DriverID = row.DriverID
			seen[key] = existing
		}
	}

	for _, key := range order {
		response.Fleets = append(response.Fleets, seen[key])
	}

	return response, nil
}

func (s *ScheduleService) internalMessage(base string, err error) string {
	message := base
	env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if env != "production" && env != "prod" {
		message = fmt.Sprintf("%s: %v", base, err)
	}
	return message
}

func (s *ScheduleService) ensureCitiesLoaded() {
	if s.citiesMap != nil {
		return
	}

	file, err := os.Open("config/location.json")
	if err != nil {
		s.citiesMap = map[string]string{}
		return
	}
	defer file.Close()

	var location model.Location
	if err := json.NewDecoder(file).Decode(&location); err != nil {
		s.citiesMap = map[string]string{}
		return
	}

	mapped := make(map[string]string, len(location.Cities))
	for _, city := range location.Cities {
		mapped[city.ID] = city.Name
	}
	s.citiesMap = mapped
}
