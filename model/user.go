package model

import (
	"time"
)

// User represents a user entity
type User struct {
    UserID         string     `json:"user_id"`
    Username       string     `json:"username"`
    Name           string     `json:"name"`
    Email          string     `json:"email"`
    Password       string     `json:"-"`
    Phone          string     `json:"phone"`
    Address        string     `json:"address"`
    City           string     `json:"city"`
    Province       string     `json:"province"`
    PostalCode     string     `json:"postal_code"`
    OrganizationID string     `json:"organization_id"`
    NPWP           string     `json:"npwp"`
    Gender         string     `json:"gender"`
    DateOfBirth    *time.Time `json:"date_of_birth"`
    Avatar         string     `json:"avatar"`
    IsActive       bool       `json:"is_active"`
    IsVerified     bool       `json:"is_verified"`
    IsAdmin        bool       `json:"is_admin"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
    DeletedAt      *time.Time `json:"-"`
}
