package model

type FleetPriceRequest struct {
	Duration     int `json:"duration"`
	RentCategory int `json:"rent_category"`
	Price        int `json:"price"`
}

type FleetAddonRequest struct {
	AddonName   string `json:"addon_name"`
	Description string `json:"description"`
	Price       int    `json:"price"`
}

type CreateFleetRequest struct {
	FleetName      string              `json:"fleet_name"`
	FleetType      string              `json:"fleet_type"`
	Capacity       int                 `json:"capacity"`
	ProductionYear int                 `json:"production_year"`
	Engine         string              `json:"engine"`
	Body           string              `json:"body"`
	Description    string              `json:"description"`
	Active         bool                `json:"active"`
	PickupPoint    []int               `json:"pickup_point"`
	Facilities     []string            `json:"fascilities"`
	Prices         []FleetPriceRequest `json:"prices"`
	Addon          []FleetAddonRequest `json:"addon"`
	Thumbnail      string              `json:"thumbnail"`
	BodyImages     []string            `json:"-"`
}
