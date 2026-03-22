package model

type CustomerListItem struct {
	CustomerID      string `json:"customer_id"`
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	CustomerEmail   string `json:"customer_email"`
	CustomerAddress string `json:"customer_address"`
	CustomerCompany string `json:"customer_company"`
	CustomerCity    string `json:"customer_city"`
	CityName        string `json:"city_name"`
	CustomerCityID  string `json:"-"`
	OrganizationID  string `json:"organization_id"`
}

type CustomerCreateRequest struct {
	CustomerName      string `json:"customer_name"`
	CustomerPhone     string `json:"customer_phone"`
	CustomerTelephone string `json:"customer_telephone"`
	CustomerAddress   string `json:"customer_address"`
	CustomerCity      string `json:"customer_city"`
	CustomerEmail     string `json:"customer_email"`
	CustomerCompany   string `json:"customer_company"`
	CustomerBOD       string `json:"customer_bod"`
}
