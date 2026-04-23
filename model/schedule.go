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
	DepartureTime  string
	CreatedAt      time.Time
	Fleets         []ScheduleFleetInsertItem
}

type ScheduleFleetListQuery struct {
	StartDate      string `query:"start_date"`
	EndDate        string `query:"end_date"`
	FleetName      string `query:"fleet_name"`
	PlateNumber    string `query:"plate_number"`
	VehicleID      string `query:"vehicle_id"`
	Engine         string `query:"engine"`
	Capacity       string `query:"capacity"`
	ProductionYear string `query:"production_year"`
}

type ScheduleFleetListServiceInput struct {
	OrganizationID string
	Query          ScheduleFleetListQuery
}

type ScheduleFleetListResponse struct {
	Period    string                  `json:"period"`
	Schedules []ScheduleFleetListItem `json:"schedules"`
}

type ScheduleFleetListItem struct {
	ScheduleID        string                  `json:"schedule_id"`
	StartDate         string                  `json:"start_date"`
	EndDate           string                  `json:"end_date"`
	DepartureTime     string                  `json:"departure_time"`
	ArrivalTime       string                  `json:"arrival_time"`
	ScheduleStatus    int                     `json:"schedule_status"`
	PaymentStatus     int                     `json:"payment_status"`
	UnitQty           int                     `json:"unit_qty"`
	PickupCityID      string                  `json:"pickup_city_id"`
	PickupCityLabel   string                  `json:"pickup_city_label"`
	AdditionalRequest string                  `json:"additional_request"`
	CreatedAt         string                  `json:"created_at"`
	CreatedBy         string                  `json:"created_by"`
	Fleets            []ScheduleFleetListUnit `json:"fleets"`
}

type ScheduleFleetListUnit struct {
	FleetID     string `json:"fleet_id"`
	FleetName   string `json:"fleet_name"`
	VehicleID   string `json:"vehicle_id"`
	PlateNumber string `json:"plate_number"`
	Engine      string `json:"engine"`
	Capacity    int    `json:"capacity"`
}

type ScheduleFleetOrderRow struct {
	ScheduleID        string
	StartDate         time.Time
	EndDate           time.Time
	DepartureTime     string
	ArrivalTime       string
	ScheduleStatus    int
	PaymentStatus     int
	UnitQty           int
	PickupCityID      string
	AdditionalRequest string
	CreatedAt         time.Time
	CreatedBy         string
}
