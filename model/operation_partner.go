package model

import (
	"time"
)

type OperationPartner struct {
	PartnerID        string     `json:"partner_id"`
	PartnerName      string     `json:"partner_name"`
	PartnerAddress   *string    `json:"partner_address"`
	PartnerCity      *int       `json:"partner_city"`
	PartnerPhone     string     `json:"partner_phone"`
	PartnerEmail     *string    `json:"partner_email"`
	PicName          string     `json:"pic_name"`
	CreatedAt        *time.Time `json:"created_at"`
	OrganizationID   *string    `json:"organization_id"`
	TotalUnit        int64      `json:"total_unit"`
	PartnerCityLabel string     `json:"partner_city_label"`
	JoinDate         *time.Time `json:"join_date,omitempty"`
	TotalUnits       int64      `json:"total_units,omitempty"`
	TotalSchedule    int64      `json:"total_schedule,omitempty"`
	TotalRevenue     float64    `json:"total_revenue"`
	TotalExpenses    float64    `json:"total_expenses"`
	ProfitEstimate   float64    `json:"profit_estimate"`
	TotalBooking     int64      `json:"total_booking"`
}

type CreateOperationPartnerRequest struct {
	PartnerName    string  `json:"partner_name" validate:"required"`
	PartnerAddress *string `json:"partner_address"`
	PartnerCity    *int    `json:"partner_city"`
	PartnerPhone   string  `json:"partner_phone" validate:"required"`
	PartnerEmail   *string `json:"partner_email"`
	PicName        string  `json:"pic_name" validate:"required"`
}

type UpdateOperationPartnerRequest struct {
	PartnerID      string  `json:"partner_id" validate:"required"`
	PartnerName    string  `json:"partner_name" validate:"required"`
	PartnerAddress *string `json:"partner_address"`
	PartnerCity    *int    `json:"partner_city"`
	PartnerPhone   string  `json:"partner_phone" validate:"required"`
	PartnerEmail   *string `json:"partner_email"`
	PicName        string  `json:"pic_name" validate:"required"`
}

type OperationPartnerDetailRequest struct {
	PartnerID            string `json:"partner_id" validate:"required"`
	TransactionStartDate string `json:"transaction_start_date,omitempty"`
	TransactionEndDate   string `json:"transaction_end_date,omitempty"`
	TripStartDate        string `json:"trip_start_date,omitempty"`
	TripEndDate          string `json:"trip_end_date,omitempty"`
}

type PartnerFleetUnit struct {
	FleetName     string  `json:"fleet_name"`
	FleetType     string  `json:"fleet_type"`
	PlateNumber   string  `json:"plate_number"`
	VehicleID     string  `json:"vehicle_id"`
	UnitID        string  `json:"unit_id"`
	TotalBooking  int64   `json:"total_booking"`
	TotalRevenue  float64 `json:"total_revenue"`
	TotalExpenses float64 `json:"total_expenses"`
}

type OperationPartnerDetailResponse struct {
	OperationPartner
	PartnerCityLabel string             `json:"partner_city_label"`
	FleetUnits       []PartnerFleetUnit `json:"fleet_units"`
}
