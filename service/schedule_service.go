package service

import (
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
	repo *repository.ScheduleRepository
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

	departureStart := strings.TrimSpace(input.Request.DepartureStart)
	if departureStart == "" {
		departureStart = strings.TrimSpace(input.Request.DepartureTime)
	}

	scheduleID, createErr := s.repo.CreateSchedule(model.ScheduleCreateRepositoryInput{
		OrganizationID: input.OrganizationID,
		UserID:         input.UserID,
		OrderID:        input.Request.OrderID,
		DepartureStart: departureStart,
		CreatedAt:      time.Now(),
		Fleets:         fleets,
	})
	if createErr != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, s.internalMessage("failed to create schedule", createErr))
	}

	return scheduleID, nil
}

func (s *ScheduleService) internalMessage(base string, err error) string {
	message := base
	env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if env != "production" && env != "prod" {
		message = fmt.Sprintf("%s: %v", base, err)
	}
	return message
}
