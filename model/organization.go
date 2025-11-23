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
	CreatedBy        string    `json:"created_by"` // User ID who created the organization
	Username         string    `json:"username"`   // Username who created the organization
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
