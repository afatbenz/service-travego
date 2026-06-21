package model

import "time"

type Garage struct {
	GarageID       string    `json:"garage_id"`
	OrganizationID string    `json:"organization_id"`
	GarageName     string    `json:"garage_name"`
	GarageAddress  string    `json:"garage_address"`
	GarageCity     string    `json:"garage_city"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      string    `json:"created_by"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by"`
}

type GarageWithLabel struct {
	GarageID        string    `json:"garage_id"`
	OrganizationID  string    `json:"organization_id"`
	GarageName      string    `json:"garage_name"`
	GarageAddress   string    `json:"garage_address"`
	GarageCity      string    `json:"garage_city"`
	GarageCityLabel string    `json:"garage_city_label"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedBy       string    `json:"created_by"`
	UpdatedAt       time.Time `json:"updated_at"`
	UpdatedBy       string    `json:"updated_by"`
}

type CreateGarageRequest struct {
	GarageName    string `json:"garage_name"`
	GarageAddress string `json:"garage_address"`
	GarageCity    string `json:"garage_city"`
}

type UpdateGarageRequest struct {
	GarageID      string `json:"garage_id"`
	GarageName    string `json:"garage_name"`
	GarageAddress string `json:"garage_address"`
	GarageCity    string `json:"garage_city"`
}

type DeleteGarageRequest struct {
	GarageID string `json:"garage_id"`
}
