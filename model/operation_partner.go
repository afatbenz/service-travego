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
	CreatedBy        *string    `json:"created_by"`
	UpdatedAt        *time.Time `json:"updated_at"`
	UpdatedBy        *string    `json:"updated_by"`
	OrganizationID   *string    `json:"organization_id"`
	TotalUnit        int64      `json:"total_unit"`
	PartnerCityLabel string     `json:"partner_city_label"`
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
	PartnerID string `json:"partner_id" validate:"required"`
}

type PartnerFleetUnit struct {
	FleetName   string `json:"fleet_name"`
	PlateNumber string `json:"plate_number"`
	VehicleID   string `json:"vehicle_id"`
	UnitID      string `json:"unit_id"`
}

type OperationPartnerDetailResponse struct {
	OperationPartner
	PartnerCityLabel string             `json:"partner_city_label"`
	FleetUnits       []PartnerFleetUnit `json:"fleet_units"`
}
