package model

type DashboardPartnerSummaryResponse struct {
	Orders   DashboardOrdersSummary `json:"orders"`
	Members  DashboardSummaryItem   `json:"members"`
	Messages DashboardSummaryItem   `json:"messages"`
	Revenue  DashboardSummaryItem   `json:"revenue"`
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
