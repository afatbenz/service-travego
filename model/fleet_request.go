package model

type FleetPriceRequest struct {
	Duration   int    `json:"duration"`
	RentType   int    `json:"rent_type"`
	Price      int    `json:"price"`
	DiscAmount int    `json:"disc_amount"`
	DiscPrice  int    `json:"disc_price"`
	Uom        string `json:"uom"`
}

type FleetAddonRequest struct {
	AddonName  string `json:"addon_name"`
	AddonDesc  string `json:"addon_desc"`
	AddonPrice int    `json:"addon_price"`
}

type FleetPickupRequest struct {
	CityID int `json:"city_id"`
}

type FleetImageRequest struct {
	PathFile string `json:"path_file"`
}

type CreateFleetRequest struct {
	FleetName      string               `json:"fleet_name"`
	FleetType      string               `json:"fleet_type"`
	Capacity       int                  `json:"capacity"`
	ProductionYear int                  `json:"production_year"`
	Engine         string               `json:"engine"`
	Body           string               `json:"body"`
	Description    string               `json:"description"`
	Active         bool                 `json:"active"`
	Pickup         []FleetPickupRequest `json:"pickup"`
	Facilities     []string             `json:"fascilities"`
	Pricing        []FleetPriceRequest  `json:"pricing"`
	Addon          []FleetAddonRequest  `json:"addon"`
	Thumbnail      string               `json:"thumbnail"`
	Images         []FleetImageRequest  `json:"images"`
	OrganizationID string               `json:"-"`
	CreatedBy      string               `json:"-"`
}

type ListFleetRequest struct {
	FleetType      string `json:"fleet_type"`
	FleetBody      string `json:"fleet_body"`
	FleetEngine    string `json:"fleet_engine"`
	PickupLocation int    `json:"pickup_location"`
	OrganizationID string `json:"-"`
}

type FleetListItem struct {
	FleetID   string `json:"fleet_id"`
	FleetType string `json:"fleet_type"`
	FleetName string `json:"fleet_name"`
	Capacity  int    `json:"capacity"`
	Engine    string `json:"engine"`
	Body      string `json:"body"`
	Active    bool   `json:"active"`
	Status    int    `json:"status"`
	Thumbnail string `json:"thumbnail"`
}

type FleetDetailRequest struct {
	FleetID        string `json:"fleet_id"`
	OrganizationID string `json:"-"`
}

type FleetDetailMeta struct {
	FleetID        string `json:"fleet_id"`
	FleetType      string `json:"fleet_type"`
	FleetName      string `json:"fleet_name"`
	Capacity       int    `json:"capacity"`
	ProductionYear int    `json:"production_year"`
	Engine         string `json:"engine"`
	Body           string `json:"body"`
	FuelType       string `json:"fuel_type"`
	Transmission   string `json:"transmission"`
	Description    string `json:"description"`
	Thumbnail      string `json:"thumbnail"`
	Active         bool   `json:"active"`
	Status         int    `json:"status"`
	CreatedAt      string `json:"created_at"`
	CreatedBy      string `json:"created_by"`
	UpdatedAt      string `json:"updated_at"`
	UpdatedBy      string `json:"updated_by"`
}

type FleetPickupItem struct {
	UUID     string `json:"uuid"`
	CityID   int    `json:"city_id"`
	CityName string `json:"city_name"`
}

type FleetAddonItem struct {
	UUID       string `json:"uuid"`
	AddonName  string `json:"addon_name"`
	AddonDesc  string `json:"addon_desc"`
	AddonPrice int    `json:"addon_price"`
}

type FleetPriceItem struct {
	UUID          string  `json:"uuid"`
	Duration      int     `json:"duration"`
	RentType      int     `json:"rent_type"`
	RentTypeLabel string  `json:"rent_type_label"`
	Price         float64 `json:"price"`
	DiscAmount    float64 `json:"disc_amount"`
	DiscPrice     float64 `json:"disc_price"`
	Uom           string  `json:"uom"`
}

type FleetImageItem struct {
	UUID     string `json:"uuid"`
	PathFile string `json:"path_file"`
}

type FleetDetailResponse struct {
	Meta       FleetDetailMeta   `json:"meta"`
	Facilities []string          `json:"facilities"`
	Pickup     []FleetPickupItem `json:"pickup"`
	Pricing    []FleetPriceItem  `json:"pricing"`
	Addon      []FleetAddonItem  `json:"addon"`
	Images     []FleetImageItem  `json:"images"`
}
