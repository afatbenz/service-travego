package repository

import (
	"database/sql"
	"fmt"
	"service-travego/model"
	"time"
)

type OrganizationRepository struct {
	db     *sql.DB
	driver string
}

func NewOrganizationRepository(db *sql.DB, driver string) *OrganizationRepository {
	return &OrganizationRepository{
		db:     db,
		driver: driver,
	}
}

// getPlaceholder returns the appropriate placeholder for the database driver
func (r *OrganizationRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

// FindByID retrieves an organization by ID from database
func (r *OrganizationRepository) FindByID(id string) (*model.Organization, error) {
	query := fmt.Sprintf(`
		SELECT id, organization_code, organization_name, company_name, address, city, province,
		       phone, email, user_id, created_at, updated_at
		FROM organizations
		WHERE id = %s
	`, r.getPlaceholder(1))

	var org model.Organization
	err := r.db.QueryRow(query, id).Scan(
		&org.ID,
		&org.OrganizationCode,
		&org.OrganizationName,
		&org.CompanyName,
		&org.Address,
		&org.City,
		&org.Province,
		&org.Phone,
		&org.Email,
		&org.UserID,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &org, nil
}

// FindByCode retrieves an organization by code from database
func (r *OrganizationRepository) FindByCode(code string) (*model.Organization, error) {
	query := fmt.Sprintf(`
		SELECT id, organization_code, organization_name, company_name, address, city, province,
		       phone, email, user_id, created_at, updated_at
		FROM organizations
		WHERE organization_code = %s
	`, r.getPlaceholder(1))

	var org model.Organization
	err := r.db.QueryRow(query, code).Scan(
		&org.ID,
		&org.OrganizationCode,
		&org.OrganizationName,
		&org.CompanyName,
		&org.Address,
		&org.City,
		&org.Province,
		&org.Phone,
		&org.Email,
		&org.UserID,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &org, nil
}

// FindByUserID retrieves all organizations by user ID from database
func (r *OrganizationRepository) FindByUserID(userID string) ([]model.Organization, error) {
	query := fmt.Sprintf(`
		SELECT id, organization_code, organization_name, company_name, address, city, province,
		       phone, email, user_id, created_at, updated_at
		FROM organizations
		WHERE user_id = %s
		ORDER BY created_at DESC
	`, r.getPlaceholder(1))

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []model.Organization
	for rows.Next() {
		var org model.Organization
		err := rows.Scan(
			&org.ID,
			&org.OrganizationCode,
			&org.OrganizationName,
			&org.CompanyName,
			&org.Address,
			&org.City,
			&org.Province,
			&org.Phone,
			&org.Email,
			&org.UserID,
			&org.CreatedAt,
			&org.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orgs, nil
}

// Create inserts a new organization into database
func (r *OrganizationRepository) Create(org *model.Organization) (*model.Organization, error) {
	now := time.Now()
	org.CreatedAt = now
	org.UpdatedAt = now

	if r.driver == "postgres" {
		query := fmt.Sprintf(`
			INSERT INTO organizations (id, organization_code, organization_name, company_name, address, 
			                          city, province, phone, email, user_id, created_at, updated_at)
			VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
			RETURNING created_at, updated_at
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12))

		err := r.db.QueryRow(
			query,
			org.ID,
			org.OrganizationCode,
			org.OrganizationName,
			org.CompanyName,
			org.Address,
			org.City,
			org.Province,
			org.Phone,
			org.Email,
			org.UserID,
			org.CreatedAt,
			org.UpdatedAt,
		).Scan(&org.CreatedAt, &org.UpdatedAt)

		if err != nil {
			return nil, err
		}
	} else {
		query := fmt.Sprintf(`
			INSERT INTO organizations (id, organization_code, organization_name, company_name, address,
			                          city, province, phone, email, user_id, created_at, updated_at)
			VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12))

		_, err := r.db.Exec(
			query,
			org.ID,
			org.OrganizationCode,
			org.OrganizationName,
			org.CompanyName,
			org.Address,
			org.City,
			org.Province,
			org.Phone,
			org.Email,
			org.UserID,
			org.CreatedAt,
			org.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
	}

	return org, nil
}

// Update updates an existing organization in database
func (r *OrganizationRepository) Update(org *model.Organization) (*model.Organization, error) {
	org.UpdatedAt = time.Now()

	if r.driver == "postgres" {
		query := fmt.Sprintf(`
			UPDATE organizations
			SET organization_name = %s, company_name = %s, address = %s, city = %s, province = %s,
			    phone = %s, email = %s, updated_at = %s
			WHERE id = %s
			RETURNING organization_code, user_id, created_at
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9))

		err := r.db.QueryRow(
			query,
			org.OrganizationName,
			org.CompanyName,
			org.Address,
			org.City,
			org.Province,
			org.Phone,
			org.Email,
			org.UpdatedAt,
			org.ID,
		).Scan(&org.OrganizationCode, &org.UserID, &org.CreatedAt)

		if err != nil {
			if err == sql.ErrNoRows {
				return nil, sql.ErrNoRows
			}
			return nil, err
		}
	} else {
		query := fmt.Sprintf(`
			UPDATE organizations
			SET organization_name = %s, company_name = %s, address = %s, city = %s, province = %s,
			    phone = %s, email = %s, updated_at = %s
			WHERE id = %s
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9))

		result, err := r.db.Exec(
			query,
			org.OrganizationName,
			org.CompanyName,
			org.Address,
			org.City,
			org.Province,
			org.Phone,
			org.Email,
			org.UpdatedAt,
			org.ID,
		)
		if err != nil {
			return nil, err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}

		if rowsAffected == 0 {
			return nil, sql.ErrNoRows
		}

		// Fetch updated data
		err = r.db.QueryRow(fmt.Sprintf(`
			SELECT organization_code, user_id, created_at 
			FROM organizations WHERE id = %s
		`, r.getPlaceholder(1)), org.ID).Scan(&org.OrganizationCode, &org.UserID, &org.CreatedAt)
		if err != nil {
			return nil, err
		}
	}

	return org, nil
}

// Delete deletes an organization from database
func (r *OrganizationRepository) Delete(id string) error {
	query := fmt.Sprintf(`DELETE FROM organizations WHERE id = %s`, r.getPlaceholder(1))

	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
