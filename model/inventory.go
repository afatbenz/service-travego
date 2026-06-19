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
	ItemID       string `json:"item_id"`
	ItemSKU      string `json:"item_sku"`
	ItemName     string `json:"item_name"`
	ItemUOM      string `json:"item_uom"`
	ItemCategory int    `json:"item_category"`
	Stock        int    `json:"stock"`
	GarageID     string `json:"garage_id"`
	MovementType int    `json:"movement_type"`
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
	GarageID       string    `json:"garage_id"`
	Quantity       int       `json:"quantity"`
	Status         int       `json:"status"`
	OrganizationID string    `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      string    `json:"created_by"`
	ApproveAt      time.Time `json:"approve_at"`
	ApproveBy      string    `json:"approve_by"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by"`
}

type InventoryRequestWithLabel struct {
	RequestID       string    `json:"request_id"`
	RequestNumber   string    `json:"request_number"`
	ItemID          string    `json:"item_id"`
	ItemName        string    `json:"item_name"`
	GarageID        string    `json:"garage_id"`
	GarageName      string    `json:"garage_name"`
	GarageCity      string    `json:"garage_city"`
	GarageCityLabel string    `json:"garage_city_label"`
	Quantity        int       `json:"quantity"`
	Status          int       `json:"status"`
	OrganizationID  string    `json:"organization_id"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedBy       string    `json:"created_by"`
	ApproveAt       time.Time `json:"approve_at"`
	ApproveBy       string    `json:"approve_by"`
	UpdatedAt       time.Time `json:"updated_at"`
	UpdatedBy       string    `json:"updated_by"`
}

type CreateInventoryRequestRequest struct {
	ItemName string `json:"item_name"`
	GarageID string `json:"garage_id"`
	Quantity int    `json:"quantity"`
}

type UpdateInventoryRequestRequest struct {
	RequestID string `json:"request_id"`
	Action    string `json:"action"`
}

type InventoryOrder struct {
	PurchaseID     string    `json:"purchase_id"`
	RequestID      string    `json:"request_id"`
	SupplierID     string    `json:"suplier_id"`
	Quantity       int       `json:"quantity"`
	Amount         float64   `json:"amount"`
	TotalAmount    float64   `json:"total_amount"`
	OrganizationID string    `json:"organization_id"`
	Status         int       `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      string    `json:"created_by"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by"`
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
	ItemName          string    `json:"item_name"`
	GarageID          string    `json:"garage_id"`
	GarageName        string    `json:"garage_name"`
	GarageCity        string    `json:"garage_city"`
	GarageCityLabel   string    `json:"garage_city_label"`
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

type SubmitInventoryOrderRequest struct {
	PurchaseID   string `json:"purchase_id"`
	SupplierName string `json:"suplier_name"`
}

type Supplier struct {
	SupplierID        string    `json:"suplier_id"`
	SupplierName      string    `json:"suplier_name"`
	SupplierAddress   string    `json:"suplier_address"`
	SupplierCity      int       `json:"suplier_city"`
	SupplierPhone     string    `json:"suplier_phone"`
	SupplierEmail     string    `json:"supliter_email"`
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
	OrganizationID string    `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      string    `json:"created_by"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by"`
}

type InventoryItemLocation struct {
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
