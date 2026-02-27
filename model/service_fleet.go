package model

import "time"

type ServiceFleetItem struct {
	FleetID        string    `json:"fleet_id"`
	FleetName      string    `json:"fleet_name"`
	FleetType      string    `json:"fleet_type"`
	Capacity       int       `json:"capacity"`
	ProductionYear int       `json:"production_year"`
	Engine         string    `json:"engine"`
	Body           string    `json:"body"`
	Description    string    `json:"description"`
	Thumbnail      string    `json:"thumbnail"`
	OriginalPrice  float64   `json:"original_price"` // Can be null in query if no prices? Query uses MIN(price)
	Uom            string    `json:"uom"`
	CreatedAt      time.Time `json:"created_at"`
	DiscountType   *string   `json:"discount_type"`  // Nullable
	DiscountValue  *float64  `json:"discount_value"` // Nullable
	Price          float64   `json:"price"`          // Calculated
	Duration       int       `json:"duration"`
	Cities         []string  `json:"cities"`
}

type ServiceFleetDetailRequest struct {
	FleetID string `json:"fleet_id"`
}

type ServiceFleetPickupItem struct {
	CityID   int    `json:"city_id"`
	CityName string `json:"city_name"`
}

type ServiceFleetPriceItem struct {
	UUID          string  `json:"uuid"`
	Duration      int     `json:"duration"`
	RentType      int     `json:"rent_type"`
	RentTypeLabel string  `json:"rent_type_label"`
	Price         float64 `json:"price"`
	DiscAmount    float64 `json:"disc_amount"`
	DiscPrice     float64 `json:"disc_price"`
	Uom           string  `json:"uom"`
}

type ServiceFleetAddonItem struct {
	AddonID    string `json:"addon_id"`
	AddonName  string `json:"addon_name"`
	AddonDesc  string `json:"addon_desc"`
	AddonPrice int    `json:"addon_price"`
}

type ServiceFleetDetailResponse struct {
	Meta       FleetDetailMeta          `json:"meta"`
	Facilities []string                 `json:"facilities"`
	Pickup     []ServiceFleetPickupItem `json:"pickup"`
	Addon      []FleetAddonItem         `json:"addon"`
	Pricing    []ServiceFleetPriceItem  `json:"pricing"`
	Images     []FleetImageItem         `json:"images"`
}
