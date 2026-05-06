package model

type LeaveManagementTypeItem struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

type LeaveManagementListItem struct {
	LeaveID        string `json:"leave_id"`
	EmployeeID     string `json:"employee_id"`
	SubstitutedBy  string `json:"substituted_by"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	LeaveType      int    `json:"leave_type"`
	LeaveTypeLabel string `json:"leave_type_label"`
}

type LeaveManagementCreateRequest struct {
	EmployeeID     string `json:"employee_id" validate:"required"`
	SubstituteID   string `json:"substitute_id" validate:"required"`
	LeaveType      int    `json:"leave_type" validate:"required"`
	StartDate      string `json:"start_date" validate:"required"`
	EndDate        string `json:"end_date" validate:"required"`
	Reason         string `json:"reason" validate:"required"`
	AttachmentPath string `json:"attachment_path"`
}
