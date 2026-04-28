package model

import (
	"encoding/json"
	"time"
)

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
	FuelType       string               `json:"fuel_type"`
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
	FleetName      string `json:"fleet_name"`
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
	TotalUnit int    `json:"total_unit"`
	Active    bool   `json:"active"`
	Status    int    `json:"status"`
	Thumbnail string `json:"thumbnail"`
}

type FleetUnitSearchItem struct {
	FleetID   string `json:"fleet_id"`
	FleetName string `json:"fleet_name"`
}

type FleetDetailRequest struct {
	FleetID        string `json:"fleet_id"`
	OrganizationID string `json:"-"`
}

type FleetDeleteRequest struct {
	FleetID        string `json:"fleet_id" validate:"required"`
	OrganizationID string `json:"-"`
	UpdatedBy      string `json:"-"`
}

type FleetFacilityUpsertItem struct {
	UUID     string `json:"uuid"`
	Facility string `json:"facility"`
}

type FleetPickupUpsertItem struct {
	UUID   string `json:"uuid"`
	CityID int    `json:"city_id"`
}

type FleetAddonUpsertItem struct {
	UUID       string `json:"uuid"`
	AddonName  string `json:"addon_name"`
	AddonDesc  string `json:"addon_desc"`
	AddonPrice int    `json:"addon_price"`
}

type FleetPriceUpsertItem struct {
	UUID       string `json:"uuid"`
	Duration   int    `json:"duration"`
	RentType   int    `json:"rent_type"`
	Price      int    `json:"price"`
	DiscAmount int    `json:"disc_amount"`
	DiscPrice  int    `json:"disc_price"`
	Uom        string `json:"uom"`
}

type FleetImageUpsertItem struct {
	UUID     string `json:"uuid"`
	PathFile string `json:"path_file"`
}

// UnmarshalJSON handles both string and object formats for images
func (f *FleetImageUpsertItem) UnmarshalJSON(data []byte) error {
	// Try as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		f.PathFile = str
		return nil
	}

	// Try as object
	type Alias FleetImageUpsertItem
	aux := &struct {
		*Alias
	}{Alias: (*Alias)(f)}
	return json.Unmarshal(data, aux)
}

type UpdateFleetRequest struct {
	FleetID        string                    `json:"fleet_id"`
	FleetName      string                    `json:"fleet_name"`
	FleetType      string                    `json:"fleet_type"`
	Capacity       int                       `json:"capacity"`
	ProductionYear int                       `json:"production_year"`
	Engine         string                    `json:"engine"`
	Body           string                    `json:"body"`
	FuelType       string                    `json:"fuel_type"`
	Description    string                    `json:"description"`
	Active         bool                      `json:"active"`
	Thumbnail      string                    `json:"thumbnail"`
	Facilities     []FleetFacilityUpsertItem `json:"fascilities"`
	Pickup         []FleetPickupUpsertItem   `json:"pickup"`
	Pricing        []FleetPriceUpsertItem    `json:"pricing"`
	Addon          []FleetAddonUpsertItem    `json:"addon"`
	Images         []FleetImageUpsertItem    `json:"images"`
	OrganizationID string                    `json:"-"`
	UpdatedBy      string                    `json:"-"`
}

type FleetDetailMeta struct {
	FleetID        string `json:"fleet_id"`
	FleetType      string `json:"fleet_type"`
	FleetTypeLabel string `json:"fleet_type_label"`
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

type ModuleScheduleInfo struct {
	ScheduleID    string    `json:"schedule_id"`
	OrderID       string    `json:"order_id"`
	DepartureTime time.Time `json:"departure_time"`
	ArrivalTime   time.Time `json:"arrival_time"`
	Status        int       `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	CreatedBy     string    `json:"created_by"`
}
