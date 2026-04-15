package model

// CreateOrganizationRequest represents create organization request payload
type CreateOrganizationRequest struct {
	OrganizationCode string `json:"organization_code" validate:"omitempty"` // Optional, will be auto-generated if not provided
	OrganizationName string `json:"organization_name" validate:"required"`
	CompanyName      string `json:"company_name" validate:"required"`
	Address          string `json:"address" validate:"required"`
	City             int    `json:"city" validate:"required"`
	Province         int    `json:"province" validate:"required"`
	Phone            string `json:"phone" validate:"required"`
	Email            string `json:"email" validate:"required,email"`
	NPWPNumber       string `json:"npwp_number" validate:"omitempty"`
	OrganizationType int    `json:"organization_type" validate:"required"`
	PostalCode       string `json:"postal_code" validate:"omitempty"`
}

// JoinOrganizationRequest represents join organization request payload
type JoinOrganizationRequest struct {
	OrganizationCode string `json:"organization_code" validate:"required"`
}

// UpdateOrganizationBankAccountRequest represents update organization bank account request payload
type UpdateOrganizationBankAccountRequest struct {
	BankAccountID string `json:"bank_account_id" validate:"required"`
	Active        *bool  `json:"active"`
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
}

// DeleteOrganizationBankAccountRequest represents delete organization bank account request payload
type DeleteOrganizationBankAccountRequest struct {
	BankAccountID string `json:"bank_account_id" validate:"required"`
}

type CreateOrganizationDivisionRequest struct {
	DivisionName string `json:"division_name" validate:"required"`
	Description  string `json:"description"`
}

type UpdateOrganizationDivisionRequest struct {
	DivisionID   string `json:"division_id" validate:"required"`
	DivisionName string `json:"division_name" validate:"required"`
	Description  string `json:"description"`
}

type DeleteOrganizationDivisionRequest struct {
	DivisionID string `json:"division_id" validate:"required"`
}

type CreateOrganizationRoleRequest struct {
	RoleName    string `json:"role_name" validate:"required"`
	Description string `json:"description"`
	DivisionID  string `json:"division_id" validate:"required"`
}

type UpdateOrganizationRoleRequest struct {
	RoleID      string `json:"role_id" validate:"required"`
	RoleName    string `json:"role_name" validate:"required"`
	Description string `json:"description"`
	DivisionID  string `json:"division_id" validate:"required"`
}

type DeleteOrganizationRoleRequest struct {
	RoleID string `json:"role_id" validate:"required"`
}
