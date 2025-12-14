package repository

import (
	"database/sql"
	"fmt"
	"service-travego/model"
)

type OrganizationTypeRepository struct {
	db     *sql.DB
	driver string
}

func NewOrganizationTypeRepository(db *sql.DB, driver string) *OrganizationTypeRepository {
	return &OrganizationTypeRepository{
		db:     db,
		driver: driver,
	}
}

// getPlaceholder returns the appropriate placeholder for the database driver
func (r *OrganizationTypeRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

// FindAll retrieves all organization types ordered by name ascending
func (r *OrganizationTypeRepository) FindAll() ([]model.OrganizationType, error) {
	query := `
        SELECT id, name
        FROM organization_types
        ORDER BY name ASC
    `

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgTypes []model.OrganizationType
	for rows.Next() {
		var orgType model.OrganizationType
		if err := rows.Scan(&orgType.ID, &orgType.Name); err != nil {
			return nil, err
		}
		orgTypes = append(orgTypes, orgType)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orgTypes, nil
}

// FindByID retrieves an organization type by id
func (r *OrganizationTypeRepository) FindByID(id int) (*model.OrganizationType, error) {
	query := fmt.Sprintf(`
        SELECT id, name
        FROM organization_types
        WHERE id = %s
    `, r.getPlaceholder(1))

	var orgType model.OrganizationType
	err := r.db.QueryRow(query, id).Scan(&orgType.ID, &orgType.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &orgType, nil
}
