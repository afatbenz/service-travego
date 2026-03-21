package model

type CustomerListItem struct {
	CustomerID      string `json:"customer_id"`
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	CustomerEmail   string `json:"customer_email"`
	CustomerAddress string `json:"customer_address"`
	OrganizationID  string `json:"organization_id"`
}

