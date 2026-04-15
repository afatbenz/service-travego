package repository

import (
	"database/sql"
	"fmt"
	"service-travego/database"
	"service-travego/model"
	"time"

	"github.com/google/uuid"
)

func (r *OrganizationRepository) ListDivisions(organizationID string) ([]model.OrganizationDivision, error) {
	orgExpr := "organization_id IN (" + r.getPlaceholder(1) + "," + r.getPlaceholder(2) + "," + r.getPlaceholder(3) + ")"
	divisionIDExpr := "division_id"
	createdByExpr := "COALESCE(created_by, '')"
	updatedByExpr := "COALESCE(updated_by, '')"
	if r.driver != "mysql" {
		orgExpr = "organization_id::text IN (" + r.getPlaceholder(1) + "," + r.getPlaceholder(2) + "," + r.getPlaceholder(3) + ")"
		divisionIDExpr = "division_id::text"
		createdByExpr = "COALESCE(created_by::text, '')"
		updatedByExpr = "COALESCE(updated_by::text, '')"
	}

	defaultOrgID := "00000000-0000-0000-0000-000000000000"
	legacyDefaultOrgID := "000"

	query := fmt.Sprintf(`
		SELECT %s AS division_id, division_name, COALESCE(description, '') AS description,
		       COALESCE(status, 0) AS status, %s AS created_by, created_at, %s AS updated_by, updated_at
		FROM organization_divisions
		WHERE %s AND COALESCE(status, 0) > 0
		ORDER BY created_at DESC
	`, divisionIDExpr, createdByExpr, updatedByExpr, orgExpr)

	rows, err := database.Query(r.db, query, organizationID, defaultOrgID, legacyDefaultOrgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.OrganizationDivision, 0)
	for rows.Next() {
		var it model.OrganizationDivision
		var createdAt time.Time
		var updatedAt sql.NullTime
		if err := rows.Scan(
			&it.DivisionID,
			&it.DivisionName,
			&it.Description,
			&it.Status,
			&it.CreatedBy,
			&createdAt,
			&it.UpdatedBy,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		it.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
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

func (r *OrganizationRepository) CreateDivision(organizationID, createdBy, divisionName, description string) (string, error) {
	id := uuid.New().String()
	now := time.Now()

	query := fmt.Sprintf(`
		INSERT INTO organization_divisions
			(division_id, division_name, description, organization_id, created_at, created_by, status)
		VALUES
			(%s, %s, %s, %s, %s, %s, 1)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))

	_, err := database.Exec(r.db, query, id, divisionName, description, organizationID, now, createdBy)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *OrganizationRepository) UpdateDivision(organizationID, updatedBy, divisionID, divisionName, description string) error {
	now := time.Now()

	orgExpr := "organization_id = " + r.getPlaceholder(6)
	divisionExpr := "division_id = " + r.getPlaceholder(5)
	if r.driver != "mysql" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(6)
		divisionExpr = "division_id::text = " + r.getPlaceholder(5)
	}

	query := fmt.Sprintf(`
		UPDATE organization_divisions
		SET division_name = %s,
		    description = %s,
		    updated_at = %s,
		    updated_by = %s
		WHERE %s AND %s AND COALESCE(status, 0) > 0
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), divisionExpr, orgExpr)

	res, err := database.Exec(r.db, query, divisionName, description, now, updatedBy, divisionID, organizationID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *OrganizationRepository) DeleteDivision(organizationID, updatedBy, divisionID string) error {
	now := time.Now()

	orgExpr := "organization_id = " + r.getPlaceholder(4)
	divisionExpr := "division_id = " + r.getPlaceholder(3)
	if r.driver != "mysql" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(4)
		divisionExpr = "division_id::text = " + r.getPlaceholder(3)
	}

	query := fmt.Sprintf(`
		UPDATE organization_divisions
		SET status = 0,
		    updated_at = %s,
		    updated_by = %s
		WHERE %s AND %s AND COALESCE(status, 0) > 0
	`, r.getPlaceholder(1), r.getPlaceholder(2), divisionExpr, orgExpr)

	res, err := database.Exec(r.db, query, now, updatedBy, divisionID, organizationID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *OrganizationRepository) DivisionExists(organizationID, divisionID string) (bool, error) {
	orgExpr := "organization_id = " + r.getPlaceholder(2)
	divisionExpr := "division_id = " + r.getPlaceholder(1)
	if r.driver != "mysql" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(2)
		divisionExpr = "division_id::text = " + r.getPlaceholder(1)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM organization_divisions
		WHERE %s AND %s AND COALESCE(status, 0) > 0
	`, divisionExpr, orgExpr)

	var cnt int
	if err := database.QueryRow(r.db, query, divisionID, organizationID).Scan(&cnt); err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (r *OrganizationRepository) ListRoles(organizationID string) ([]model.OrganizationRole, error) {
	orgExpr := "r.organization_id IN (" + r.getPlaceholder(1) + "," + r.getPlaceholder(2) + "," + r.getPlaceholder(3) + ")"
	roleIDExpr := "r.role_id"
	divisionIDExpr := "r.division_id"
	createdByExpr := "COALESCE(r.created_by, '')"
	updatedByExpr := "COALESCE(r.updated_by, '')"
	divisionNameExpr := "COALESCE(d.division_name, '')"
	joinExpr := "d.division_id = r.division_id"
	if r.driver != "mysql" {
		orgExpr = "r.organization_id::text IN (" + r.getPlaceholder(1) + "," + r.getPlaceholder(2) + "," + r.getPlaceholder(3) + ")"
		roleIDExpr = "r.role_id::text"
		divisionIDExpr = "COALESCE(r.division_id::text, '')"
		createdByExpr = "COALESCE(r.created_by::text, '')"
		updatedByExpr = "COALESCE(r.updated_by::text, '')"
		joinExpr = "d.division_id::text = r.division_id::text"
	} else {
		divisionIDExpr = "COALESCE(r.division_id, '')"
	}

	defaultOrgID := "00000000-0000-0000-0000-000000000000"
	legacyDefaultOrgID := "000"

	query := fmt.Sprintf(`
		SELECT %s AS role_id, r.role_name, COALESCE(r.description, '') AS description, %s AS division_id, %s AS division_name,
		       COALESCE(r.status, 0) AS status, %s AS created_by, r.created_at, %s AS updated_by, r.updated_at
		FROM organization_roles r
		LEFT JOIN organization_divisions d ON %s
		WHERE %s AND COALESCE(r.status, 0) > 0
		ORDER BY r.created_at DESC
	`, roleIDExpr, divisionIDExpr, divisionNameExpr, createdByExpr, updatedByExpr, joinExpr, orgExpr)

	rows, err := database.Query(r.db, query, organizationID, defaultOrgID, legacyDefaultOrgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.OrganizationRole, 0)
	for rows.Next() {
		var it model.OrganizationRole
		var createdAt time.Time
		var updatedAt sql.NullTime
		if err := rows.Scan(
			&it.RoleID,
			&it.RoleName,
			&it.Description,
			&it.DivisionID,
			&it.DivisionName,
			&it.Status,
			&it.CreatedBy,
			&createdAt,
			&it.UpdatedBy,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		it.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
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

func (r *OrganizationRepository) CreateRole(organizationID, createdBy, roleName, description, divisionID string) (string, error) {
	id := uuid.New().String()
	now := time.Now()

	query := fmt.Sprintf(`
		INSERT INTO organization_roles
			(role_id, role_name, description, organization_id, created_at, created_by, updated_at, updated_by, division_id, status)
		VALUES
			(%s, %s, %s, %s, %s, %s, %s, %s, %s, 1)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9))

	_, err := database.Exec(r.db, query, id, roleName, description, organizationID, now, createdBy, now, createdBy, divisionID)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *OrganizationRepository) UpdateRole(organizationID, updatedBy, roleID, roleName, description, divisionID string) error {
	now := time.Now()

	orgExpr := "organization_id = " + r.getPlaceholder(7)
	roleExpr := "role_id = " + r.getPlaceholder(6)
	if r.driver != "mysql" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(7)
		roleExpr = "role_id::text = " + r.getPlaceholder(6)
	}

	query := fmt.Sprintf(`
		UPDATE organization_roles
		SET role_name = %s,
		    description = %s,
		    division_id = %s,
		    updated_at = %s,
		    updated_by = %s
		WHERE %s AND %s AND COALESCE(status, 0) > 0
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), roleExpr, orgExpr)

	res, err := database.Exec(r.db, query, roleName, description, divisionID, now, updatedBy, roleID, organizationID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *OrganizationRepository) DeleteRole(organizationID, updatedBy, roleID string) error {
	now := time.Now()

	orgExpr := "organization_id = " + r.getPlaceholder(4)
	roleExpr := "role_id = " + r.getPlaceholder(3)
	if r.driver != "mysql" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(4)
		roleExpr = "role_id::text = " + r.getPlaceholder(3)
	}

	query := fmt.Sprintf(`
		UPDATE organization_roles
		SET status = 0,
		    updated_at = %s,
		    updated_by = %s
		WHERE %s AND %s AND COALESCE(status, 0) > 0
	`, r.getPlaceholder(1), r.getPlaceholder(2), roleExpr, orgExpr)

	res, err := database.Exec(r.db, query, now, updatedBy, roleID, organizationID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *OrganizationRepository) GetDivisionOrganizationID(divisionID string) (string, error) {
	orgExpr := "COALESCE(organization_id, '')"
	idExpr := "division_id = " + r.getPlaceholder(1)
	if r.driver != "mysql" {
		orgExpr = "COALESCE(organization_id::text, '')"
		idExpr = "division_id::text = " + r.getPlaceholder(1)
	}
	query := fmt.Sprintf(`
		SELECT %s AS organization_id
		FROM organization_divisions
		WHERE %s AND COALESCE(status, 0) > 0
		LIMIT 1
	`, orgExpr, idExpr)

	var orgID string
	err := database.QueryRow(r.db, query, divisionID).Scan(&orgID)
	if err != nil {
		return "", err
	}
	return orgID, nil
}

func (r *OrganizationRepository) GetRoleOrganizationID(roleID string) (string, error) {
	orgExpr := "COALESCE(organization_id, '')"
	idExpr := "role_id = " + r.getPlaceholder(1)
	if r.driver != "mysql" {
		orgExpr = "COALESCE(organization_id::text, '')"
		idExpr = "role_id::text = " + r.getPlaceholder(1)
	}
	query := fmt.Sprintf(`
		SELECT %s AS organization_id
		FROM organization_roles
		WHERE %s AND COALESCE(status, 0) > 0
		LIMIT 1
	`, orgExpr, idExpr)

	var orgID string
	err := database.QueryRow(r.db, query, roleID).Scan(&orgID)
	if err != nil {
		return "", err
	}
	return orgID, nil
}
