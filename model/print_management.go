package model

type PrintOrderFleetRequest struct {
	OrderID string `json:"order_id"`
}

type PrintFleetInvoiceRequest struct {
	OrderID       string  `json:"order_id"`
	InvoiceNumber *string `json:"invoice_number"`
}
