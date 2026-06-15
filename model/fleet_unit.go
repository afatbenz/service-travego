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
	OwnershipType  int    `json:"ownership_type"`
}

type FleetUnitCreateRequest struct {
	VehicleID      string  `json:"vehicle_id" validate:"required"`
	PlateNumber    string  `json:"plate_number" validate:"required"`
	FleetID        string  `json:"fleet_id" validate:"required"`
	Engine         string  `json:"engine" validate:"required"`
	Transmission   string  `json:"transmission" validate:"required"`
	Capacity       int     `json:"capacity" validate:"required"`
	ProductionYear int     `json:"production_year" validate:"required"`
	OwnershipType  *int    `json:"ownership_type"`
	PartnerID      *string `json:"partner_id"`
	PartnerName    *string `json:"partner_name"`
	PartnerPhone   *string `json:"partner_phone"`
	PartnerEmail   *string `json:"partner_email"`

	OrganizationID string `json:"-"`
	CreatedBy      string `json:"-"`
	UnitID         string `json:"-"`
	CreatedDate    time.Time
}

type FleetUnitCreateUnit struct {
	VehicleID      string  `json:"vehicle_id" validate:"required"`
	PlateNumber    string  `json:"plate_number" validate:"required"`
	Engine         string  `json:"engine" validate:"required"`
	Transmission   string  `json:"transmission" validate:"required"`
	Capacity       int     `json:"capacity" validate:"required"`
	ProductionYear int     `json:"production_year" validate:"required"`
	OwnershipType  *int    `json:"ownership_type"`
	PartnerID      *string `json:"partner_id"`
	PartnerName    *string `json:"partner_name"`
	PartnerPhone   *string `json:"partner_phone"`
	PartnerEmail   *string `json:"partner_email"`
}

type FleetUnitBatchCreateRequest struct {
	FleetID string                `json:"fleet_id" validate:"required"`
	Units   []FleetUnitCreateUnit `json:"units" validate:"required,dive"`
}

type FleetUnitUpdateRequest struct {
	UnitID         string  `json:"unit_id" validate:"required"`
	VehicleID      string  `json:"vehicle_id" validate:"required"`
	PlateNumber    string  `json:"plate_number" validate:"required"`
	FleetID        string  `json:"fleet_id" validate:"required"`
	Engine         string  `json:"engine" validate:"required"`
	Transmission   string  `json:"transmission" validate:"required"`
	Capacity       int     `json:"capacity" validate:"required"`
	ProductionYear int     `json:"production_year" validate:"required"`
	OwnershipType  *int    `json:"ownership_type"`
	PartnerID      *string `json:"partner_id"`
	PartnerName    *string `json:"partner_name"`
	PartnerPhone   *string `json:"partner_phone"`
	PartnerPic     *string `json:"partner_pic"`

	OrganizationID string `json:"-"`
	UpdatedBy      string `json:"-"`
	UpdatedDate    time.Time
}

type FleetUnitDetailResponse struct {
	UnitID               string                         `json:"unit_id"`
	VehicleID            string                         `json:"vehicle_id"`
	PlateNumber          string                         `json:"plate_number"`
	FleetID              string                         `json:"fleet_id"`
	FleetName            string                         `json:"fleet_name"`
	FleetType            string                         `json:"fleet_type"`
	Engine               string                         `json:"engine"`
	TransmissionID       string                         `json:"transmission_id"`
	Transmission         string                         `json:"transmission"`
	Capacity             int                            `json:"capacity"`
	ProductionYear       int                            `json:"production_year"`
	Status               int                            `json:"status"`
	Description          string                         `json:"description"`
	Thumbnail            string                         `json:"thumbnail"`
	PickupPoint          []string                       `json:"pickup_point"`
	CreatedBy            string                         `json:"created_by"`
	CreatedDate          string                         `json:"created_date"`
	UpdatedBy            string                         `json:"updated_by"`
	UpdatedDate          string                         `json:"updated_date"`
	OwnershipType        *int                           `json:"ownership_type"`
	OwnershipInformation *FleetUnitOwnershipInformation `json:"ownership_information"`
	PartnerID            string                         `json:"partner_id"`
}

type FleetUnitOwnershipInformation struct {
	PartnerID    string  `json:"partner_id"`
	PartnerName  string  `json:"partner_name"`
	PartnerPhone string  `json:"partner_phone"`
	PartnerEmail *string `json:"partner_email"`
	PartnerPic   string  `json:"partner_pic"`
}

type FleetUnitRevenueRequest struct {
	UnitID string `json:"unit_id"`
	Period string `json:"period"`
}

type FleetUnitRevenue struct {
	Period       string  `json:"period,omitempty"`
	TotalRevenue float64 `json:"total_revenue"`
	TotalBooking int64   `json:"total_booking"`
}

type FleetUnitRevenueHistoryItem struct {
	TransactionDate    string  `json:"transaction_date"`
	OrderID            string  `json:"order_id"`
	PaymentType        int     `json:"payment_type"`
	InvoiceNumber      string  `json:"invoice_number"`
	PaymentMethod      int     `json:"payment_method"`
	Amount             float64 `json:"amount"`
	PaymentTypeLabel   string  `json:"payment_type_label"`
	PaymentMethodLabel string  `json:"payment_method_label"`
}

type FleetUnitRevenueResponse struct {
	Summary []*FleetUnitRevenue           `json:"summary"`
	History []FleetUnitRevenueHistoryItem `json:"history"`
}

type FleetUnitOrderHistoryRequest struct {
	UnitID    string `json:"unit_id" validate:"required"`
	StartDate string `json:"start_date" validate:"required"`
	EndDate   string `json:"end_date" validate:"required"`
}

type FleetUnitOrderHistoryItem struct {
	OrderID         string `json:"order_id"`
	UnitID          string `json:"unit_id"`
	DriverID        string `json:"driver_id"`
	DriverName      string `json:"driver_name"`
	StartDate       string `json:"start_date"`
	EndDate         string `json:"end_date"`
	Status          int    `json:"status"`
	PickupCityID    string `json:"pickup_city_id"`
	PickupCityLabel string `json:"pickup_city_label"`
	Destinations    string `json:"destinations"`
	DestinationIDs  string `json:"destination_ids"`
	DestinationCity string `json:"destination_city"`
}

type FleetUnitScheduleRange struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type FleetUnitExpensesRequest struct {
	UnitID string `json:"unit_id"`
	Period string `json:"period"`
}

type FleetUnitExpenseItem struct {
	TransactionFleetID       string  `json:"transaction_fleet_id"`
	TransactionCategory      string  `json:"transaction_category"`
	TransactionCategoryLabel string  `json:"transaction_category_label"`
	TransactionItem          string  `json:"transaction_item"`
	TransactionItemLabel     string  `json:"transaction_item_label"`
	Description              string  `json:"description"`
	TransactionDate          string  `json:"transaction_date"`
	PaymentType              int     `json:"payment_type"`
	PaymentTypeLabel         string  `json:"payment_type_label"`
	Amount                   float64 `json:"amount"`
}
