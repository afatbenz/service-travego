package model

type AssistantAccountListItem struct {
	AssistantID    string `json:"assistant_id"`
	EmployeeID     string `json:"employee_id"`
	CreatedAt      string `json:"created_at"`
	OrganizationID string `json:"organization_id"`
	Avatar         string `json:"avatar"`
	Fullname       string `json:"fullname"`
	RoleName       string `json:"role_name"`
	DivisionName   string `json:"division_name"`
	AccountNumber  string `json:"account_number"`
	UserType       int    `json:"user_type"`
}

type AssistantSubmitRequest struct {
	EmployeeID    string `json:"employee_id"`
	UserType      int    `json:"user_type" validate:"required"`
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
}

type AssistantUpdateRequest struct {
	AssistantID   string  `json:"assistant_id" validate:"required"`
	AccountName   *string `json:"account_name"`
	AccountNumber *string `json:"account_number"`
}

type AssistantDeleteRequest struct {
	EmployeeID string `json:"employee_id" validate:"required"`
}

type AssistantEmployeeTarget struct {
	UUID       string
	EmployeeID string
	Fullname   string
	Phone      string
}

type EmployeeWhatsAppResponse struct {
	EmployeeID string `json:"employee_id"`
	Phone      string `json:"phone"`
	HasPhone   bool   `json:"has_phone"`
}

type AssistantWhatsAppBusinessUpdateRequest struct {
	AccountNumber string `json:"account_number" validate:"required"`
}

type AssistantWhatsAppBusinessListResponse struct {
	AccountNumber string `json:"account_number"`
	DeviceID      string `json:"device_id"`
	DeviceName    string `json:"device_name"`
	DeviceToken   string `json:"device_token"`
	Status        int    `json:"status"`
	StatusLabel   string `json:"status_label"`
	Available     bool   `json:"available"`
}
