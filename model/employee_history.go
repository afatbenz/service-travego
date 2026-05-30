package model

type EmployeeOperationsHistoryResponse struct {
	TotalSchedules int                             `json:"total_schedules"`
	OrderHistory   []EmployeeOperationsHistoryItem `json:"order_history"`
}

type EmployeeOperationsHistoryItem struct {
	OrderID       string `json:"order_id"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	Destinations  string `json:"destinations"`
	RentTypeLabel string `json:"rent_type_label"`
	FleetName     string `json:"fleet_name"`
	VehicleID     string `json:"vehicle_id"`
	PlateNumber   string `json:"plate_number"`
}

type EmployeeOperationsHistoryRow struct {
	Fullname    string
	OrderID     string
	StartDate   string
	EndDate     string
	CityIDs     string
	RentType    int
	FleetName   string
	VehicleID   string
	PlateNumber string
}
