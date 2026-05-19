package model

// PaymentRequest adalah payload dari UI untuk membuat pembayaran
type PaymentRequest struct {
	OrderID       string `json:"order_id"`
	OrderType     int64  `json:"order_type"`    // 1 fleet, 2 tour-package
	PaymentType   int    `json:"payment_type"`   // 1 full, 2 partial
	PriceID       string `json:"price_id"`
	PaymentAmount int64  `json:"payment_amount"` // hanya jika payment_type 2
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
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	StatusMessage     string `json:"status_message"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	PaymentType       string `json:"payment_type"`
	OrderID           string `json:"order_id"`
	MerchantID        string `json:"merchant_id"`
	GrossAmount       string `json:"gross_amount"`
	Currency          string `json:"currency"`
}
