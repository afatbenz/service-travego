package model

import "time"

type InventoryItem struct {
	ItemID         string    `json:"item_id"`
	ItemSKU        string    `json:"item_sku"`
	OrganizationID string    `json:"organization_id"`
	ItemName       string    `json:"item_name"`
	ItemUOM        string    `json:"item_uom"`
	GarageID       string    `json:"garage_id"`
	ItemCategory   int       `json:"item_category"`
	Stock          int       `json:"stock"`
	Status         int       `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      string    `json:"created_by"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by"`
}

type InventoryItemWithLabel struct {
	ItemID            string    `json:"item_id"`
	ItemSKU           string    `json:"item_sku"`
	OrganizationID    string    `json:"organization_id"`
	ItemName          string    `json:"item_name"`
	ItemUOM           string    `json:"item_uom"`
	ItemCategory      int       `json:"item_category"`
	ItemCategoryLabel string    `json:"item_category_label"`
	GarageNames       string    `json:"garage_names"`
	TotalStock        int       `json:"total_stock"`
	Status            int       `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	CreatedBy         string    `json:"created_by"`
	UpdatedAt         time.Time `json:"updated_at"`
	UpdatedBy         string    `json:"updated_by"`
}

type CreateInventoryItemRequest struct {
	ItemID          string  `json:"item_id"`
	ItemSKU         string  `json:"item_sku"`
	ItemName        string  `json:"item_name"`
	ItemUOM         string  `json:"item_uom"`
	ItemCategory    int     `json:"item_category"`
	Stock           int     `json:"stock"`
	ItemPrice       float64 `json:"item_price"`
	GarageID        string  `json:"garage_id"`
	TransactionType string  `json:"transaction_type"`
	TransactionDate string  `json:"transaction_date"`
	SupplierID      string  `json:"supplier_id"`
	SupplierName    string  `json:"supplier_name"`
	SupplierPhone   string  `json:"supplier_phone"`
	SupplierURL     string  `json:"supplier_url"`
	SupplierPrice   float64 `json:"supplier_price"`
	MovementType    int     `json:"movement_type"`
	Notes           string  `json:"notes"`
}

type InventoryMovement struct {
	MovementID   string    `json:"movement_id"`
	ItemID       string    `json:"item_id"`
	GarageID     string    `json:"garage_id"`
	Quantity     int       `json:"quantity"`
	StockBefore  int       `json:"stock_before"`
	StockFinal   int       `json:"stock_final"`
	MovementType int       `json:"movement_type"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	CreatedBy    string    `json:"created_by"`
}

type UpdateInventoryItemRequest struct {
	ItemID       string `json:"item_id"`
	ItemName     string `json:"item_name"`
	ItemUOM      string `json:"item_uom"`
	GarageID     string `json:"garage_id"`
	ItemCategory int    `json:"item_category"`
}

type DeleteInventoryItemRequest struct {
	ItemID string `json:"item_id"`
}

type InventoryRequest struct {
	RequestID      string    `json:"request_id"`
	RequestNumber  string    `json:"request_number"`
	ItemID         string    `json:"item_id"`
	ItemName       string    `json:"item_name"`
	ItemUOM        string    `json:"item_uom"`
	ItemCategory   int       `json:"item_category"`
	GarageID       string    `json:"garage_id"`
	Quantity       int       `json:"quantity"`
	Status         int       `json:"status"`
	OrganizationID string    `json:"organization_id"`
	EmployeeID     string    `json:"employee_id"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      string    `json:"created_by"`
	ApproveAt      time.Time `json:"approve_at"`
	ApproveBy      string    `json:"approve_by"`
	ReceivedAt     time.Time `json:"received_at"`
	ReceivedBy     string    `json:"received_by"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by"`
}

type InventoryRequestWithLabel struct {
	RequestID          string    `json:"request_id"`
	RequestNumber      string    `json:"request_number"`
	ItemID             string    `json:"item_id"`
	ItemCategory       int       `json:"item_category"`
	ItemCategoryLabel  string    `json:"item_category_label"`
	ItemUOM            string    `json:"item_uom"`
	ItemSKU            string    `json:"item_sku"`
	ItemName           string    `json:"item_name"`
	GarageID           string    `json:"garage_id"`
	GarageName         string    `json:"garage_name"`
	GarageCity         string    `json:"garage_city"`
	GarageCityLabel    string    `json:"garage_city_label"`
	Quantity           int       `json:"quantity"`
	Status             int       `json:"status"`
	RequestStatusLabel string    `json:"request_status_label"`
	OrganizationID     string    `json:"organization_id"`
	CreatedAt          time.Time `json:"created_at"`
	CreatedBy          string    `json:"created_by"`
	ApproveAt          time.Time `json:"approve_at"`
	ApproveBy          string    `json:"approve_by"`
	UpdatedAt          time.Time `json:"updated_at"`
	UpdatedBy          string    `json:"updated_by"`
	EmployeeName       string    `json:"employee_name"`
	UnitID             string    `json:"unit_id"`
	VehicleID          string    `json:"vehicle_id"`
	PlateNumber        string    `json:"plate_number"`
	PurchaseID         string    `json:"purchase_id"`
	TransactionDate    string    `json:"transaction_date"`
	OrderStatus        int       `json:"order_status"`
	OrderStatusLabel   string    `json:"order_status_label"`
	ReceivedAt         time.Time `json:"received_at"`
	ReceivedBy         string    `json:"received_by"`
	Stock              int       `json:"stock"`
}

type CreateInventoryRequestRequest struct {
	RequestID    string `json:"request_id"`
	ItemID       string `json:"item_id"`
	ItemName     string `json:"item_name"`
	ItemPhone    string `json:"item_phone"`
	ItemURL      string `json:"item_url"`
	GarageID     string `json:"garage_id"`
	Quantity     int    `json:"quantity"`
	ItemUOM      string `json:"item_uom"`
	EmployeeID   string `json:"employee_id"`
	ItemCategory int    `json:"item_category"`
	UnitID       string `json:"unit_id"`
	Notes        string `json:"notes"`
}

type UpdateInventoryRequestRequest struct {
	RequestID string `json:"request_id"`
	Action    string `json:"action"`
}

type ApproveInventoryRequestRequest struct {
	RequestID string `json:"request_id"`
	ItemID    string `json:"item_id"`
}

type RejectInventoryRequestRequest struct {
	RequestID string `json:"request_id"`
}

type InventoryOrder struct {
	PurchaseID      string    `json:"purchase_id"`
	RequestID       string    `json:"request_id"`
	SupplierID      string    `json:"suplier_id"`
	ItemID          string    `json:"item_id"`
	ItemCategory    int       `json:"item_category"`
	GarageID        string    `json:"garage_id"`
	Quantity        int       `json:"quantity"`
	Amount          float64   `json:"amount"`
	TotalAmount     float64   `json:"total_amount"`
	TransactionDate string    `json:"transaction_date"`
	OrganizationID  string    `json:"organization_id"`
	Status          int       `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedBy       string    `json:"created_by"`
	UpdatedAt       time.Time `json:"updated_at"`
	UpdatedBy       string    `json:"updated_by"`
}

type InventoryHistory struct {
	PurchaseID      string  `json:"purchase_id"`
	RequestID       string  `json:"request_id"`
	SupplierID      string  `json:"suplier_id"`
	ItemID          string  `json:"item_id"`
	ItemCategory    int     `json:"item_category"`
	ItemPrice       float64 `json:"item_price"`
	TotalAmount     float64 `json:"total_amount"`
	GarageID        string  `json:"garage_id"`
	Quantity        int     `json:"quantity"`
	Amount          float64 `json:"amount"`
	TransactionDate string  `json:"transaction_date"`
	CompleteDate    string  `json:"complete_date"`
	ReceivedBy      string  `json:"received_by"`
	ReceivedAt      string  `json:"received_at"`
}

type InventoryOrderWithLabel struct {
	PurchaseID        string    `json:"purchase_id"`
	RequestID         string    `json:"request_id"`
	SupplierID        string    `json:"suplier_id"`
	SupplierName      string    `json:"suplier_name"`
	SupplierCity      int       `json:"suplier_city"`
	SupplierCityLabel string    `json:"suplier_city_label"`
	Quantity          int       `json:"quantity"`
	Amount            float64   `json:"amount"`
	TotalAmount       float64   `json:"total_amount"`
	OrganizationID    string    `json:"organization_id"`
	Status            int       `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	CreatedBy         string    `json:"created_by"`
	UpdatedAt         time.Time `json:"updated_at"`
	UpdatedBy         string    `json:"updated_by"`
}

type InventoryOrderWithDetail struct {
	PurchaseID        string    `json:"purchase_id"`
	RequestID         string    `json:"request_id"`
	SupplierID        string    `json:"suplier_id"`
	SupplierName      string    `json:"suplier_name"`
	SupplierCity      int       `json:"suplier_city"`
	SupplierCityLabel string    `json:"suplier_city_label"`
	ItemID            string    `json:"item_id"`
	ItemSKU           string    `json:"item_sku"`
	ItemUOM           string    `json:"item_uom"`
	ItemName          string    `json:"item_name"`
	GarageID          string    `json:"garage_id"`
	GarageName        string    `json:"garage_name"`
	GarageCity        string    `json:"garage_city"`
	GarageCityLabel   string    `json:"garage_city_label"`
	ItemCategory      int       `json:"item_category"`
	ItemCategoryLabel string    `json:"item_category_label"`
	Quantity          int       `json:"quantity"`
	Amount            float64   `json:"amount"`
	TotalAmount       float64   `json:"total_amount"`
	TransactionDate   string    `json:"transaction_date"`
	OrganizationID    string    `json:"organization_id"`
	Status            int       `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	CreatedBy         string    `json:"created_by"`
	UpdatedAt         time.Time `json:"updated_at"`
	UpdatedBy         string    `json:"updated_by"`
	InvoiceNumber     string    `json:"invoice_number"`
}

type SubmitInventoryOrderRequest struct {
	PurchaseID   string `json:"purchase_id"`
	SupplierName string `json:"suplier_name"`
}

type SubmitRequestOrderRequest struct {
	RequestID     string  `json:"request_id"`
	ItemID        string  `json:"item_id"`
	ItemName      string  `json:"item_name"`
	ItemUOM       string  `json:"item_uom"`
	SupplierID    string  `json:"supplier_id"`
	SupplierName  string  `json:"supplier_name"`
	SupplierPhone string  `json:"supplier_phone"`
	ItemPrice     float64 `json:"item_price"`
	Quantity      int     `json:"quantity"`
}

type ReceiveInventoryOrderRequest struct {
	PurchaseID string `json:"purchase_id"`
}

type Supplier struct {
	SupplierID        string    `json:"suplier_id"`
	SupplierName      string    `json:"suplier_name"`
	SupplierAddress   string    `json:"suplier_address"`
	SupplierCity      int       `json:"suplier_city"`
	SupplierPhone     string    `json:"suplier_phone"`
	SupplierEmail     string    `json:"supliter_email"`
	SupplierURL       string    `json:"suplier_url"`
	SupplierCityLabel string    `json:"suplier_city_label"`
	CreatedAt         time.Time `json:"created_at"`
	CreatedBy         string    `json:"created_by"`
	UpdatedAt         time.Time `json:"updated_at"`
	UpdatedBy         string    `json:"updated_by"`
}

type CreateSupplierRequest struct {
	SupplierName string `json:"suplier_name"`
	Address      string `json:"suplier_address"`
	City         int    `json:"suplier_city"`
	Phone        string `json:"suplier_phone"`
	Email        string `json:"supliter_email"`
}

type DeleteSupplierRequest struct {
	SupplierID string `json:"suplier_id"`
}

type InventoryItemGarage struct {
	ItemGarageID   string    `json:"item_garage_id"`
	ItemID         string    `json:"item_id"`
	GarageID       string    `json:"garage_id"`
	Stock          int       `json:"stock"`
	Status         int       `json:"status"`
	OrganizationID string    `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      string    `json:"created_by"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by"`
}

type InventoryGarageStock struct {
	Stock      int    `json:"stock"`
	GarageName string `json:"garage_name"`
}

type TransferInventoryItemRequest struct {
	ItemID            string `json:"item_id"`
	GarageFrom        string `json:"garage_from"`
	GarageDestination string `json:"garage_destination"`
	Stock             int    `json:"stock"`
}

type InventoryItemLocation struct {
	GarageID        string    `json:"garage_id"`
	GarageName      string    `json:"garage_name"`
	GarageAddress   string    `json:"garage_address"`
	GarageCity      string    `json:"garage_city"`
	GarageCityLabel string    `json:"garage_city_label"`
	Stock           int       `json:"stock"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type InventoryItemDetail struct {
	ItemID       string                  `json:"item_id"`
	ItemSKU      string                  `json:"item_sku"`
	ItemName     string                  `json:"item_name"`
	ItemUOM      string                  `json:"item_uom"`
	ItemCategory int                     `json:"item_category"`
	Status       int                     `json:"status"`
	Locations    []InventoryItemLocation `json:"locations"`
}

type GetItemMovementRequest struct {
	ItemID    string `json:"item_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	GarageID  string `json:"garage_id"`
}

type InventoryItemMovement struct {
	MovementID   string    `json:"movement_id"`
	GarageName   string    `json:"garage_name"`
	MovementType string    `json:"movement_type"`
	Quantity     int       `json:"quantity"`
	StockBefore  int       `json:"stock_before"`
	StockFinal   int       `json:"stock_final"`
	Label        string    `json:"label"`
	ItemUom      string    `json:"item_uom"`
	Notes        string    `json:"notes"`
	MovementDate time.Time `json:"movement_date"`
}

type InventoryTransaction struct {
	TransactionID          string    `json:"transaction_id"`
	InvoiceNumber          string    `json:"invoice_number"`
	Description            string    `json:"description"`
	TransactionType        int       `json:"transaction_type"`
	TransactionItem        string    `json:"transaction_item"`
	TransactionCategory    string    `json:"transaction_category"`
	TransactionDateStr     string    `json:"transaction_date"`
	PaymentType            int       `json:"payment_type"`
	TransactionCategoryInt int       `json:"-"`
	Amount                 float64   `json:"amount"`
	Status                 int       `json:"status"`
	ReferenceID            string    `json:"reference_id"`
	CreatedBy              string    `json:"created_by"`
	CreatedAt              time.Time `json:"created_at"`
}

type GetItemOrderHistoryRequest struct {
	ItemID    string `json:"item_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}
