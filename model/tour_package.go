package model

type TourPackageListItem struct {
	PackageID          string  `json:"package_id"`
	PackageName        string  `json:"package_name"`
	Thumbnail          string  `json:"thumbnail"`
	PackageDescription string  `json:"package_description"`
	Duration           string  `json:"duration"` // User requested duration
	MinPax             int     `json:"min_pax"`
	MaxPax             int     `json:"max_pax"`
	Price              float64 `json:"price"` // mapped from min_price
	MinPrice           float64 `json:"min_price"`
	MaxPrice           float64 `json:"max_price"`
	Status             int     `json:"status"`
}

type TourPackageAddon struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type TourPackageActivityCity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TourPackageActivity struct {
	Time        string                  `json:"time"`
	Description string                  `json:"description"`
	Location    string                  `json:"location"`
	City        TourPackageActivityCity `json:"city"`
}

type TourPackageItinerary struct {
	Day        int                   `json:"day"`
	Activities []TourPackageActivity `json:"activities"`
}

type TourPackagePricing struct {
	MinPax int     `json:"min_pax"`
	MaxPax int     `json:"max_pax"`
	Price  float64 `json:"price"`
}

type TourPackagePickupArea struct {
	ID string `json:"id"`
}

type CreateTourPackageRequest struct {
	PackageName        string                  `json:"package_name" validate:"required"`
	PackageType        string                  `json:"package_type" validate:"required"`
	PackageDescription string                  `json:"package_description"`
	Thumbnail          string                  `json:"thumbnail"`
	Images             []string                `json:"images"`
	Facilities         []string                `json:"facilities"`
	Itineraries        []TourPackageItinerary  `json:"itineraries"`
	Pricing            []TourPackagePricing    `json:"pricing"`
	Addons             []TourPackageAddon      `json:"addons"`
	PickupAreas        []TourPackagePickupArea `json:"pickup_areas"`
	Active             bool                    `json:"active"`
}

type TourPackageDetailRequest struct {
	PackageID string `json:"package_id"`
}

type TourPackageDetailMeta struct {
	PackageID          string `json:"package_id"`
	PackageName        string `json:"package_name"`
	PackageType        int    `json:"package_type"`
	PackageTypeLabel   string `json:"package_type_label"`
	PackageDescription string `json:"package_description"`
	Thumbnail          string `json:"thumbnail"`
	Duration           int    `json:"duration"`
	MinPax             int    `json:"min_pax"`
	MaxPax             int    `json:"max_pax"`
	Active             bool   `json:"active"`
	Status             int    `json:"status"`
}

type TourPackageScheduleItem struct {
	DateStart string `json:"date_start"`
	DateEnd   string `json:"date_end"`
}

type TourPackageDestinationItem struct {
	CityID      int    `json:"city_id"`
	Destination string `json:"destination"`
}

type TourPackageItineraryDetailItem struct {
	Time        string `json:"time"`
	Description string `json:"description"`
	Location    string `json:"location"`
	CityID      int    `json:"city_id"`
	CityName    string `json:"city_name"`
}

type TourPackagePickupAreaItem struct {
	CityID   int    `json:"city_id"`
	CityName string `json:"city_name"`
}

type TourPackageDetailResponse struct {
	Meta         TourPackageDetailMeta            `json:"meta"`
	Schedules    []TourPackageScheduleItem        `json:"schedules"`
	Pricing      []TourPackagePricing             `json:"pricing"`
	PickupAreas  []TourPackagePickupAreaItem      `json:"pickup_areas"`
	Images       []string                         `json:"images"`
	Itineraries  []TourPackageItineraryDetailItem `json:"itineraries"`
	Facilities   []string                         `json:"facilities"`
	Destinations []TourPackageDestinationItem     `json:"destinations"`
	Addons       []TourPackageAddon               `json:"addons"`
}
