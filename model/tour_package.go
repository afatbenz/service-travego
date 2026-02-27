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
