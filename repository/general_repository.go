package repository

import (
	"database/sql"
	"fmt"
	"service-travego/database"
	"service-travego/model"
)

type GeneralRepository struct {
	db     *sql.DB
	driver string
}

func NewGeneralRepository(db *sql.DB, driver string) *GeneralRepository {
	return &GeneralRepository{
		db:     db,
		driver: driver,
	}
}

func (r *GeneralRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

// GetBankList retrieves bank list
func (r *GeneralRepository) GetBankList() ([]model.Bank, error) {
	query := `
        SELECT code, name
        FROM bank_list
        ORDER BY name ASC
    `

	rows, err := database.Query(r.db, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var banks []model.Bank
	for rows.Next() {
		var bank model.Bank
		if err := rows.Scan(&bank.Code, &bank.Name); err != nil {
			return nil, err
		}
		banks = append(banks, bank)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return banks, nil
}

func (r *GeneralRepository) GetPreferenceCities(organizationID string, cityID *int, serviceType *int) ([]model.PreferenceCity, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT pc.preference_id, pc.city_id, pc.minimal_day, pc.organization_id, pc.created_at, pc.created_by
		FROM preference_cities pc
		LEFT JOIN preference_city_types pct
			ON pct.city_id = pc.city_id
			AND pct.organization_id = pc.organization_id
		WHERE pc.organization_id = %s
	`, r.getPlaceholder(1))

	args := []interface{}{organizationID}
	argPos := 2

	if cityID != nil {
		query += fmt.Sprintf(" AND pc.city_id = %s", r.getPlaceholder(argPos))
		args = append(args, *cityID)
		argPos++
	}

	if serviceType != nil {
		query += fmt.Sprintf(" AND pct.service_type = %s", r.getPlaceholder(argPos))
		args = append(args, *serviceType)
		argPos++
	}

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []model.PreferenceCity
	for rows.Next() {
		var pref model.PreferenceCity
		if err := rows.Scan(
			&pref.PreferenceID,
			&pref.CityID,
			&pref.MinimalDay,
			&pref.OrganizationID,
			&pref.CreatedAt,
			&pref.CreatedBy,
		); err != nil {
			return nil, err
		}
		prefs = append(prefs, pref)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return prefs, nil
}

func (r *GeneralRepository) GetPreferenceCityTypesByCityIDAndOrganizationID(cityID int, organizationID string) ([]int, error) {
	query := fmt.Sprintf(`
		SELECT service_type
		FROM preference_city_types
		WHERE city_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	rows, err := database.Query(r.db, query, cityID, organizationID)
	if err != nil {
		return []int{}, nil
	}
	defer rows.Close()

	var types []int
	for rows.Next() {
		var serviceType int
		if err := rows.Scan(&serviceType); err != nil {
			return []int{}, nil
		}
		types = append(types, serviceType)
	}

	if err := rows.Err(); err != nil {
		return []int{}, nil
	}

	return types, nil
}
