package model

type CheckoutFleetSummaryRequest struct {
	FleetID string `json:"fleet_id" validate:"required"`
	PriceID string `json:"price_id" validate:"required"`
}

type CheckoutFleetSummaryResponse struct {
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
}
