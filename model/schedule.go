package model

import "time"

type ScheduleAssignmentUnitRequest struct {
	FleetID  string   `json:"fleet_id" validate:"required"`
	UnitID   string   `json:"unit_id" validate:"required"`
	DriverID []string `json:"driver_id" validate:"required,min=1,dive,required"`
	CrewID   []string `json:"crew_id" validate:"omitempty,min=1,dive,required"`
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

type ScheduleUpdateRequest struct {
	ScheduleID      string                          `json:"schedule_id" validate:"required"`
	OrderID         string                          `json:"order_id" validate:"required"`
	DepartureTime   string                          `json:"departure_time"`
	ArrivalTime     string                          `json:"arrival_time"`
	AssignmentUnits []ScheduleAssignmentUnitRequest `json:"assignment_units" validate:"required,min=1,dive"`
}

type ScheduleUpdateServiceInput struct {
	OrganizationID string
	UserID         string
	Request        *ScheduleUpdateRequest
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
	CrewID   []string
}

type ScheduleCreateRepositoryInput struct {
	OrganizationID string
	UserID         string
	OrderID        string
	DepartureTime  time.Time
	CreatedAt      time.Time
	Fleets         []ScheduleFleetInsertItem
}

type ScheduleUpdateRepositoryInput struct {
	OrganizationID string
	UserID         string
	ScheduleID     string
	OrderID        string
	DepartureTime  time.Time
	ArrivalTime    *time.Time
	UpdatedAt      time.Time
	Fleets         []ScheduleFleetInsertItem
}

type ScheduleFleetListQuery struct {
	Period         string `query:"period"`
	OrderID        string `query:"order_id"`
	FleetID        string `query:"fleet_id"`
	UnitID         string `query:"unit_id"`
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
	FleetID         string `json:"fleet_id"`
	FleetName       string `json:"fleet_name"`
	VehicleID       string `json:"vehicle_id"`
	PlateNumber     string `json:"plate_number"`
	Engine          string `json:"engine"`
	Capacity        int    `json:"capacity"`
	ScheduleID      string `json:"schedule_id"`
	OrderID         string `json:"order_id"`
	StartDate       string `json:"start_date"`
	EndDate         string `json:"end_date"`
	PickupCityLabel string `json:"pickup_city_label"`
}

type ScheduleFleetOrderRow struct {
	ScheduleID        string
	OrderID           string
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

type ScheduleFleetAvailabilityRequest struct {
	StartDate      string      `json:"start_date" validate:"required"`
	EndDate        string      `json:"end_date" validate:"required"`
	VehicleID      interface{} `json:"vehicle_id"`
	FleetName      interface{} `json:"fleet_name"`
	PlateNumber    interface{} `json:"plate_number"`
	FleetType      interface{} `json:"fleet_type"`
	Engine         interface{} `json:"engine"`
	Capacity       interface{} `json:"capacity"`
	ProductionYear interface{} `json:"production_year"`
}

type ScheduleFleetAvailabilityFilter struct {
	StartDate      string
	EndDate        string
	VehicleID      []string
	FleetName      []string
	PlateNumber    []string
	FleetType      []string
	Engine         []string
	Capacity       []string
	ProductionYear []string
}

type ScheduleFleetAvailabilityServiceInput struct {
	OrganizationID string
	Filter         ScheduleFleetAvailabilityFilter
}

type ScheduleFleetAvailabilityItem struct {
	ScheduleID     string `json:"schedule_id"`
	FleetType      string `json:"fleet_type"`
	FleetName      string `json:"fleet_name"`
	DepartureTime  string `json:"departure_time"`
	ArrivalTime    string `json:"arrival_time"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	VehicleID      string `json:"vehicle_id"`
	PlateNumber    string `json:"plate_number"`
	Engine         string `json:"engine"`
	Capacity       int    `json:"capacity"`
	ProductionYear int    `json:"production_year"`
	Transmission   string `json:"transmission"`
}

type ScheduleFleetAvailabilityRow struct {
	ScheduleID     string
	FleetType      string
	FleetName      string
	DepartureTime  string
	ArrivalTime    string
	StartDate      time.Time
	EndDate        time.Time
	VehicleID      string
	PlateNumber    string
	Engine         string
	Capacity       int
	ProductionYear int
	Transmission   string
}

type ScheduleDetailServiceInput struct {
	OrganizationID string
	OrderID        string
}

type ScheduleDetailResponse struct {
	ScheduleID    string                    `json:"schedule_id"`
	OrderID       string                    `json:"order_id"`
	OrderType     int                       `json:"order_type"`
	DepartureTime string                    `json:"departure_time"`
	ArrivalTime   string                    `json:"arrival_time"`
	Status        int                       `json:"status"`
	Fleets        []ScheduleDetailFleetItem `json:"fleets"`
}

type ScheduleDetailFleetItem struct {
	FleetName   string `json:"fleet_name"`
	FleetType   string `json:"fleet_type"`
	UnitID      string `json:"unit_id"`
	DriverID    string `json:"driver_id"`
	VehicleID   string `json:"vehicle_id"`
	PlateNumber string `json:"plate_number"`
}

type ScheduleDetailRow struct {
	ScheduleID    string
	OrderID       string
	OrderType     int
	DepartureTime string
	ArrivalTime   string
	Status        int
	FleetID       string
	FleetName     string
	FleetType     string
	UnitID        string
	VehicleID     string
	PlateNumber   string
	DriverID      string
	Fullname      string
	RoleName      string
}
