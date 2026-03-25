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
}

type DashboardTransaction struct {
	TotalOrder      int     `json:"total_order"`
	OrderPercentage float64 `json:"order_percentage"`
}

type DashboardCustomers struct {
	TotalCustomers     int     `json:"total_customers"`
	CustomerPercentage float64 `json:"customer_percentage"`
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
