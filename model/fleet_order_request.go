package model

type FleetOrderCreateRequest struct {
	FleetID         string                    `json:"fleet_id"`
	CustomerID      string                    `json:"customer_id"`
	PickupDatetime  string                    `json:"pickup_datetime"`
	DropoffDatetime string                    `json:"dropoff_datetime"`
	PickupAddress   string                    `json:"pickup_address"`
	PickupCityID    string                    `json:"pickup_city_id"`
	PickupLocation  string                    `json:"pickup_location"`
	Quantity        int                       `json:"quantity"`
	PriceID         string                    `json:"price_id"`
	Price           float64                   `json:"price"`
	DpAmount        float64                   `json:"dp_amount"`
	Itinerary       []FleetOrderItineraryItem `json:"itinerary"`
}

type FleetOrderItineraryItem struct {
	Day         int    `json:"day"`
	CityID      string `json:"city_id"`
	Destination string `json:"destination"`
}

