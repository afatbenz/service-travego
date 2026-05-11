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
	TransactionType int
	TransactionMark int
	TransactionDate time.Time
	Status          int
	Amount          float64
	CreatedAt       time.Time
	CreatedBy       string
}

type TransactionListItem struct {
	TransactionID        string  `json:"transaction_id"`
	OrderType            int     `json:"order_type"`
	InvoiceNumber        string  `json:"invoice_number"`
	Description          string  `json:"description"`
	TransactionType      int     `json:"transaction_type"`
	TransactionTypeLabel string  `json:"transaction_type_label"`
	TransactionMarkLabel string  `json:"transaction_mark_label"`
	TransactionMark      int     `json:"transaction_mark"`
	TransactionDate      string  `json:"transaction_date"`
	Status               int     `json:"status"`
	StatusLabel          string  `json:"status_label"`
	CreatedAt            string  `json:"created_at"`
	CreatedBy            string  `json:"created_by"`
	Amount               float64 `json:"amount"`
}
