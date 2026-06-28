package model

const (
	ServiceTypeOverland = 1
	ServiceTypeCityTour = 2
	ServiceTypeDropOnly = 3
)

var ServiceTypeLabels = map[int]string{
	ServiceTypeOverland: "overland",
	ServiceTypeCityTour: "city_tour",
	ServiceTypeDropOnly: "drop_only",
}

var ServiceTypeValues = map[string]int{
	"overland":  ServiceTypeOverland,
	"city_tour": ServiceTypeCityTour,
	"drop_only": ServiceTypeDropOnly,
}

type PreferenceCity struct {
	PreferenceID   string `json:"preference_id"`
	CityID         int    `json:"city_id"`
	MinimalDay     int    `json:"minimal_day"`
	OrganizationID string `json:"organization_id"`
	CreatedAt      string `json:"created_at"`
	CreatedBy      string `json:"created_by"`
}

type PreferenceCityWithLabels struct {
	PreferenceID   string   `json:"preference_id"`
	CityID         int      `json:"city_id"`
	MinimalDay     int      `json:"minimal_day"`
	OrganizationID string   `json:"organization_id"`
	CreatedAt      string   `json:"created_at"`
	CreatedBy      string   `json:"created_by"`
	CityLabel      string   `json:"city_label"`
	ProvinceLabel  string   `json:"province_label"`
	ServiceTypes   []string `json:"service_types"`
}

type PreferenceCityType struct {
	PreferenceTypeID string `json:"preference_type_id"`
	CityID           int    `json:"city_id"`
	ServiceType      int    `json:"service_type"`
	OrganizationID   string `json:"organization_id"`
}
