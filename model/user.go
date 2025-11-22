package model

import (
	"time"
)

// User represents a user entity
type User struct {
	UserID         string     `json:"user_id"` // UUID
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
	Gender         string     `json:"gender"` // M/F
	DateOfBirth    *time.Time `json:"date_of_birth"`
	Status         int        `json:"status"` // 1: active, 2: unverified
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"-"`
}
