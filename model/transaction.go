package model

import "time"

type TransactionListRequest struct {
	Month           int    `query:"month"`
	Year            int    `query:"year"`
	NoInvoice       string `query:"no_invoice"`
	Source          int    `query:"source"`
	TransactionType int    `query:"transaction_type"`
}

type TransactionListRow struct {
	TransactionID   string
	OrderType       int
	InvoiceNumber   string
	Description     string
	TransactionDate time.Time
	Status          int
	CreatedAt       time.Time
	CreatedBy       string
}

type TransactionListItem struct {
	TransactionID   string `json:"transaction_id"`
	OrderType       int    `json:"order_type"`
	InvoiceNumber   string `json:"invoice_number"`
	Description     string `json:"description"`
	TransactionDate string `json:"transaction_date"`
	Status          int    `json:"status"`
	CreatedAt       string `json:"created_at"`
	CreatedBy       string `json:"created_by"`
}
