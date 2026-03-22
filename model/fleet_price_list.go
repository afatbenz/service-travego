package model

type FleetPriceListItem struct {
	FleetID       string  `json:"fleet_id"`
	Duration      int     `json:"duration"`
	RentType      int     `json:"rent_type"`
	RentTypeLabel string  `json:"rent_type_label"`
	Price         float64 `json:"price"`
}

