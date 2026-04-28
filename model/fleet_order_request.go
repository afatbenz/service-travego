package model

type FleetOrderCreateRequest struct {
	FleetID           string                    `json:"fleet_id"`
	CustomerID        string                    `json:"customer_id"`
	Duration          int                       `json:"duration"`
	RentType          int                       `json:"rent_type"`
	PickupDatetime    string                    `json:"pickup_datetime"`
	DropoffDatetime   string                    `json:"dropoff_datetime"`
	PickupAddress     string                    `json:"pickup_address"`
	PickupCityID      string                    `json:"pickup_city_id"`
	PickupLocation    string                    `json:"pickup_location"`
	Quantity          int                       `json:"quantity"`
	FleetQty          int                       `json:"fleet_qty"`
	PriceID           string                    `json:"price_id"`
	Price             float64                   `json:"price"`
	DiscountAmount    float64                   `json:"discount_amount"`
	AdditionalAmount  float64                   `json:"additional_amount"`
	AdditionalRequest string                    `json:"additional_request"`
	Addons            []FleetOrderAddonItem     `json:"addons"`
	Itinerary         []FleetOrderItineraryItem `json:"itinerary"`
	Fleets            []FleetOrderFleetItem     `json:"fleets"`
}

type FleetOrderFleetItem struct {
	ArmadaID     string  `json:"armada_id"`
	PriceID      string  `json:"price_id"`
	Qty          int     `json:"qty"`
	BiayaLain    float64 `json:"biaya_lain"`
	Discount     float64 `json:"discount"`
}

type FleetOrderAddonItem struct {
	AddonID    string  `json:"addon_id"`
	AddonPrice float64 `json:"addon_price"`
	Quantity   int     `json:"quantity"`
}

type FleetOrderItineraryItem struct {
	Day         int    `json:"day"`
	CityID      string `json:"city_id"`
	CityLabel   string `json:"city_label"`
	Destination string `json:"destination"`
}
