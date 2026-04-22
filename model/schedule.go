package model

import "time"

type ScheduleAssignmentUnitRequest struct {
	FleetID  string   `json:"fleet_id" validate:"required"`
	UnitID   string   `json:"unit_id" validate:"required"`
	DriverID []string `json:"driver_id" validate:"required,min=1,dive,required"`
}

type ScheduleCreateRequest struct {
	OrderID         string                          `json:"order_id" validate:"required"`
	DepartureTime   string                          `json:"departure_time"`
	DepartureStart  string                          `json:"departure_start"`
	AssignmentUnits []ScheduleAssignmentUnitRequest `json:"assignment_units" validate:"required,min=1,dive"`
}

type ScheduleCreateServiceInput struct {
	OrganizationID string
	UserID         string
	Request        *ScheduleCreateRequest
}

type ScheduleOrderValidationInput struct {
	OrganizationID string
	OrderID        string
}

type ScheduleOrderItemValidationInput struct {
	OrganizationID string
	OrderID        string
	FleetID        string
}

type ScheduleFleetInsertItem struct {
	FleetID  string
	UnitID   string
	DriverID []string
}

type ScheduleCreateRepositoryInput struct {
	OrganizationID string
	UserID         string
	OrderID        string
	DepartureStart string
	CreatedAt      time.Time
	Fleets         []ScheduleFleetInsertItem
}
