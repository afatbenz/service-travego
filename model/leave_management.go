package model

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

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

func (r *LeaveManagementCreateRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		EmployeeID     string      `json:"employee_id"`
		SubstituteID   string      `json:"substitute_id"`
		LeaveType      interface{} `json:"leave_type"`
		StartDate      string      `json:"start_date"`
		EndDate        string      `json:"end_date"`
		Reason         string      `json:"reason"`
		AttachmentPath string      `json:"attachment_path"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.EmployeeID = raw.EmployeeID
	r.SubstituteID = raw.SubstituteID
	r.StartDate = raw.StartDate
	r.EndDate = raw.EndDate
	r.Reason = raw.Reason
	r.AttachmentPath = raw.AttachmentPath

	switch v := raw.LeaveType.(type) {
	case nil:
		r.LeaveType = 0
	case float64:
		r.LeaveType = int(v)
	case int:
		r.LeaveType = v
	case int64:
		r.LeaveType = int(v)
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			r.LeaveType = 0
			break
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return errors.New("invalid leave_type")
		}
		r.LeaveType = n
	default:
		return errors.New("invalid leave_type")
	}

	return nil
}
