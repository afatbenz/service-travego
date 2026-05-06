package model

type LeaveManagementTypeItem struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

type LeaveManagementListItem struct {
	LeaveID         string `json:"leave_id"`
	EmployeeID      string `json:"employee_id"`
	SubstitutedBy   string `json:"substituted_by"`
	StartDate       string `json:"start_date"`
	EndDate         string `json:"end_date"`
	LeaveType       int    `json:"leave_type"`
	LeaveTypeLabel  string `json:"leave_type_label"`
}
