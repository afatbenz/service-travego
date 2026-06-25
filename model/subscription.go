package model

import "time"

type GetSubscriptionRequest struct {
	OrganizationID string `json:"organization_id"`
}

type SubscriptionDetail struct {
	PackageID    string    `json:"package_id"`
	PackagePrice float64   `json:"package_price"`
	PackageName  string    `json:"package_name"`
	StartDate    time.Time `json:"start_date"`
	ExpireDate   time.Time `json:"expire_date"`
	Status       string    `json:"status"`
}

type SubscriptionHistory struct {
	SubscriptionDetail
	TransactionID    string    `json:"transaction_id"`
	TransactionDate  time.Time `json:"transaction_date"`
	InvoiceNumber    string    `json:"invoice_number"`
	PaymentMethod    string    `json:"payment_method"`
	PaymentStatus    string    `json:"payment_status"`
	ExpiryDate       time.Time `json:"expiry_date"`
	CreatedAt        time.Time `json:"created_at"`
	CreatedBy        string    `json:"created_by"`
	PaymentAmount    float64   `json:"payment_amount"`
	StartDateFormatted string  `json:"start_date_formatted,omitempty"`
	ExpiryDateFormatted string `json:"expiry_date_formatted,omitempty"`
}

type SubmitSubscriptionResponse struct {
	PackageID           string   `json:"package_id"`
	PackageName         string   `json:"package_name"`
	PackageDuration     int      `json:"package_duration"`
	PackageDescription  string   `json:"package_description"`
	Features            []string `json:"features"`
	PaymentAmount       int      `json:"payment_amount"`
	PackagePrice        int      `json:"package_price"`
	OriginalPrice       int      `json:"original_price"`
	CurrentPackagePrice float64  `json:"current_package_price"`
	DiscountPrice       int      `json:"discount_price"`
}

type SubscriptionDetailByInvoiceResponse struct {
	PackageID       string    `json:"package_id"`
	PackageName     string    `json:"package_name"`
	PackageDuration int       `json:"package_duration"`
	StartDate       time.Time `json:"start_date"`
	ExpiryDate      time.Time `json:"expiry_date"`
	CreatedAt       time.Time `json:"created_at"`
	PaymentMethod   string    `json:"payment_method"`
	PaymentAmount   float64   `json:"payment_amount"`
}
