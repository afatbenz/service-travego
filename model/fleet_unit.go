package model

import "time"

type FleetUnitListItem struct {
	UnitID         string `json:"unit_id"`
	VehicleID      string `json:"vehicle_id"`
	PlateNumber    string `json:"plate_number"`
	FleetID        string `json:"fleet_id"`
	FleetName      string `json:"fleet_name"`
	Engine         string `json:"engine"`
	Transmission   string `json:"transmission"`
	Capacity       int    `json:"capacity"`
	ProductionYear int    `json:"production_year"`
	CreatedBy      string `json:"created_by"`
	CreatedDate    string `json:"created_date"`
	Status         int    `json:"status"`
}

type FleetUnitCreateRequest struct {
	VehicleID      string `json:"vehicle_id" validate:"required"`
	PlateNumber    string `json:"plate_number" validate:"required"`
	FleetID        string `json:"fleet_id" validate:"required"`
	Engine         string `json:"engine" validate:"required"`
	Transmission   string `json:"transmission" validate:"required"`
	Capacity       int    `json:"capacity" validate:"required"`
	ProductionYear int    `json:"production_year" validate:"required"`

	OrganizationID string `json:"-"`
	CreatedBy      string `json:"-"`
	UnitID         string `json:"-"`
	CreatedDate    time.Time
}

type FleetUnitCreateUnit struct {
	VehicleID      string `json:"vehicle_id" validate:"required"`
	PlateNumber    string `json:"plate_number" validate:"required"`
	Engine         string `json:"engine" validate:"required"`
	Transmission   string `json:"transmission" validate:"required"`
	Capacity       int    `json:"capacity" validate:"required"`
	ProductionYear int    `json:"production_year" validate:"required"`
}

type FleetUnitBatchCreateRequest struct {
	FleetID string                `json:"fleet_id" validate:"required"`
	Units   []FleetUnitCreateUnit `json:"units" validate:"required,dive"`
}

type FleetUnitUpdateRequest struct {
	UnitID         string `json:"unit_id" validate:"required"`
	VehicleID      string `json:"vehicle_id" validate:"required"`
	PlateNumber    string `json:"plate_number" validate:"required"`
	FleetID        string `json:"fleet_id" validate:"required"`
	Engine         string `json:"engine" validate:"required"`
	Transmission   string `json:"transmission" validate:"required"`
	Capacity       int    `json:"capacity" validate:"required"`
	ProductionYear int    `json:"production_year" validate:"required"`

	OrganizationID string `json:"-"`
	UpdatedBy      string `json:"-"`
	UpdatedDate    time.Time
}

type FleetUnitDetailResponse struct {
	UnitID         string   `json:"unit_id"`
	VehicleID      string   `json:"vehicle_id"`
	PlateNumber    string   `json:"plate_number"`
	FleetID        string   `json:"fleet_id"`
	FleetName      string   `json:"fleet_name"`
	FleetType      string   `json:"fleet_type"`
	Engine         string   `json:"engine"`
	TransmissionID string   `json:"transmission_id"`
	Transmission   string   `json:"transmission"`
	Capacity       int      `json:"capacity"`
	ProductionYear int      `json:"production_year"`
	Status         int      `json:"status"`
	Description    string   `json:"description"`
	Thumbnail      string   `json:"thumbnail"`
	PickupPoint    []string `json:"pickup_point"`
	CreatedBy      string   `json:"created_by"`
	CreatedDate    string   `json:"created_date"`
	UpdatedBy      string   `json:"updated_by"`
	UpdatedDate    string   `json:"updated_date"`
}
