package model

type FleetFuelType struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type FleetConfig struct {
	FuelType []FleetFuelType `json:"fuel_type"`
}
