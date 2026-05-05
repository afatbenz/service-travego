package model

import (
	"time"
)

// Organization represents an organization entity
type Organization struct {
	ID               string    `json:"id"`                // UUID
	OrganizationCode string    `json:"organization_code"` // Unique code (4 vowels + 4 digits)
	OrganizationName string    `json:"organization_name"`
	CompanyName      string    `json:"company_name"`
	Address          string    `json:"address"`
	City             string    `json:"city"`
	Province         string    `json:"province"`
	Phone            string    `json:"phone"`
	Email            string    `json:"email"`
	NPWPNumber       string    `json:"npwp_number"`
	OrganizationType int       `json:"organization_type"`
	PostalCode       string    `json:"postal_code"`
	DomainURL        string    `json:"domain_url"`
	Logo             string    `json:"logo"`
	CreatedBy        string    `json:"created_by"` // User ID who created the organization
	Username         string    `json:"username"`   // Username who created the organization
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// AccountType enum
type AccountType int

const (
	AccountTypePersonal AccountType = 1
	AccountTypeCompany  AccountType = 2
)

// BankAccountPaymentMethod enum
type BankAccountPaymentMethod int

const (
	BankAccountPaymentMethodBankTransfer BankAccountPaymentMethod = 1
	BankAccountPaymentMethodQRIS         BankAccountPaymentMethod = 2
)

// CreateOrganizationBankAccountRequest represents the request payload for creating a bank account
type CreateOrganizationBankAccountRequest struct {
	BankCode           string                   `json:"bank_code"`
	AccountNumber      string                   `json:"account_number"`
	AccountHolder      string                   `json:"account_holder"`
	PaymentMethod      BankAccountPaymentMethod `json:"payment_method"`
	MerchantName       string                   `json:"merchant_name"`
	MerchantMCC        string                   `json:"merchant_mcc"`
	MerchantAddress    string                   `json:"merchant_address"`
	MerchantCity       string                   `json:"merchant_city"`
	MerchantPostalCode string                   `json:"merchant_postal_code"`
	AccountType        AccountType              `json:"account_type"`
}

// OrganizationBankAccountResponse represents the response for bank account
type OrganizationBankAccountResponse struct {
	BankAccountID      string      `json:"bank_account_id"`
	BankCode           string      `json:"bank_code"`
	AccountNumber      string      `json:"account_number"`
	AccountName        string      `json:"account_name"`
	MerchantID         string      `json:"merchant_id"`
	MerchantNMID       string      `json:"merchant_nmid"`
	MerchantMCC        string      `json:"merchant_mcc"`
	MerchantAddress    string      `json:"merchant_address"`
	MerchantCity       string      `json:"merchant_city"`
	MerchantPostalCode string      `json:"merchant_postal_code"`
	AccountType        AccountType `json:"account_type"`
	PaymentMethod      string      `json:"payment_method"`
	CreatedAt          time.Time   `json:"created_at"`
	CreatedBy          string      `json:"created_by"`
	CreatedByFullName  string      `json:"created_by_fullname"`
	BankName           string      `json:"bank_name"`
	Active             bool        `json:"active"`
	BankIcon           string      `json:"bank_icon"`
}

type OrganizationDivision struct {
	DivisionID   string `json:"division_id"`
	DivisionName string `json:"division_name"`
	Description  string `json:"description"`
	Status       int    `json:"status"`
	CreatedBy    string `json:"created_by"`
	CreatedAt    string `json:"created_at"`
	UpdatedBy    string `json:"updated_by"`
	UpdatedAt    string `json:"updated_at"`
}

type OrganizationRole struct {
	RoleID       string `json:"role_id"`
	RoleName     string `json:"role_name"`
	Description  string `json:"description"`
	DivisionID   string `json:"division_id"`
	DivisionName string `json:"division_name"`
	Status       int    `json:"status"`
	CreatedBy    string `json:"created_by"`
	CreatedAt    string `json:"created_at"`
	UpdatedBy    string `json:"updated_by"`
	UpdatedAt    string `json:"updated_at"`
}

type EmployeeListItem struct {
	UUID                string  `json:"uuid"`
	EmployeeID          string  `json:"employee_id"`
	NIK                 string  `json:"nik"`
	Fullname            string  `json:"fullname"`
	Avatar              string  `json:"avatar"`
	Phone               string  `json:"phone"`
	BirthDate           string  `json:"birth_date"`
	Email               string  `json:"email"`
	Address             string  `json:"address"`
	AddressCity         int     `json:"address_city"`
	AddressCityName     string  `json:"address_city_name"`
	JoinDate            string  `json:"join_date"`
	RoleID              string  `json:"role_id"`
	RoleName            string  `json:"role_name"`
	DivisionName        string  `json:"division_name"`
	ContractStatus      *int    `json:"contract_status"`
	ContractStatusLabel string  `json:"contract_status_label"`
	ResignDate          *string `json:"resign_date"`
	Status              int     `json:"status"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

type EmployeeDetailResponse struct {
	UUID                string  `json:"uuid"`
	EmployeeID          string  `json:"employee_id"`
	NIK                 string  `json:"nik"`
	Fullname            string  `json:"fullname"`
	Avatar              string  `json:"avatar"`
	Phone               string  `json:"phone"`
	BirthDate           string  `json:"birth_date"`
	Email               string  `json:"email"`
	Address             string  `json:"address"`
	AddressCity         int     `json:"address_city"`
	AddressCityName     string  `json:"address_city_name"`
	JoinDate            string  `json:"join_date"`
	RoleID              string  `json:"role_id"`
	RoleName            string  `json:"role_name"`
	DivisionID          string  `json:"division_id"`
	DivisionName        string  `json:"division_name"`
	ContractStatus      *int    `json:"contract_status"`
	ContractStatusLabel string  `json:"contract_status_label"`
	ResignDate          *string `json:"resign_date"`
	Status              int     `json:"status"`
	CreatedBy           string  `json:"created_by"`
	CreatedAt           string  `json:"created_at"`
	UpdatedBy           string  `json:"updated_by"`
	UpdatedAt           string  `json:"updated_at"`
}

type EmployeeShiftScheduleRow struct {
	UUID       string
	EmployeeID string
	Fullname   string
	Avatar     string
	RoleName   string
	ShiftID    string
	ShiftDate  string
	ShiftType  *int
}

type EmployeeShiftScheduleItem struct {
	ShiftID   string `json:"shift_id"`
	ShiftDate string `json:"shift_date"`
	ShiftType int    `json:"shift_type"`
}

type EmployeeShiftScheduleEmployee struct {
	UUID         string                      `json:"uuid"`
	EmployeeID   string                      `json:"employee_id"`
	Fullname     string                      `json:"fullname"`
	Avatar       string                      `json:"avatar"`
	RoleName     string                      `json:"role_name"`
	TotalWorkday int                         `json:"total_workday"`
	TotalOffday  int                         `json:"total_offday"`
	Shifts       []EmployeeShiftScheduleItem `json:"shifts"`
}

type EmployeeShiftScheduleResponse struct {
	StartDate string                          `json:"start_date"`
	EndDate   string                          `json:"end_date"`
	Employees []EmployeeShiftScheduleEmployee `json:"employees"`
}
