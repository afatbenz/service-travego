package service

import (
	"encoding/json"
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

	scheduleID, createErr := s.repo.CreateSchedule(model.ScheduleCreateRepositoryInput{
		OrganizationID: input.OrganizationID,
		UserID:         input.UserID,
		OrderID:        input.Request.OrderID,
		DepartureTime:  departureStart,
		CreatedAt:      time.Now(),
		Fleets:         fleets,
	})
	if createErr != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, s.internalMessage("failed to create schedule", createErr))
	}

	return scheduleID, nil
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
