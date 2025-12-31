package model

type OrderFleetSummaryRequest struct {
	FleetID string `json:"fleet_id" validate:"required"`
	PriceID string `json:"price_id" validate:"required"`
}

type OrderFleetSummaryResponse struct {
	// Fleet info
	FleetName   string `json:"fleet_name"`
	Capacity    int    `json:"capacity"`
	Engine      string `json:"engine"`
	Body        string `json:"body"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
	Thumbnail   string `json:"thumbnail"`

	// Price info
	Duration int     `json:"duration"`
	RentType int     `json:"rent_type"`
	Price    float64 `json:"price"`
	Uom      string  `json:"uom"`

	Facilities   []string      `json:"facilities"`
	PickupPoints []PickupPoint `json:"pickup_points"`
}

type PickupPoint struct {
	CityID   int    `json:"city_id"`
	CityName string `json:"city_name"`
}
