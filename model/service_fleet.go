package model

import "time"

type ServiceFleetItem struct {
	FleetID        string     `json:"fleet_id"`
	FleetName      string     `json:"fleet_name"`
	FleetType      string     `json:"fleet_type"`
	Capacity       int        `json:"capacity"`
	ProductionYear int        `json:"production_year"`
	Engine         string     `json:"engine"`
	Body           string     `json:"body"`
	Description    string     `json:"description"`
	Thumbnail      string     `json:"thumbnail"`
	OriginalPrice  float64    `json:"original_price"` // Can be null in query if no prices? Query uses MIN(price)
	Uom            string     `json:"uom"`
	CreatedAt      time.Time  `json:"created_at"`
	DiscountType   *string    `json:"discount_type"`  // Nullable
	DiscountValue  *float64   `json:"discount_value"` // Nullable
	Price          float64    `json:"price"`          // Calculated
}
