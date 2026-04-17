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
	if r.driver != "mysql" {
		orgExpr = "e.organization_id::text = " + r.getPlaceholder(1)
	}

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
				COALESCE(d.division_name, ''),
				e.contract_status,
				e.resign_date,
				COALESCE(e.status, 0),
				e.created_at,
				e.updated_at
			FROM employee e
			LEFT JOIN organization_roles r ON r.role_id = e.role_id
			LEFT JOIN organization_divisions d ON d.division_id = r.division_id
			WHERE %s AND COALESCE(e.status, 0) > 0
			%s
			ORDER BY e.created_at DESC
		`, orgExpr, divisionFilter)
	}

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
