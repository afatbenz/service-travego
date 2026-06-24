package model

import "time"

type Package struct {
	PackageID          string   `json:"package_id"`
	PackageName        string   `json:"package_name"`
	PackageDescription string   `json:"package_description"`
	PackageNotes       string   `json:"package_notes"`
	PackagePrice       int      `json:"package_price"`
	OriginalPrice      int      `json:"original_price"`
	PackageDuration    int      `json:"package_duration"`
	Features           []string `json:"features"`
}

type PackageResponse struct {
	PackageID            string   `json:"package_id"`
	PackageName          string   `json:"package_name"`
	PackageDescription   string   `json:"package_description"`
	PackageNotes         string   `json:"package_notes"`
	PackagePrice         int      `json:"package_price"`
	PackageOriginalPrice int      `json:"package_original_price"`
	PackageDuration      int      `json:"package_duration"`
	Features             []string `json:"features"`
}

type Review struct {
	ReviewID    string    `json:"review_id"`
	UserID      string    `json:"user_id"`
	Stars       int       `json:"stars"`
	Review      string    `json:"review"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	CompanyName string    `json:"company_name"`
}
