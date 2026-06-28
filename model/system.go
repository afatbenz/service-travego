package model

type DeviceListItem struct {
	DeviceID         string `json:"device_id"`
	DeviceName       string `json:"device_name"`
	DeviceToken      string `json:"device_token"`
	OrganizationName string `json:"organization_name"`
	AccountNumber    string `json:"account_number"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type DeviceEnableRequest struct {
	Account     string `json:"account" validate:"required"`
	DeviceID    string `json:"device_id" validate:"required"`
	DeviceName  string `json:"device_name" validate:"required"`
	DeviceToken string `json:"device_token" validate:"required"`
}

type DeviceDisableRequest struct {
	Account string `json:"account" validate:"required"`
}

type PeriodSummary struct {
	CurrentPeriod int64 `json:"current_period"`
	LastPeriod    int64 `json:"last_period"`
}

type MetricItem struct {
	PackageName string  `json:"package_name"`
	Revenue     float64 `json:"revenue"`
}

type MetricPeriod struct {
	Period string       `json:"period"`
	Items  []MetricItem `json:"items"`
}

type VisitorMetricPeriod struct {
	Period     string `json:"period"`
	TotalVisit int64  `json:"total_visit"`
}

type ActiveUserMetricPeriod struct {
	Period      string `json:"period"`
	ActiveUsers int64  `json:"active_users"`
}

type SystemSummarymarizeResponse struct {
	Revenue           PeriodSummary            `json:"revenue"`
	TotalUsers        PeriodSummary            `json:"total_users"`
	ActiveUsers       PeriodSummary            `json:"active_users"`
	Organization      PeriodSummary            `json:"organization"`
	Period            string                   `json:"period"`
	Matrics           []MetricPeriod           `json:"matrics"`
	TotalVisit        PeriodSummary            `json:"total_visit"`
	VisitorMatrics    []VisitorMetricPeriod    `json:"visitor_matrics"`
	ActiveUserMatrics []ActiveUserMetricPeriod `json:"active_user_matrics"`
}
