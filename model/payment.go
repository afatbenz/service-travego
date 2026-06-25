package model

// PaymentRequest adalah payload dari UI untuk membuat pembayaran
type PaymentRequest struct {
	OrderID        string `json:"order_id"`
	OrderType      int64  `json:"order_type"`   // 1 fleet, 2 tour-package
	PaymentType    int    `json:"payment_type"` // 1 full, 2 partial
	PriceID        string `json:"price_id"`
	PaymentAmount  int64  `json:"payment_amount"`  // hanya jika payment_type 2
	OrganizationID string `json:"organization_id"` // required for tracking
	UserID         string `json:"user_id"`         // required for created_by
}

// PaymentResponse adalah response sukses create payment
type PaymentResponse struct {
	SnapToken string `json:"snap_token"`
	OrderID   string `json:"order_id"`
}

// WebhookResponse adalah response sukses untuk webhook
type WebhookResponse struct {
	Message string `json:"message"`
}

// MidtransWebhookRequest adalah payload dari Midtrans webhook
type MidtransWebhookRequest struct {
	VaNumbers         []VaNumber             `json:"va_numbers"`
	TransactionTime   string                 `json:"transaction_time"`
	TransactionStatus string                 `json:"transaction_status"`
	TransactionID     string                 `json:"transaction_id"`
	StatusMessage     string                 `json:"status_message"`
	StatusCode        string                 `json:"status_code"`
	SignatureKey      string                 `json:"signature_key"`
	SettlementTime    string                 `json:"settlement_time"`
	PaymentType       string                 `json:"payment_type"`
	PaymentAmounts    []interface{}          `json:"payment_amounts"`
	OrderID           string                 `json:"order_id"`
	MerchantID        string                 `json:"merchant_id"`
	GrossAmount       string                 `json:"gross_amount"`
	FraudStatus       string                 `json:"fraud_status"`
	ExpiryTime        string                 `json:"expiry_time"`
	CustomerDetails   map[string]interface{} `json:"customer_details"`
	Currency          string                 `json:"currency"`
	PaymentStatus     string                 `json:"payment_status"`
}

type VaNumber struct {
	VaNumber string `json:"va_number"`
	Bank     string `json:"bank"`
}
