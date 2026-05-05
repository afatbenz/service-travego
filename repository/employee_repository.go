package repository

import (
	"database/sql"
	"fmt"
	"service-travego/database"
	"service-travego/model"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (r *OrganizationRepository) RoleExistsForOrgOrDefault(organizationID, roleID string) (bool, error) {
	defaultOrgID := "00000000-0000-0000-0000-000000000000"
	legacyDefaultOrgID := "000"

	orgExpr := "organization_id IN (" + r.getPlaceholder(2) + "," + r.getPlaceholder(3) + "," + r.getPlaceholder(4) + ")"
	roleExpr := "role_id = " + r.getPlaceholder(1)
	if r.driver != "mysql" {
		orgExpr = "organization_id::text IN (" + r.getPlaceholder(2) + "," + r.getPlaceholder(3) + "," + r.getPlaceholder(4) + ")"
		roleExpr = "role_id::text = " + r.getPlaceholder(1)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM organization_roles
		WHERE %s AND %s AND COALESCE(status, 0) > 0
	`, roleExpr, orgExpr)

	var cnt int
	if err := database.QueryRow(r.db, query, roleID, organizationID, defaultOrgID, legacyDefaultOrgID).Scan(&cnt); err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (r *OrganizationRepository) EmployeeIDExists(organizationID, employeeID string) (bool, error) {
	orgExpr := "organization_id = " + r.getPlaceholder(1)
	if r.driver != "mysql" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(1)
	}
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM employee
		WHERE %s AND employee_id = %s AND COALESCE(status, 0) > 0
	`, orgExpr, r.getPlaceholder(2))

	var cnt int
	if err := database.QueryRow(r.db, query, organizationID, employeeID).Scan(&cnt); err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (r *OrganizationRepository) NIKExists(organizationID, nik string) (bool, error) {
	orgExpr := "organization_id = " + r.getPlaceholder(1)
	if r.driver != "mysql" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(1)
	}
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM employee
		WHERE %s AND nik = %s AND COALESCE(status, 0) > 0
	`, orgExpr, r.getPlaceholder(2))

	var cnt int
	if err := database.QueryRow(r.db, query, organizationID, nik).Scan(&cnt); err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (r *OrganizationRepository) ListEmployees(organizationID, divisionName string) ([]model.EmployeeListItem, error) {
	orgExpr := "e.organization_id = " + r.getPlaceholder(1)

	divisionNameNormalized := strings.ToLower(strings.TrimSpace(divisionName))
	operationDivisionID := "fe8b3916-5eff-420c-8110-8d974d767afe"
	isOperationDivision :=
		divisionNameNormalized == "operation" ||
			divisionNameNormalized == "operatio" ||
			divisionNameNormalized == "operations" ||
			divisionNameNormalized == "operator" ||
			divisionNameNormalized == "operators"

	divisionFilter := ""
	if isOperationDivision {
		if r.driver == "mysql" {
			divisionFilter = fmt.Sprintf(`
				AND (
					LOWER(COALESCE(d.division_name, '')) IN ('operation','operatio','operations','operator','operators')
					OR COALESCE(d.division_id, '') = '%s'
					OR COALESCE(r.division_id, '') = '%s'
				)
			`, operationDivisionID, operationDivisionID)
		} else {
			divisionFilter = fmt.Sprintf(`
				AND (
					LOWER(COALESCE(d.division_name, '')) IN ('operation','operatio','operations','operator','operators')
					OR COALESCE(d.division_id::text, '') = '%s'
					OR COALESCE(r.division_id::text, '') = '%s'
				)
			`, operationDivisionID, operationDivisionID)
		}
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(e.uuid::text, ''),
			COALESCE(e.employee_id, ''),
			COALESCE(e.nik, ''),
			COALESCE(e.fullname, ''),
			COALESCE(e.avatar, ''),
			COALESCE(e.phone, ''),
			e.birth_date,
			COALESCE(e.email, ''),
			COALESCE(e.address, ''),
			COALESCE(e.address_city, 0),
			e.join_date,
			COALESCE(e.role_id::text, ''),
			COALESCE(r.role_name, ''),
			COALESCE(d.division_name, ''),
			e.contract_status,
			e.resign_date,
			COALESCE(e.status, 0),
			e.created_at,
			e.updated_at
		FROM employee e
		LEFT JOIN organization_roles r ON r.role_id::text = e.role_id::text
		LEFT JOIN organization_divisions d ON d.division_id::text = r.division_id::text
		WHERE %s AND COALESCE(e.status, 0) > 0
		%s
		ORDER BY e.created_at DESC
	`, orgExpr, divisionFilter)

	rows, err := database.Query(r.db, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.EmployeeListItem, 0)
	for rows.Next() {
		var it model.EmployeeListItem
		var birthDate sql.NullTime
		var joinDate sql.NullTime
		var contractStatus sql.NullInt64
		var resignDate sql.NullTime
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(
			&it.UUID,
			&it.EmployeeID,
			&it.NIK,
			&it.Fullname,
			&it.Avatar,
			&it.Phone,
			&birthDate,
			&it.Email,
			&it.Address,
			&it.AddressCity,
			&joinDate,
			&it.RoleID,
			&it.RoleName,
			&it.DivisionName,
			&contractStatus,
			&resignDate,
			&it.Status,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		if birthDate.Valid {
			it.BirthDate = birthDate.Time.Format("2006-01-02")
		}
		if joinDate.Valid {
			it.JoinDate = joinDate.Time.Format("2006-01-02")
		}
		if contractStatus.Valid {
			v := int(contractStatus.Int64)
			it.ContractStatus = &v
		}
		if resignDate.Valid {
			v := resignDate.Time.Format("2006-01-02")
			it.ResignDate = &v
		}
		if createdAt.Valid {
			it.CreatedAt = createdAt.Time.Format("2006-01-02 15:04:05")
		}
		if updatedAt.Valid {
			it.UpdatedAt = updatedAt.Time.Format("2006-01-02 15:04:05")
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *OrganizationRepository) EmployeeDetail(organizationID, uuid string) (*model.EmployeeDetailResponse, error) {
	orgExpr := "e.organization_id = " + r.getPlaceholder(2)
	uuidExpr := "e.uuid = " + r.getPlaceholder(1)
	if r.driver != "mysql" {
		orgExpr = "e.organization_id::text = " + r.getPlaceholder(2)
		uuidExpr = "e.uuid::text = " + r.getPlaceholder(1)
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(e.uuid::text, ''),
			COALESCE(e.employee_id, ''),
			COALESCE(e.nik, ''),
			COALESCE(e.fullname, ''),
			COALESCE(e.avatar, ''),
			COALESCE(e.phone, ''),
			e.birth_date,
			COALESCE(e.email, ''),
			COALESCE(e.address, ''),
			COALESCE(e.address_city, 0),
			e.join_date,
			COALESCE(e.role_id::text, ''),
			COALESCE(r.role_name, ''),
			COALESCE(r.division_id::text, ''),
			COALESCE(d.division_name, ''),
			e.contract_status,
			e.resign_date,
			COALESCE(e.status, 0),
			COALESCE(e.created_by::text, ''),
			e.created_at,
			COALESCE(e.updated_by::text, ''),
			e.updated_at
		FROM employee e
		LEFT JOIN organization_roles r ON r.role_id::text = e.role_id::text
		LEFT JOIN organization_divisions d ON d.division_id::text = r.division_id::text
		WHERE %s AND %s AND COALESCE(e.status, 0) > 0
		LIMIT 1
	`, uuidExpr, orgExpr)

	if r.driver == "mysql" {
		query = fmt.Sprintf(`
			SELECT
				COALESCE(e.uuid, ''),
				COALESCE(e.employee_id, ''),
				COALESCE(e.nik, ''),
				COALESCE(e.fullname, ''),
				COALESCE(e.avatar, ''),
				COALESCE(e.phone, ''),
				e.birth_date,
				COALESCE(e.email, ''),
				COALESCE(e.address, ''),
				COALESCE(e.address_city, 0),
				e.join_date,
				COALESCE(e.role_id, ''),
				COALESCE(r.role_name, ''),
				COALESCE(r.division_id, ''),
				COALESCE(d.division_name, ''),
				e.contract_status,
				e.resign_date,
				COALESCE(e.status, 0),
				COALESCE(e.created_by, ''),
				e.created_at,
				COALESCE(e.updated_by, ''),
				e.updated_at
			FROM employee e
			LEFT JOIN organization_roles r ON r.role_id = e.role_id
			LEFT JOIN organization_divisions d ON d.division_id = r.division_id
			WHERE %s AND %s AND COALESCE(e.status, 0) > 0
			LIMIT 1
		`, uuidExpr, orgExpr)
	}

	var it model.EmployeeDetailResponse
	var birthDate sql.NullTime
	var joinDate sql.NullTime
	var contractStatus sql.NullInt64
	var resignDate sql.NullTime
	var createdAt sql.NullTime
	var updatedAt sql.NullTime

	err := database.QueryRow(r.db, query, uuid, organizationID).Scan(
		&it.UUID,
		&it.EmployeeID,
		&it.NIK,
		&it.Fullname,
		&it.Avatar,
		&it.Phone,
		&birthDate,
		&it.Email,
		&it.Address,
		&it.AddressCity,
		&joinDate,
		&it.RoleID,
		&it.RoleName,
		&it.DivisionID,
		&it.DivisionName,
		&contractStatus,
		&resignDate,
		&it.Status,
		&it.CreatedBy,
		&createdAt,
		&it.UpdatedBy,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}
	if birthDate.Valid {
		it.BirthDate = birthDate.Time.Format("2006-01-02")
	}
	if joinDate.Valid {
		it.JoinDate = joinDate.Time.Format("2006-01-02")
	}
	if contractStatus.Valid {
		v := int(contractStatus.Int64)
		it.ContractStatus = &v
	}
	if resignDate.Valid {
		v := resignDate.Time.Format("2006-01-02")
		it.ResignDate = &v
	}
	if createdAt.Valid {
		it.CreatedAt = createdAt.Time.Format("2006-01-02 15:04:05")
	}
	if updatedAt.Valid {
		it.UpdatedAt = updatedAt.Time.Format("2006-01-02 15:04:05")
	}
	return &it, nil
}

func (r *OrganizationRepository) CreateEmployee(organizationID, createdBy string, req *model.CreateEmployeeRequest) (string, error) {
	id := uuid.New().String()
	now := time.Now()

	var birth interface{}
	if v := strings.TrimSpace(req.BirthDate); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			birth = t
		} else {
			return "", fmt.Errorf("invalid birth_date")
		}
	}
	var join interface{}
	if v := strings.TrimSpace(req.JoinDate); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			join = t
		} else {
			return "", fmt.Errorf("invalid join_date")
		}
	}
	var resign interface{}
	if req.ResignDate != nil {
		if v := strings.TrimSpace(*req.ResignDate); v != "" {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				resign = t
			} else {
				return "", fmt.Errorf("invalid resign_date")
			}
		}
	}

	query := fmt.Sprintf(`
		INSERT INTO employee
			(uuid, employee_id, nik, fullname, avatar, phone, birth_date, email, address, address_city, join_date, role_id, contract_status, resign_date, organization_id, created_at, created_by, status)
		VALUES
			(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10),
		r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14), r.getPlaceholder(15), r.getPlaceholder(16),
		r.getPlaceholder(17))

	_, err := database.Exec(
		r.db,
		query,
		id,
		req.EmployeeID,
		req.NIK,
		req.Fullname,
		req.Avatar,
		req.Phone,
		birth,
		req.Email,
		req.Address,
		req.AddressCity,
		join,
		req.RoleID,
		req.ContractStatus,
		resign,
		organizationID,
		now,
		createdBy,
	)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *OrganizationRepository) UpdateEmployee(organizationID, updatedBy string, req *model.UpdateEmployeeRequest) error {
	now := time.Now()

	var birth interface{}
	if v := strings.TrimSpace(req.BirthDate); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			birth = t
		} else {
			return fmt.Errorf("invalid birth_date")
		}
	}
	var join interface{}
	if v := strings.TrimSpace(req.JoinDate); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			join = t
		} else {
			return fmt.Errorf("invalid join_date")
		}
	}
	var resign interface{}
	if req.ResignDate != nil {
		if v := strings.TrimSpace(*req.ResignDate); v != "" {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				resign = t
			} else {
				return fmt.Errorf("invalid resign_date")
			}
		}
	}

	orgExpr := "organization_id = " + r.getPlaceholder(1)
	uuidExpr := "uuid = " + r.getPlaceholder(2)
	if r.driver != "mysql" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(1)
		uuidExpr = "uuid::text = " + r.getPlaceholder(2)
	}

	setParts := make([]string, 0, 16)
	args := make([]interface{}, 0, 20)

	pos := 3
	setParts = append(setParts, "employee_id = "+r.getPlaceholder(pos))
	args = append(args, req.EmployeeID)
	pos++
	setParts = append(setParts, "nik = "+r.getPlaceholder(pos))
	args = append(args, req.NIK)
	pos++
	setParts = append(setParts, "fullname = "+r.getPlaceholder(pos))
	args = append(args, req.Fullname)
	pos++
	setParts = append(setParts, "avatar = "+r.getPlaceholder(pos))
	args = append(args, req.Avatar)
	pos++
	setParts = append(setParts, "phone = "+r.getPlaceholder(pos))
	args = append(args, req.Phone)
	pos++
	setParts = append(setParts, "birth_date = "+r.getPlaceholder(pos))
	args = append(args, birth)
	pos++
	setParts = append(setParts, "email = "+r.getPlaceholder(pos))
	args = append(args, req.Email)
	pos++
	setParts = append(setParts, "address = "+r.getPlaceholder(pos))
	args = append(args, req.Address)
	pos++
	setParts = append(setParts, "address_city = "+r.getPlaceholder(pos))
	args = append(args, req.AddressCity)
	pos++
	setParts = append(setParts, "join_date = "+r.getPlaceholder(pos))
	args = append(args, join)
	pos++
	setParts = append(setParts, "role_id = "+r.getPlaceholder(pos))
	args = append(args, req.RoleID)
	pos++

	if req.ContractStatus != nil {
		setParts = append(setParts, "contract_status = "+r.getPlaceholder(pos))
		args = append(args, req.ContractStatus)
		pos++
	}
	if req.ResignDate != nil {
		setParts = append(setParts, "resign_date = "+r.getPlaceholder(pos))
		args = append(args, resign)
		pos++
	}
	if req.Status != nil {
		setParts = append(setParts, "status = "+r.getPlaceholder(pos))
		args = append(args, req.Status)
		pos++
	}

	setParts = append(setParts, "updated_at = "+r.getPlaceholder(pos))
	args = append(args, now)
	pos++
	setParts = append(setParts, "updated_by = "+r.getPlaceholder(pos))
	args = append(args, updatedBy)
	pos++

	query := fmt.Sprintf(`
		UPDATE employee
		SET %s
		WHERE %s AND %s AND COALESCE(status, 0) > 0
	`, strings.Join(setParts, ", "), uuidExpr, orgExpr)

	args = append([]interface{}{organizationID, req.UUID}, args...)

	res, err := database.Exec(r.db, query, args...)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *OrganizationRepository) DeactivateEmployeeByEmployeeID(organizationID, updatedBy, employeeID string) error {
	now := time.Now()

	orgExpr := "organization_id = " + r.getPlaceholder(1)

	query := fmt.Sprintf(`
		UPDATE employee
		SET status = 0,
		    updated_at = %s,
		    updated_by = %s
		WHERE %s AND uuid = %s AND COALESCE(status, 0) > 0
	`, r.getPlaceholder(3), r.getPlaceholder(4), orgExpr, r.getPlaceholder(2))

	res, err := database.Exec(r.db, query, organizationID, employeeID, now, updatedBy)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *OrganizationRepository) EmployeeShiftSchedule(organizationID, roleID, divisionID string, startDate, endDate time.Time) ([]model.EmployeeShiftScheduleRow, error) {
	orgExpr := "e.organization_id = " + r.getPlaceholder(1)
	shiftJoinEmployeeExpr := "es.employee_id = e.uuid"
	shiftJoinOrgExpr := "es.organization_id = e.organization_id"
	shiftJoinDateExpr := "es.shift_date BETWEEN " + r.getPlaceholder(2) + " AND " + r.getPlaceholder(3)
	roleJoinExpr := "r.role_id = e.role_id"

	if r.driver != "mysql" {
		orgExpr = "e.organization_id::text = " + r.getPlaceholder(1)
		shiftJoinEmployeeExpr = "es.employee_id::text = e.uuid::text"
		shiftJoinOrgExpr = "es.organization_id::text = e.organization_id::text"
		roleJoinExpr = "r.role_id::text = e.role_id::text"
	}

	whereParts := make([]string, 0, 4)
	whereParts = append(whereParts, orgExpr)
	whereParts = append(whereParts, "COALESCE(e.status, 0) > 0")

	args := make([]interface{}, 0, 5)
	args = append(args, organizationID, startDate, endDate)
	pos := 4

	if strings.TrimSpace(roleID) != "" {
		roleExpr := "e.role_id = " + r.getPlaceholder(pos)
		whereParts = append(whereParts, roleExpr)
		args = append(args, roleID)
		pos++
	}

	if strings.TrimSpace(divisionID) != "" {
		divisionExpr := "r.division_id = " + r.getPlaceholder(pos)
		if r.driver != "mysql" {
			divisionExpr = "r.division_id::text = " + r.getPlaceholder(pos)
		}
		whereParts = append(whereParts, divisionExpr)
		args = append(args, divisionID)
		pos++
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(e.uuid::text, ''),
			COALESCE(e.employee_id, ''),
			COALESCE(e.fullname, ''),
			COALESCE(e.avatar, ''),
			COALESCE(r.role_name, ''),
			COALESCE(es.shift_id::text, ''),
			es.shift_date,
			es.shift_type
		FROM employee e
		LEFT JOIN organization_roles r ON %s
		LEFT JOIN employee_shift es ON %s AND %s AND %s
		WHERE %s
		ORDER BY e.fullname ASC, es.shift_date ASC
	`, roleJoinExpr, shiftJoinEmployeeExpr, shiftJoinOrgExpr, shiftJoinDateExpr, strings.Join(whereParts, " AND "))

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.EmployeeShiftScheduleRow, 0)
	for rows.Next() {
		var it model.EmployeeShiftScheduleRow
		var shiftDate sql.NullTime
		var shiftType sql.NullInt64
		if err := rows.Scan(
			&it.UUID,
			&it.EmployeeID,
			&it.Fullname,
			&it.Avatar,
			&it.RoleName,
			&it.ShiftID,
			&shiftDate,
			&shiftType,
		); err != nil {
			return nil, err
		}
		if shiftDate.Valid {
			it.ShiftDate = shiftDate.Time.Format("2006-01-02")
		}
		if shiftType.Valid {
			v := int(shiftType.Int64)
			it.ShiftType = &v
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *OrganizationRepository) EmployeeShiftOffdayCounts(organizationID string, employeeIDs []string, monthStart, monthEnd time.Time) (map[string]int, error) {
	out := make(map[string]int)
	if len(employeeIDs) == 0 {
		return out, nil
	}

	orgExpr := "organization_id = " + r.getPlaceholder(1)
	employeeCol := "employee_id"
	if r.driver != "mysql" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(1)
		employeeCol = "employee_id::text"
	}

	inParts := make([]string, 0, len(employeeIDs))
	args := make([]interface{}, 0, len(employeeIDs)+3)
	args = append(args, organizationID, monthStart, monthEnd)

	for i, id := range employeeIDs {
		inParts = append(inParts, r.getPlaceholder(4+i))
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(%s, ''),
			COUNT(DISTINCT shift_date)
		FROM employee_shift
		WHERE %s
		  AND shift_date BETWEEN %s AND %s
		  AND %s IN (%s)
		GROUP BY %s
	`, employeeCol, orgExpr, r.getPlaceholder(2), r.getPlaceholder(3), employeeCol, strings.Join(inParts, ", "), employeeCol)

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var employeeID string
		var cnt int
		if err := rows.Scan(&employeeID, &cnt); err != nil {
			return nil, err
		}
		out[employeeID] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *OrganizationRepository) CreateEmployeeShiftSchedules(organizationID, createdBy string, items []model.EmployeeShiftSubmitItem) ([]string, error) {
	if len(items) == 0 {
		return []string{}, nil
	}

	now := time.Now()

	valueParts := make([]string, 0, len(items))
	args := make([]interface{}, 0, len(items)*7)
	ids := make([]string, 0, len(items))

	pos := 1
	for _, it := range items {
		shiftID := uuid.New().String()
		ids = append(ids, shiftID)

		shiftDateRaw := strings.TrimSpace(it.ShiftDate)
		if shiftDateRaw == "" {
			return nil, fmt.Errorf("invalid shift_date")
		}
		shiftDate, err := time.Parse("2006-01-02", shiftDateRaw)
		if err != nil {
			return nil, fmt.Errorf("invalid shift_date")
		}

		valueParts = append(valueParts, fmt.Sprintf("(%s,%s,%s,%s,%s,%s,%s)",
			r.getPlaceholder(pos),
			r.getPlaceholder(pos+1),
			r.getPlaceholder(pos+2),
			r.getPlaceholder(pos+3),
			r.getPlaceholder(pos+4),
			r.getPlaceholder(pos+5),
			r.getPlaceholder(pos+6),
		))
		pos += 7

		args = append(args,
			shiftID,
			organizationID,
			strings.TrimSpace(it.EmployeeID),
			shiftDate,
			it.ShiftType,
			now,
			createdBy,
		)
	}

	query := fmt.Sprintf(`
		INSERT INTO employee_shift
			(shift_id, organization_id, employee_id, shift_date, shift_type, created_at, created_by)
		VALUES %s
	`, strings.Join(valueParts, ", "))

	if _, err := database.Exec(r.db, query, args...); err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *OrganizationRepository) DeleteEmployeeShiftSchedule(organizationID, employeeID, shiftID string) error {
	orgExpr := "organization_id = " + r.getPlaceholder(1)
	employeeExpr := "employee_id = " + r.getPlaceholder(2)
	shiftExpr := "shift_id = " + r.getPlaceholder(3)
	if r.driver != "mysql" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(1)
		employeeExpr = "employee_id::text = " + r.getPlaceholder(2)
		shiftExpr = "shift_id::text = " + r.getPlaceholder(3)
	}

	query := fmt.Sprintf(`
		DELETE FROM employee_shift
		WHERE %s AND %s AND %s
	`, orgExpr, employeeExpr, shiftExpr)

	res, err := database.Exec(r.db, query, organizationID, employeeID, shiftID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
