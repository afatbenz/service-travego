package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"service-travego/model"
	"strconv"
	"strings"
	"time"
)

func (s *OrganizationService) ensureContractTypesLoaded() {
	if s.contractTypeLabels != nil {
		return
	}
	f, err := os.Open("config/common.json")
	if err != nil {
		s.contractTypeLabels = map[int]string{}
		return
	}
	defer f.Close()

	var cfg model.CommonConfig
	d := json.NewDecoder(f)
	if err := d.Decode(&cfg); err != nil {
		s.contractTypeLabels = map[int]string{}
		return
	}

	out := make(map[int]string, len(cfg.ContractType))
	for _, it := range cfg.ContractType {
		out[it.ID] = it.Label
	}
	s.contractTypeLabels = out
}

func (s *OrganizationService) EmployeeAll(organizationID, divisionName string) ([]model.EmployeeListItem, error) {
	items, err := s.orgRepo.ListEmployees(organizationID, divisionName)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, 500, "failed to get employees")
	}
	s.ensureLocationsLoaded()
	s.ensureContractTypesLoaded()
	for i := range items {
		items[i].AddressCityName = s.citiesName[strconv.Itoa(items[i].AddressCity)]
		if items[i].ContractStatus != nil {
			items[i].ContractStatusLabel = s.contractTypeLabels[*items[i].ContractStatus]
		}
	}
	return items, nil
}

func (s *OrganizationService) EmployeeDetail(organizationID, uuid string) (*model.EmployeeDetailResponse, error) {
	it, err := s.orgRepo.EmployeeDetail(organizationID, uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, 404, "employee not found")
		}
		return nil, NewServiceError(ErrInternalServer, 500, "failed to get employee detail")
	}
	s.ensureLocationsLoaded()
	it.AddressCityName = s.citiesName[strconv.Itoa(it.AddressCity)]
	return it, nil
}

func (s *OrganizationService) EmployeeCreate(organizationID, userID string, req *model.CreateEmployeeRequest) (string, error) {
	ok, err := s.orgRepo.RoleExistsForOrgOrDefault(organizationID, strings.TrimSpace(req.RoleID))
	if err != nil {
		return "", NewServiceError(ErrInternalServer, 500, "failed to validate role_id")
	}
	if !ok {
		return "", NewServiceError(ErrInvalidInput, 400, "role_id tidak ditemukan")
	}

	if ok, err := s.orgRepo.EmployeeIDExists(organizationID, strings.TrimSpace(req.EmployeeID)); err != nil {
		return "", NewServiceError(ErrInternalServer, 500, "failed to validate employee_id")
	} else if ok {
		return "", NewServiceError(ErrInvalidInput, 400, "DUPLICATE_EMPLOYEE_ID")
	}

	if nik := strings.TrimSpace(req.NIK); nik != "" {
		if ok, err := s.orgRepo.NIKExists(organizationID, nik); err != nil {
			return "", NewServiceError(ErrInternalServer, 500, "failed to validate nik")
		} else if ok {
			return "", NewServiceError(ErrInvalidInput, 400, "DUPLICATE_NIK")
		}
	}

	id, err := s.orgRepo.CreateEmployee(organizationID, userID, req)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid birth_date") {
			return "", NewServiceError(ErrInvalidInput, 400, "invalid birth_date")
		}
		if strings.Contains(strings.ToLower(err.Error()), "invalid join_date") {
			return "", NewServiceError(ErrInvalidInput, 400, "invalid join_date")
		}
		if strings.Contains(strings.ToLower(err.Error()), "invalid resign_date") {
			return "", NewServiceError(ErrInvalidInput, 400, "invalid resign_date")
		}
		return "", NewServiceError(ErrInternalServer, 500, "failed to create employee")
	}
	return id, nil
}

func (s *OrganizationService) EmployeeUpdate(organizationID, userID string, req *model.UpdateEmployeeRequest) error {
	ok, err := s.orgRepo.RoleExistsForOrgOrDefault(organizationID, strings.TrimSpace(req.RoleID))
	if err != nil {
		return NewServiceError(ErrInternalServer, 500, "failed to validate role_id")
	}
	if !ok {
		return NewServiceError(ErrInvalidInput, 400, "role_id tidak ditemukan")
	}

	err = s.orgRepo.UpdateEmployee(organizationID, userID, req)
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "employee not found")
		}
		if strings.Contains(strings.ToLower(err.Error()), "invalid birth_date") {
			return NewServiceError(ErrInvalidInput, 400, "invalid birth_date")
		}
		if strings.Contains(strings.ToLower(err.Error()), "invalid join_date") {
			return NewServiceError(ErrInvalidInput, 400, "invalid join_date")
		}
		if strings.Contains(strings.ToLower(err.Error()), "invalid resign_date") {
			return NewServiceError(ErrInvalidInput, 400, "invalid resign_date")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to update employee")
	}
	return nil
}

func (s *OrganizationService) EmployeeDelete(organizationID, userID, uuid string) error {
	if strings.TrimSpace(uuid) == "" {
		return NewServiceError(ErrInvalidInput, 400, "ID is required")
	}
	err := s.orgRepo.DeactivateEmployeeByEmployeeID(organizationID, userID, strings.TrimSpace(uuid))
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "employee not found")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to delete employee")
	}
	return nil
}

func (s *OrganizationService) EmployeeShiftSchedule(organizationID string, req *model.EmployeeShiftScheduleRequest) (*model.EmployeeShiftScheduleResponse, error) {
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endDate := startDate.AddDate(0, 0, 6)

	startDateRaw := strings.TrimSpace(req.StartDate)
	endDateRaw := strings.TrimSpace(req.EndDate)
	if startDateRaw != "" || endDateRaw != "" {
		if startDateRaw == "" {
			return nil, NewServiceError(ErrInvalidInput, 400, "start_date is required")
		}
		if endDateRaw == "" {
			return nil, NewServiceError(ErrInvalidInput, 400, "end_date is required")
		}

		startParsed, err := time.Parse("2006-01-02", startDateRaw)
		if err != nil {
			return nil, NewServiceError(ErrInvalidInput, 400, "invalid start_date")
		}
		endParsed, err := time.Parse("2006-01-02", endDateRaw)
		if err != nil {
			return nil, NewServiceError(ErrInvalidInput, 400, "invalid end_date")
		}

		startDate = time.Date(startParsed.Year(), startParsed.Month(), startParsed.Day(), 0, 0, 0, 0, now.Location())
		endDate = time.Date(endParsed.Year(), endParsed.Month(), endParsed.Day(), 0, 0, 0, 0, now.Location())
		if endDate.Before(startDate) {
			return nil, NewServiceError(ErrInvalidInput, 400, "invalid date range")
		}
	}

	rows, err := s.orgRepo.EmployeeShiftSchedule(
		organizationID,
		strings.TrimSpace(req.RoleID),
		strings.TrimSpace(req.DivisionID),
		startDate,
		endDate,
	)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, 500, "failed to get employee shift schedule")
	}

	employees := make([]model.EmployeeShiftScheduleEmployee, 0)
	index := make(map[string]int)
	employeeUUIDs := make([]string, 0)

	for _, r := range rows {
		i, ok := index[r.UUID]
		if !ok {
			employees = append(employees, model.EmployeeShiftScheduleEmployee{
				UUID:       r.UUID,
				EmployeeID: r.EmployeeID,
				Fullname:   r.Fullname,
				Avatar:     r.Avatar,
				RoleName:   r.RoleName,
				Shifts:     []model.EmployeeShiftScheduleItem{},
			})
			i = len(employees) - 1
			index[r.UUID] = i
			employeeUUIDs = append(employeeUUIDs, r.UUID)
		}
		if strings.TrimSpace(r.ShiftID) != "" && strings.TrimSpace(r.ShiftDate) != "" && r.ShiftType != nil {
			employees[i].Shifts = append(employees[i].Shifts, model.EmployeeShiftScheduleItem{
				ShiftID:   r.ShiftID,
				ShiftDate: r.ShiftDate,
				ShiftType: *r.ShiftType,
			})
		}
	}

	monthStart, monthEnd := dominantMonthRange(startDate, endDate)
	offdays, err := s.orgRepo.EmployeeShiftOffdayCounts(organizationID, employeeUUIDs, monthStart, monthEnd)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, 500, "failed to get employee shift offday")
	}

	totalDays := daysInMonth(monthStart)
	for i := range employees {
		off := offdays[employees[i].UUID]
		employees[i].TotalOffday = off
		employees[i].TotalWorkday = totalDays - off
	}

	return &model.EmployeeShiftScheduleResponse{
		StartDate: startDate.Format("2006-01-02"),
		EndDate:   endDate.Format("2006-01-02"),
		Employees: employees,
	}, nil
}

func dominantMonthRange(startDate, endDate time.Time) (time.Time, time.Time) {
	if endDate.Before(startDate) {
		startDate, endDate = endDate, startDate
	}
	loc := startDate.Location()

	monthCursor := time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, loc)
	var bestMonthStart time.Time
	bestDays := -1

	for !monthCursor.After(endDate) {
		nextMonth := monthCursor.AddDate(0, 1, 0)
		monthEnd := nextMonth.AddDate(0, 0, -1)

		segStart := maxTime(startDate, monthCursor)
		segEnd := minTime(endDate, monthEnd)
		days := 0
		if !segEnd.Before(segStart) {
			days = int(segEnd.Sub(segStart).Hours()/24) + 1
		}

		if days > bestDays || (days == bestDays && monthCursor.After(bestMonthStart)) {
			bestDays = days
			bestMonthStart = monthCursor
		}

		monthCursor = nextMonth
	}

	if bestDays < 0 {
		bestMonthStart = time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, loc)
	}
	bestMonthEnd := bestMonthStart.AddDate(0, 1, 0).AddDate(0, 0, -1)
	return bestMonthStart, bestMonthEnd
}

func daysInMonth(monthStart time.Time) int {
	loc := monthStart.Location()
	start := time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 1, 0).AddDate(0, 0, -1)
	return end.Day()
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func (s *OrganizationService) EmployeeShiftSetSchedule(organizationID, userID string, req *model.EmployeeShiftSetScheduleRequest) (interface{}, error) {
	typ := strings.ToLower(strings.TrimSpace(req.Type))
	if typ == "delete" {
		if strings.TrimSpace(req.ShiftID) == "" {
			return nil, NewServiceError(ErrInvalidInput, 400, "shift_id is required")
		}
		if strings.TrimSpace(req.EmployeeID) == "" {
			return nil, NewServiceError(ErrInvalidInput, 400, "employee_id is required")
		}
		if err := s.orgRepo.DeleteEmployeeShiftSchedule(organizationID, strings.TrimSpace(req.EmployeeID), strings.TrimSpace(req.ShiftID)); err != nil {
			if err == sql.ErrNoRows {
				return nil, NewServiceError(ErrNotFound, 404, "shift not found")
			}
			return nil, NewServiceError(ErrInternalServer, 500, "failed to delete shift schedule")
		}
		return map[string]interface{}{"shift_id": strings.TrimSpace(req.ShiftID)}, nil
	}

	if typ != "submit" {
		return nil, NewServiceError(ErrInvalidInput, 400, "invalid type")
	}

	items := req.Schedules
	if len(items) == 0 {
		if strings.TrimSpace(req.EmployeeID) == "" {
			return nil, NewServiceError(ErrInvalidInput, 400, "employee_id is required")
		}
		if strings.TrimSpace(req.ShiftDate) == "" {
			return nil, NewServiceError(ErrInvalidInput, 400, "shift_date is required")
		}
		items = []model.EmployeeShiftSubmitItem{
			{
				EmployeeID: strings.TrimSpace(req.EmployeeID),
				ShiftDate:  strings.TrimSpace(req.ShiftDate),
				ShiftType:  req.ShiftType,
			},
		}
	}

	ids, err := s.orgRepo.CreateEmployeeShiftSchedules(organizationID, userID, items)
	if err != nil {
		fmt.Println("CreateEmployeeShiftSchedules error:", err)
		fmt.Println("CreateEmployeeShiftSchedules org_id:", organizationID, "user_id:", userID)
		if strings.Contains(strings.ToLower(err.Error()), "invalid shift_date") {
			return nil, NewServiceError(ErrInvalidInput, 400, "invalid shift_date")
		}
		return nil, NewServiceError(ErrInternalServer, 500, "failed to create shift schedule")
	}
	if len(ids) == 0 {
		fmt.Println("CreateEmployeeShiftSchedules inserted 0 rows org_id:", organizationID, "user_id:", userID)
		return nil, NewServiceError(ErrInternalServer, 500, "failed to create shift schedule")
	}
	return map[string]interface{}{"shift_id": ids[0]}, nil
}
