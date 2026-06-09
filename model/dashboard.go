package model

type DashboardPartnerSummaryResponse struct {
	Orders   DashboardOrdersSummary `json:"orders"`
	Members  DashboardSummaryItem   `json:"members"`
	Messages DashboardSummaryItem   `json:"messages"`
	Revenue  DashboardSummaryItem   `json:"revenue"`
}

type DashboardResponse struct {
	Transaction DashboardTransaction `json:"transaction"`
	Customers   DashboardCustomers   `json:"customers"`
	Messages    DashboardMessages    `json:"messages"`
	Revenue     DashboardRevenue     `json:"revenue"`
	Expenses    DashboardRevenue     `json:"expenses"`
}

type DashboardTransaction struct {
	TotalOrder      int     `json:"total_order"`
	PrevTotalOrders int     `json:"prev_total_orders"`
	OrderPercentage float64 `json:"order_percentage"`
}

type DashboardCustomers struct {
	TotalCustomers     int     `json:"total_customers"`
	PrevTotalCustomers int     `json:"prev_total_customers"`
	CustomerPercentage float64 `json:"customer_percentage"`
}

type DashboardMessages struct {
	Current  int `json:"current"`
	Previous int `json:"previous"`
}

type DashboardRevenue struct {
	Current             int                              `json:"current"`
	TotalAmount         float64                          `json:"total_amount"`
	Previous            int                              `json:"previous"`
	PrevAmount          float64                          `json:"prev_amount"`
	TransactionMetrics  []DashboardTransactionMetricItem `json:"transaction_metrics"`
	TransactionCategory []DashboardTransactionMetricItem `json:"transaction_category"`
}

type DashboardTransactionMetricItem struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type DashboardTopDestination struct {
	CityID    string `json:"city_id"`
	CityLabel string `json:"city_label"`
	Total     int    `json:"total"`
}

type DashboardTopPickupCity struct {
	PickupCityID    string `json:"pickup_city_id"`
	PickupCityLabel string `json:"pickup_city_label"`
	Total           int    `json:"total"`
}

type DashboardTopFleet struct {
	VehicleID   string `json:"vehicle_id"`
	PlateNumber string `json:"plate_number"`
	Total       int    `json:"total"`
}

type DashboardTopTourPackage struct {
	PackageName string `json:"package_name"`
	Total       int    `json:"total"`
}

type DashboardTopDriver struct {
	Fullname string `json:"fullname"`
	Total    int    `json:"total"`
}

type DashboardTopCustomer struct {
	CustomerName string `json:"customer_name"`
	Total        int    `json:"total"`
}

type DashboardOrdersSummary struct {
	TotalOrders int     `json:"total_orders"`
	Percentage  float64 `json:"percentage"`
	Direction   string  `json:"direction"`
	Period      string  `json:"period"`
}

type DashboardSummaryItem struct {
	Total      float64 `json:"total"`
	Percentage float64 `json:"percentage"`
	Direction  string  `json:"direction"`
	Period     string  `json:"period"`
}
