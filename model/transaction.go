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
	TransactionID            string
	OrderType                int
	InvoiceNumber            string
	Description              string
	TransactionType          int
	TransactionItem          string
	TransactionCategory      string
	TransactionCategoryLabel string
	PaymentType              int
	PaymentTypeLabel         string
	Status                   int
	StatusLabel              string
	UnitID                   string
	TransactionDate          time.Time
	Amount                   float64
	CreatedAt                time.Time
	CreatedBy                string
}

type TransactionListItem struct {
	TransactionID            string  `json:"transaction_id"`
	OrderType                int     `json:"order_type"`
	InvoiceNumber            string  `json:"invoice_number"`
	Description              string  `json:"description"`
	TransactionType          int     `json:"transaction_type"`
	TransactionTypeLabel     string  `json:"transaction_type_label"`
	TransactionItemLabel     string  `json:"transaction_item_label"`
	TransactionItem          string  `json:"transaction_item"`
	TransactionCategory      string  `json:"transaction_category"`
	TransactionCategoryLabel string  `json:"transaction_category_label"`
	TransactionDate          string  `json:"transaction_date"`
	PaymentMethod            int     `json:"payment_method"`
	PaymentMethodLabel       string  `json:"payment_method_label"`
	Status                   int     `json:"status"`
	CreatedAt                string  `json:"created_at"`
	CreatedBy                string  `json:"created_by"`
	Amount                   float64 `json:"amount"`
	PaymentType              int     `json:"payment_type"`
	PaymentTypeLabel         string  `json:"payment_type_label"`
}

type CreateManualRevenueRequest struct {
	Description     string  `json:"description"`
	TransactionDate string  `json:"transaction_date"`
	Status          int     `json:"status"`
	TransactionType int     `json:"transaction_type"`
	Amount          float64 `json:"amount"`
	PaymentMethod   int     `json:"payment_method"`
	BankAccount     string  `json:"bank_account,omitempty"`
	BankCode        string  `json:"bank_code,omitempty"`
	OrderType       int     `json:"order_type"`
	OrderID         string  `json:"order_id,omitempty"`
}

type FleetTripExpenseRow struct {
	TransactionTripID   string
	TransactionCategory string
	TransactionItem     string
	Amount              float64
	PaymentMethod       int
	Description         string
	CreatedAt           time.Time
	CreatedBy           string
}

type FleetTripExpenseItem struct {
	TransactionTripID        string  `json:"transaction_trip_id"`
	TransactionCategory      string  `json:"transaction_category"`
	TransactionCategoryLabel string  `json:"transaction_category_label"`
	TransactionItem          string  `json:"transaction_item"`
	TransactionItemLabel     string  `json:"transaction_item_label"`
	Amount                   float64 `json:"amount"`
	PaymentMethod            int     `json:"payment_method"`
	PaymentMethodLabel       string  `json:"payment_method_label"`
	Description              string  `json:"description"`
	CreatedAt                string  `json:"created_at"`
	CreatedBy                string  `json:"created_by"`
}

type SubmitExpenseTransactionRequest struct {
	Amount              float64 `json:"amount"`
	Description         string  `json:"description"`
	UnitID              string  `json:"unit_id,omitempty"`
	PaymentMethod       int     `json:"payment_method"`
	PaymentType         int     `json:"payment_type"`
	TransactionDate     string  `json:"transaction_date"`
	TransactionCategory string  `json:"transaction_category"`
	TransactionItem     string  `json:"transaction_item"`
}

type DeleteExpenseTransactionRequest struct {
	TransactionID string `json:"transaction_id"`
}

type UpdateExpenseTransactionRequest struct {
	TransactionID       string  `json:"transaction_id"`
	UnitID              string  `json:"unit_id,omitempty"`
	PaymentMethod       int     `json:"payment_method"`
	TransactionDate     string  `json:"transaction_date"`
	TransactionCategory string  `json:"transaction_category"`
	TransactionItem     string  `json:"transaction_item"`
	Amount              float64 `json:"amount"`
}
