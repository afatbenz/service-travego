package model

import (
	"database/sql"
	"time"
)

type OrderFleetSummaryRequest struct {
	FleetID string `json:"fleet_id" validate:"required"`
	PriceID string `json:"price_id" validate:"required"`
}

type OrderFleetSummaryResponse struct {
	// Fleet info
	FleetName   string `json:"fleet_name"`
	Capacity    int    `json:"capacity"`
	Engine      string `json:"engine"`
	Body        string `json:"body"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
	Thumbnail   string `json:"thumbnail"`

	// Price info
	Duration      int     `json:"duration"`
	RentType      int     `json:"rent_type"`
	RentTypeLabel string  `json:"rent_type_label"`
	Price         float64 `json:"price"`
	Uom           string  `json:"uom"`

	Facilities   []string      `json:"facilities"`
	PickupPoints []PickupPoint `json:"pickup_points"`
}

type PickupPoint struct {
	CityID   int    `json:"city_id"`
	CityName string `json:"city_name"`
}

type CreateOrderRequest struct {
	FleetID           string             `json:"fleet_id" validate:"required"`
	PriceID           string             `json:"price_id" validate:"required"`
	Fullname          string             `json:"fullname" validate:"required"`
	Email             string             `json:"email" validate:"required,email"`
	Phone             string             `json:"phone" validate:"required"`
	Address           string             `json:"address" validate:"required"`
	StartDate         string             `json:"start_date" validate:"required"`
	EndDate           string             `json:"end_date" validate:"required"`
	PickupCityID      string             `json:"pickup_city_id" validate:"required"`
	PickupLocation    string             `json:"pickup_location" validate:"required"`
	Destinations      []OrderDestination `json:"destinations"`
	Qty               int                `json:"qty" validate:"required,min=1"`
	Addons            []string           `json:"addons"`
	AdditionalAmount  float64            `json:"additional_amount"`
	AdditionalRequest string             `json:"additional_request"`
	CompanyName       string             `json:"company_name"`
	CityID            int                `json:"city_id"`
	OrderType         int                `json:"order_type"`
	OrganizationID    string             `json:"-"`
	OrganizationCode  string             `json:"-"`
	OrderID           string             `json:"-"`
	TotalAmount       float64            `json:"-"`
}

type OrderDestination struct {
	Location string `json:"location"`
	CityID   string `json:"city_id"`
}

type OrderTokenPayload struct {
	OrderID string `json:"order_id"`
	PriceID string `json:"price_id"`
}

type CreateOrderResponse struct {
	Token   string `json:"token"`
	OrderID string `json:"order_id"`
}

type GetOrderListRequest struct {
	Status         int    `query:"status"`
	Page           int    `query:"page"`
	Limit          int    `query:"limit"`
	OrganizationID string `json:"-"`
}

type GetOrderListResponse struct {
	Data        []OrderListItem `json:"data"`
	TotalData   int             `json:"total_data"`
	TotalPage   int             `json:"total_page"`
	CurrentPage int             `json:"current_page"`
}

type OrderListItem struct {
	OrderID       string  `json:"order_id"`
	CreatedAt     string  `json:"created_at"`
	TotalAmount   float64 `json:"total_amount"`
	Status        int     `json:"status"`
	PaymentStatus int     `json:"payment_status"`
	CustomerName  string  `json:"customer_name"`
	CustomerEmail string  `json:"customer_email"`
	CustomerPhone string  `json:"customer_phone"`
}

type OrderDetailResponse struct {
	OrderID           string                    `json:"order_id"`
	FleetID           string                    `json:"fleet_id"`
	OrderDate         string                    `json:"order_date"`
	FleetName         string                    `json:"fleet_name"`
	RentType          int                       `json:"rent_type"`
	RentTypeLabel     string                    `json:"rent_type_label"`
	Duration          string                    `json:"duration"`
	Price             float64                   `json:"price"`
	Quantity          int                       `json:"quantity"`
	TotalAmount       float64                   `json:"total_amount"`
	AdditionalAmount  float64                   `json:"additional_amount"`
	Pickup            OrderDetailPickup         `json:"pickup"`
	Destination       []OrderDetailDest         `json:"destination"`
	Itinerary         []FleetOrderItineraryItem `json:"itinerary"`
	Addon             []OrderDetailAddon        `json:"addon"`
	Customer          OrderDetailCustomer       `json:"customer"`
	Payment           []PaymentDetail           `json:"payment"`
	PaymentStatus     int                       `json:"payment_status"`
	Status            int                       `json:"status"`
	StatusLabel       string                    `json:"status_label"`
	AdditionalRequest string                    `json:"additional_request"`
	Token             string                    `json:"token"`
	PriceID           string                    `json:"-"`
	StartDate         string                    `json:"start_date"`
	EndDate           string                    `json:"end_date"`
	Fleets            []OrderDetailFleetItem    `json:"fleets"`
	Scheduled         bool                      `json:"scheduled"`
}

type OrderReviewItem struct {
	OrderID      string `json:"order_id"`
	Star         int    `json:"star"`
	Review       string `json:"review"`
	CustomerName string `json:"customer_name"`
	CreatedAt    string `json:"created_at"`
}

type OrderRatingSummary struct {
	Rating      float64 `json:"rating"`
	TotalUlasan int64   `json:"total_ulasan"`
}

type PaymentDetail struct {
	BankCode          string        `json:"bank_code"`
	AccountName       string        `json:"account_name"`
	AccountNumber     string        `json:"account_number"`
	BankName          string        `json:"bank_name"`
	PaymentType       int           `json:"payment_type"`
	PaymentPercentage float64       `json:"payment_percentage"`
	PaymentAmount     float64       `json:"payment_amount"`
	TotalAmount       float64       `json:"total_amount"`
	PaymentRemaining  float64       `json:"payment_remaining"`
	Status            PaymentStatus `json:"status"`
	PaymentDate       string        `json:"payment_date"`
	UniqueCode        int           `json:"unique_code"`
}

type PartnerOrderListItem struct {
	OrderID             string                    `json:"order_id"`
	TransactionID       string                    `json:"transaction_id"`
	OrderDate           string                    `json:"order_date"`
	Thumbnail           string                    `json:"thumbnail"`
	FleetName           string                    `json:"fleet_name"`
	CustomerName        string                    `json:"customer_name"`
	CustomerPhone       string                    `json:"customer_phone"`
	StartDate           string                    `json:"start_date"`
	EndDate             string                    `json:"end_date"`
	UnitQty             int                       `json:"unit_qty"`
	PaymentStatus       PaymentStatus             `json:"payment_status"`
	Status              int                       `json:"status"`
	Itinerary           []FleetOrderItineraryItem `json:"itinerary"`
	LatestPaymentStatus string                    `json:"latest_payment_status"`
	Duration            int                       `json:"duration"`
	Uom                 string                    `json:"uom"`
	TotalAmount         float64                   `json:"total_amount"`
	RentType            string                    `json:"rent_type"`
	LatestPaymentType   int                       `json:"-"`
	ScheduleID          string                    `json:"schedule_id"`
	CreatedAt           string                    `json:"created_at"`
}

type PartnerOrderSummary struct {
	TotalOrders int     `json:"total_orders"`
	Paid        int     `json:"paid"`
	Unpaid      int     `json:"unpaid"`
	Pending     int     `json:"pending"`
	Ongoing     int     `json:"ongoing"`
	Revenue     float64 `json:"revenue"`
}

type PartnerOrderListResponse struct {
	Summary PartnerOrderSummary    `json:"summary"`
	Orders  []PartnerOrderListItem `json:"orders"`
}

type PartnerOrderListFilter struct {
	StartDateFrom    string
	StartDateTo      string
	OrderDateFrom    string
	OrderDateTo      string
	Search           string
	PaymentStatus    int
	HasPaymentStatus bool
}

type ServiceOrderListRequest struct {
	OrderType   string `query:"order_type"`
	ProcessType string `query:"process_type"`
}

type ServiceOrderListItem struct {
	OrderID   string `json:"order_id"`
	FleetID   string `json:"fleet_id"`
	StartDate string `json:"start_date"`
}

type OrderDetailCustomer struct {
	CustomerID      string `json:"customer_id"`
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	CustomerEmail   string `json:"customer_email"`
	CustomerAddress string `json:"customer_address"`
	CustomerCity    int    `json:"customer_city"`
	CityLabel       string `json:"city_label"`
}

type OrderDetailPickup struct {
	PickupLocation string `json:"pickup_location"`
	PickupCity     string `json:"pickup_city"`
	CityLabel      string `json:"city_label"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
}

type OrderDetailDest struct {
	City      string `json:"city"`
	ID        string `json:"id"`
	CityLabel string `json:"city_label"`
	Location  string `json:"location"`
}

type OrderDetailAddon struct {
	AddonID     string  `json:"addon_id"`
	AddonName   string  `json:"addon_name"`
	AddonDesc   string  `json:"addon_desc"`
	AddonPrice  float64 `json:"addon_price"`
	OrderItemID string  `json:"order_item_id"`
}

type OrderDetailFleetItem struct {
	OrderItemID  string             `json:"order_item_id"`
	OrderID      string             `json:"order_id"`
	FleetID      string             `json:"fleet_id"`
	FleetName    string             `json:"fleet_name"`
	FleetType    string             `json:"fleet_type"`
	PriceID      string             `json:"price_id"`
	Price        float64            `json:"price"`
	Quantity     int                `json:"quantity"`
	ChargeAmount float64            `json:"charge_amount"`
	AddonAmount  float64            `json:"addon_amount"`
	Discount     float64            `json:"discount"`
	SubTotal     float64            `json:"sub_total"`
	Addons       []OrderDetailAddon `json:"addons"`
}

type CreatePaymentRequest struct {
	Token             string  `json:"token"`
	PaymentMethod     string  `json:"payment_method"`
	PaymentType       int     `json:"payment_type"`
	PaymentPercentage float64 `json:"payment_percentage"`
	OrganizationID    string  `json:"-"`
}

type PaymentStatus int

const (
	PaymentStatusCancelled           PaymentStatus = 0
	PaymentStatusPaid                PaymentStatus = 1
	PaymentStatusPendingVerification PaymentStatus = 2
	PaymentStatusPartialPaid         PaymentStatus = 3
	PaymentStatusWaitingApproval     PaymentStatus = 10
)

type PaymentMethod int

const (
	PaymentMethodBank PaymentMethod = 1
	PaymentMethodQris PaymentMethod = 2
)

type TransactionItem int

const (
	TransactionItemIncome  TransactionItem = 1
	TransactionItemExpense TransactionItem = 2
)

type TransactionType int

const (
	TransactionTypeIncomeRental      TransactionType = 1
	TransactionTypeIncomeTourPackage TransactionType = 2
	TransactionTypeIncomeComission   TransactionType = 3
	TransactionTypeIncomeOtherIncome TransactionType = 4
	TransactionTypeIncomeAds         TransactionType = 5

	TransactionTypeExpenseFuel               TransactionType = 101
	TransactionTypeExpenseTol                TransactionType = 102
	TransactionTypeExpenseDriverAllowance    TransactionType = 103
	TransactionTypeExpenseGuideFee           TransactionType = 104
	TransactionTypeExpenseCrewMeal           TransactionType = 105
	TransactionTypeExpenseVehicleMaintenance TransactionType = 106
	TransactionTypeExpenseVehicleTax         TransactionType = 107
	TransactionTypeExpenseVehicleInsurance   TransactionType = 108
	TransactionTypeExpenseHotel              TransactionType = 109
	TransactionTypeExpenseRestaurant         TransactionType = 110
	TransactionTypeExpenseAttractionTicket   TransactionType = 111
	TransactionTypeExpenseSalary             TransactionType = 112
	TransactionTypeExpenseOfficeRent         TransactionType = 113
	TransactionTypeExpenseUtility            TransactionType = 114
	TransactionTypeExpenseMarketing          TransactionType = 115
	TransactionTypeExpenseBankCharge         TransactionType = 116
	TransactionTypeExpenseOtherExpenses      TransactionType = 117
	TransactionTypeExpenseCommission         TransactionType = 118
)

type FleetOrderPayment struct {
	OrderPaymentID    string        `json:"order_payment_id"`
	OrderID           string        `json:"order_id"`
	OrganizationID    string        `json:"organization_id"`
	PaymentMethod     string        `json:"payment_method"`
	PaymentType       int           `json:"payment_type"`
	PaymentPercentage float64       `json:"payment_percentage"`
	PaymentAmount     float64       `json:"payment_amount"`
	TotalAmount       float64       `json:"total_amount"`
	PaymentRemaining  float64       `json:"payment_remaining"`
	Status            PaymentStatus `json:"status"`
	CreatedAt         time.Time     `json:"created_at"`
	BankCode          string        `json:"bank_code"`
	AccountNumber     string        `json:"account_number"`
	AccountName       string        `json:"account_name"`
	UniqueCode        int           `json:"unique_code"`
}

type OrderPaymentHistory struct {
	PaymentHistoryID string    `json:"payment_history_id"`
	OrderID          string    `json:"order_id"`
	BankCode         string    `json:"bank_code"`
	BankAccountID    string    `json:"bank_account_id"`
	AccountNumber    string    `json:"account_number"`
	AccountName      string    `json:"account_name"`
	CreatedAt        time.Time `json:"created:"`
	OrganizationID   string    `json:"organization_id"`
	PaymentAmount    float64   `json:"payment_amount"`
	UniqueCode       int       `json:"unique_code"`
}

type PaymentMethodResponse struct {
	BankAccountID string `json:"bank_account_id"`
	Icon          string `json:"icon"`
	BankCode      string `json:"bank_code"`
	BankName      string `json:"bank_name"`
}

type PaymentMethodGroupedResponse struct {
	Transfer []PaymentMethodResponse `json:"transfer"`
	Qris     []PaymentMethodResponse `json:"qris"`
}

type PaymentConfirmationRequest struct {
	OrderType      string `json:"order_type"`
	Token          string `json:"token"`
	OrganizationID string `json:"-"`
}

type CreateServiceOrderPaymentRequest struct {
	OrderID             string  `json:"order_id" validate:"required"`
	OrderType           int     `json:"order_type" validate:"required"`
	PaymentType         int     `json:"payment_type" validate:"required"`
	PaymentMethod       int     `json:"payment_method" validate:"required"`
	PaymentAmount       float64 `json:"payment_amount" validate:"required"`
	Type                string  `json:"type" validate:"required"`
	TransactionCategory string  `json:"transaction_category"`
	TransactionItem     string  `json:"transaction_item"`
	EvidenceFile        string  `json:"evidence_file"`
	BankID              *int    `json:"bank_id"`
	BankAccount         *int    `json:"bank_account"`
	OrganizationID      string  `json:"-"`
	CreatedBy           string  `json:"-"`
}

type ServiceOrderPaymentCreateResult struct {
	PaymentID       string  `json:"payment_id"`
	InvoiceNumber   string  `json:"invoice_number"`
	OrderID         string  `json:"order_id"`
	OrderType       int     `json:"order_type"`
	PaymentType     int     `json:"payment_type"`
	PaymentMethod   int     `json:"payment_method"`
	PaymentAmount   float64 `json:"payment_amount"`
	TotalAmount     float64 `json:"total_amount"`
	RemainingAmount float64 `json:"remaining_amount"`
}

type ServiceOrderPaymentStats struct {
	TotalPaid      float64
	DownPaymentCnt int
}

type ServiceOrderPaymentHistoryRequest struct {
	OrderID   string `json:"order_id" validate:"required"`
	OrderType int    `json:"order_type" validate:"required"`
}

type ServiceOrderPaymentHistoryItem struct {
	PaymentID          string  `json:"payment_id"`
	OrderType          int     `json:"order_type"`
	OrderID            string  `json:"order_id"`
	OrganizationID     string  `json:"organization_id"`
	PaymentType        int     `json:"payment_type"`
	PaymentTypeLabel   string  `json:"payment_type_label"`
	PaymentMethod      int     `json:"payment_method"`
	PaymentMethodLabel string  `json:"payment_method_label"`
	BankID             *int    `json:"bank_id"`
	BankAccount        *int    `json:"bank_account"`
	PaymentAmount      float64 `json:"payment_amount"`
	TotalAmount        float64 `json:"total_amount"`
	RemainingAmount    float64 `json:"remaining_amount"`
	EvidenceFile       string  `json:"evidence_file"`
	Status             int     `json:"status"`
	CreatedAt          string  `json:"created_at"`
	CreatedBy          string  `json:"created_by"`
}

type PaymentOrderRow struct {
	PaymentID       string
	OrderType       int
	OrderID         string
	OrganizationID  string
	PaymentType     int
	PaymentMethod   int
	BankID          sql.NullInt64
	BankAccount     sql.NullInt64
	PaymentAmount   float64
	TotalAmount     float64
	RemainingAmount float64
	EvidenceFile    sql.NullString
	Status          int
	CreatedAt       time.Time
	CreatedBy       sql.NullString
}

type FleetOrderDetailByPrefixRequest struct {
	OrderID string `json:"order_id"`
}

type FleetOrderDetailByPrefixResponse struct {
	OrderDetailResponse
	PaymentHistory []FleetOrderPaymentHistoryItem `json:"payment_history"`
}

type FleetOrderPaymentHistoryItem struct {
	PaymentType        int     `json:"payment_type"`
	PaymentTypeLabel   string  `json:"payment_type_label"`
	PaymentMethod      int     `json:"payment_method"`
	PaymentMethodLabel string  `json:"payment_method_label"`
	PaymentAmount      float64 `json:"payment_amount"`
	TotalAmount        float64 `json:"total_amount"`
	RemainingAmount    float64 `json:"remaining_amount"`
	Status             int     `json:"status"`
	CreatedAt          string  `json:"created_at"`
	SettledAt          string  `json:"settled_at"`
	InvoiceNumber      string  `json:"invoice_number"`
	Notes              string  `json:"notes"`
}

type PaymentOrderHistoryRow struct {
	PaymentType     int
	PaymentMethod   int
	PaymentAmount   float64
	TotalAmount     float64
	RemainingAmount float64
	Status          int
	CreatedAt       time.Time
	SettledAt       sql.NullTime
	InvoiceNumber   sql.NullString
	Notes           sql.NullString
}

type PaymentSummary struct {
	PaymentAmount      float64 `json:"payment_amount"`
	PaymentRemaining   float64 `json:"payment_remaining"`
	PaidAmount         float64 `json:"paid_amount"`
	TotalAddon         float64 `json:"total_addon"`
	TotalDiscount      float64 `json:"total_discount"`
	TotalCharge        float64 `json:"total_charge"`
	TotalPayment       float64 `json:"total_payment"`
	PaymentMethod      int     `json:"payment_method"`
	PaymentMethodLabel string  `json:"payment_method_label"`
	PaymentStatus      string  `json:"payment_status"`
	PaymentDate        string  `json:"payment_date"`
}
