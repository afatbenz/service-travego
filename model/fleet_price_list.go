package model

type FleetPriceListItem struct {
	PriceID       string  `json:"price_id"`
	FleetID       string  `json:"fleet_id"`
	Duration      int     `json:"duration"`
	RentType      int     `json:"rent_type"`
	RentTypeLabel string  `json:"rent_type_label"`
	Price         float64 `json:"price"`
}
