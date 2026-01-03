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
