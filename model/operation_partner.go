package model

import (
	"time"
)

type OperationPartner struct {
	PartnerID      string     `json:"partner_id"`
	PartnerName    string     `json:"partner_name"`
	PartnerAddress *string    `json:"partner_address"`
	PartnerCity    *int       `json:"partner_city"`
	PartnerPhone   string     `json:"partner_phone"`
	PicName        string     `json:"pic_name"`
	CreatedAt      *time.Time `json:"created_at"`
	CreatedBy      *string    `json:"created_by"`
	UpdatedAt      *time.Time `json:"updated_at"`
	UpdatedBy      *string    `json:"updated_by"`
	OrganizationID *string    `json:"organization_id"`
}

type CreateOperationPartnerRequest struct {
	PartnerName    string  `json:"partner_name" validate:"required"`
	PartnerAddress *string `json:"partner_address"`
	PartnerCity    *int    `json:"partner_city"`
	PartnerPhone   string  `json:"partner_phone" validate:"required"`
	PicName        string  `json:"pic_name" validate:"required"`
}

type UpdateOperationPartnerRequest struct {
	PartnerID      string  `json:"partner_id" validate:"required"`
	PartnerName    string  `json:"partner_name" validate:"required"`
	PartnerAddress *string `json:"partner_address"`
	PartnerCity    *int    `json:"partner_city"`
	PartnerPhone   string  `json:"partner_phone" validate:"required"`
	PicName        string  `json:"pic_name" validate:"required"`
}

type OperationPartnerDetailRequest struct {
	PartnerID string `json:"partner_id" validate:"required"`
}

type OperationPartnerDetailResponse struct {
	OperationPartner
	PartnerCityLabel string `json:"partner_city_label"`
}
