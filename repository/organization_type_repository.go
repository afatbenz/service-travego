package repository

import (
	"database/sql"
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
