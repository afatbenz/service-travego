package model

import (
	"time"
)

// OrganizationUser represents a user-organization relationship
type OrganizationUser struct {
	UUID             string    `json:"uuid"`
	UserID           string    `json:"user_id"`
	OrganizationID   string    `json:"organization_id"`
	OrganizationRole int       `json:"organization_role"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	CreatedBy        string    `json:"created_by"`
	UpdatedAt        time.Time `json:"updated_at"`
	UpdatedBy        string    `json:"updated_by"`
}
