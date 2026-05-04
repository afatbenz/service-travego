package model

import "time"

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
	Active             bool    `json:"active"`
}

type TourPackageOrderListItem struct {
	OrderID      string     `json:"order_id"`
	PackageName  string     `json:"package_name"`
	PackageID    string     `json:"package_id"`
	TotalPax     int        `json:"total_pax"`
	CustomerID   string     `json:"customer_id"`
	CustomerName string     `json:"customer_name"`
	StartDate    *time.Time `json:"start_date"`
	EndDate      *time.Time `json:"end_date"`
}

type TourPackageOrderCreateRequest struct {
	CustomerID       string   `json:"customer_id" validate:"required"`
	StartDate        string   `json:"start_date" validate:"required"`
	EndDate          string   `json:"end_date" validate:"required"`
	MemberPax        int      `json:"member_pax"`
	OfficialPax      int      `json:"official_pax"`
	PickupAddress    string   `json:"pickup_address"`
	PickupCityID     string   `json:"pickup_city_id"`
	DiscountAmount   float64  `json:"discount_amount"`
	AdditionalAmount float64  `json:"additional_amount"`
	PackageID        string   `json:"package_id" validate:"required"`
	PriceID          string   `json:"price_id" validate:"required"`
	Addons           []string `json:"addons"`
}

type TourPackageOrderUpdateRequest struct {
	OrderID          string   `json:"order_id" validate:"required"`
	CustomerID       string   `json:"customer_id" validate:"required"`
	StartDate        string   `json:"start_date" validate:"required"`
	EndDate          string   `json:"end_date" validate:"required"`
	MemberPax        int      `json:"member_pax"`
	OfficialPax      int      `json:"official_pax"`
	PickupAddress    string   `json:"pickup_address"`
	PickupCityID     string   `json:"pickup_city_id"`
	DiscountAmount   float64  `json:"discount_amount"`
	AdditionalAmount float64  `json:"additional_amount"`
	PackageID        string   `json:"package_id" validate:"required"`
	PriceID          string   `json:"price_id" validate:"required"`
	Addons           []string `json:"addons"`
}

type TourPackageAddon struct {
	AddonID     string  `json:"addon_id,omitempty"`
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
	PriceID string  `json:"price_id,omitempty"`
	MinPax  int     `json:"min_pax"`
	MaxPax  int     `json:"max_pax"`
	Price   float64 `json:"price"`
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

type TourPackageAddonUpsertItem struct {
	UUID        string  `json:"uuid"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type TourPackageFacilityUpsertItem struct {
	UUID     string `json:"uuid"`
	Facility string `json:"facility"`
}

type TourPackageActivityUpsert struct {
	UUID        string                  `json:"uuid"`
	Time        string                  `json:"time"`
	Description string                  `json:"description"`
	Location    string                  `json:"location"`
	City        TourPackageActivityCity `json:"city"`
}

type TourPackageItineraryUpsert struct {
	Day        int                         `json:"day"`
	Activities []TourPackageActivityUpsert `json:"activities"`
}

type TourPackagePricingUpsertItem struct {
	UUID   string  `json:"uuid"`
	MinPax int     `json:"min_pax"`
	MaxPax int     `json:"max_pax"`
	Price  float64 `json:"price"`
}

type TourPackagePickupAreaUpsertItem struct {
	UUID string `json:"uuid"`
	ID   string `json:"id"`
}

type TourPackageImageUpsertItem struct {
	UUID      string `json:"uuid"`
	ImagePath string `json:"image_path"`
}

type UpdateTourPackageRequest struct {
	PackageID          string                            `json:"package_id"`
	PackageName        string                            `json:"package_name"`
	PackageType        string                            `json:"package_type"`
	PackageDescription string                            `json:"package_description"`
	Thumbnail          string                            `json:"thumbnail"`
	Images             []TourPackageImageUpsertItem      `json:"images"`
	Facilities         []TourPackageFacilityUpsertItem   `json:"facilities"`
	Itineraries        []TourPackageItineraryUpsert      `json:"itineraries"`
	Pricing            []TourPackagePricingUpsertItem    `json:"pricing"`
	Addons             []TourPackageAddonUpsertItem      `json:"addons"`
	PickupAreas        []TourPackagePickupAreaUpsertItem `json:"pickup_areas"`
	Active             bool                              `json:"active"`
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
