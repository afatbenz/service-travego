package model

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
	FleetID          string             `json:"fleet_id" validate:"required"`
	PriceID          string             `json:"price_id" validate:"required"`
	Fullname         string             `json:"fullname" validate:"required"`
	Email            string             `json:"email" validate:"required,email"`
	Phone            string             `json:"phone" validate:"required"`
	Address          string             `json:"address" validate:"required"`
	StartDate        string             `json:"start_date" validate:"required"`
	EndDate          string             `json:"end_date" validate:"required"`
	PickupCityID     string             `json:"pickup_city_id" validate:"required"`
	PickupLocation   string             `json:"pickup_location" validate:"required"`
	Destinations     []OrderDestination `json:"destinations"`
	Qty              int                `json:"qty" validate:"required,min=1"`
	Addons           []string           `json:"addons"`
	OrganizationID   string             `json:"-"`
	OrganizationCode string             `json:"-"`
}

type OrderDestination struct {
	Location string `json:"location"`
	CityID   string `json:"city_id"`
}

type CreateOrderResponse struct {
	OrderID     string  `json:"order_id"`
	TotalAmount float64 `json:"total_amount"`
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
	OrderID       string              `json:"order_id"`
	OrderDate     string              `json:"order_date"`
	FleetName     string              `json:"fleet_name"`
	RentType      int                 `json:"rent_type"`
	RentTypeLabel string              `json:"rent_type_label"`
	Duration      int                 `json:"duration"`
	DurationUom   string              `json:"duration_uom"`
	Price         float64             `json:"price"`
	Quantity      int                 `json:"quantity"`
	TotalAmount   float64             `json:"total_amount"`
	Pickup        OrderDetailPickup   `json:"pickup"`
	Destination   []OrderDetailDest   `json:"destination"`
	Addon         []OrderDetailAddon  `json:"addon"`
	Customer      OrderDetailCustomer `json:"customer"`
}

type OrderDetailPickup struct {
	PickupLocation string `json:"pickup_location"`
	PickupCity     string `json:"pickup_city"`
}

type OrderDetailDest struct {
	City     string `json:"city"`
	Location string `json:"location"`
}

type OrderDetailAddon struct {
	AddonName string `json:"addon_name"`
}

type OrderDetailCustomer struct {
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	CustomerEmail   string `json:"customer_email"`
	CustomerAddress string `json:"customer_address"`
}
