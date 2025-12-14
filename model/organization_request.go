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
