package repository

import (
	"database/sql"
	"fmt"
	"service-travego/database"
	"service-travego/model"
	"time"

	"github.com/google/uuid"
)

type PreferenceCityRepository struct {
	db     *sql.DB
	driver string
}

func NewPreferenceCityRepository(db *sql.DB, driver string) *PreferenceCityRepository {
	return &PreferenceCityRepository{
		db:     db,
		driver: driver,
	}
}

func (r *PreferenceCityRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

func (r *PreferenceCityRepository) Create(cityID int, minimalDay int, organizationID string, createdBy string, serviceTypes []int) error {
	var existingPreferenceID string
	checkQuery := fmt.Sprintf(`
		SELECT preference_id
		FROM preference_cities
		WHERE city_id = %s AND organization_id = %s
	`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
	)
	err := database.QueryRow(r.db, checkQuery, cityID, organizationID).Scan(&existingPreferenceID)
	if err == nil {
		return fmt.Errorf("preference city with city_id %d already exists for this organization", cityID)
	}

	now := time.Now()
	query := fmt.Sprintf(`
		INSERT INTO preference_cities (preference_id, city_id, minimal_day, organization_id, created_at, created_by)
		VALUES (%s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
		r.getPlaceholder(3),
		r.getPlaceholder(4),
		r.getPlaceholder(5),
		r.getPlaceholder(6),
	)
	_, err = database.Exec(r.db, query,
		uuid.New().String(),
		cityID,
		minimalDay,
		organizationID,
		now,
		createdBy,
	)
	if err != nil {
		return err
	}

	if len(serviceTypes) > 0 {
		return r.CreateTypes(cityID, organizationID, serviceTypes)
	}

	return nil
}

func (r *PreferenceCityRepository) CreateTypes(cityID int, organizationID string, serviceTypes []int) error {
	for _, st := range serviceTypes {
		query := fmt.Sprintf(`
			INSERT INTO preference_city_types (preference_type_id, city_id, service_type, organization_id)
			VALUES (%s, %s, %s, %s)
		`,
			r.getPlaceholder(1),
			r.getPlaceholder(2),
			r.getPlaceholder(3),
			r.getPlaceholder(4),
		)
		_, err := database.Exec(r.db, query, uuid.New().String(), cityID, st, organizationID)
		if err != nil {
			fmt.Println("Error insert into preference_city_types ", err)
			return err
		}
	}
	return nil
}

func (r *PreferenceCityRepository) Update(preferenceID string, cityID int, minimalDay int, organizationID string, serviceTypeIDs []int) error {
	var oldCityID int
	query := fmt.Sprintf(`
		SELECT city_id
		FROM preference_cities
		WHERE preference_id = %s AND organization_id = %s
	`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
	)
	err := database.QueryRow(r.db, query, preferenceID, organizationID).Scan(&oldCityID)
	if err != nil {
		return fmt.Errorf("failed to find existing preference city: %w", err)
	}

	query = fmt.Sprintf(`
		UPDATE preference_cities
		SET city_id = %s, minimal_day = %s
		WHERE preference_id = %s AND organization_id = %s
	`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
		r.getPlaceholder(3),
		r.getPlaceholder(4),
	)
	_, err = database.Exec(r.db, query,
		cityID,
		minimalDay,
		preferenceID,
		organizationID,
	)
	if err != nil {
		return err
	}

	_ = r.DeleteTypesByCityIDAndOrganizationID(oldCityID, organizationID)

	if oldCityID != cityID {
		_ = r.DeleteTypesByCityIDAndOrganizationID(cityID, organizationID)
	}

	if len(serviceTypeIDs) > 0 {
		return r.CreateTypes(cityID, organizationID, serviceTypeIDs)
	}

	return nil
}

func (r *PreferenceCityRepository) Delete(preferenceID, organizationID string) error {
	query := fmt.Sprintf(`
		SELECT city_id
		FROM preference_cities
		WHERE preference_id = %s AND organization_id = %s
	`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
	)
	var cityID int
	err := database.QueryRow(r.db, query, preferenceID, organizationID).Scan(&cityID)
	if err == nil {
		_ = r.DeleteTypesByCityIDAndOrganizationID(cityID, organizationID)
	}

	query = fmt.Sprintf(`
		DELETE FROM preference_cities
		WHERE preference_id = %s AND organization_id = %s
	`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
	)
	_, err = database.Exec(r.db, query, preferenceID, organizationID)
	return err
}

func (r *PreferenceCityRepository) DeleteTypesByCityIDAndOrganizationID(cityID int, organizationID string) error {
	query := fmt.Sprintf(`
		DELETE FROM preference_city_types
		WHERE city_id = %s AND organization_id = %s
	`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
	)
	_, err := database.Exec(r.db, query, cityID, organizationID)
	return err
}

func (r *PreferenceCityRepository) DeleteByCityAndServiceType(cityID int, serviceType int, organizationID string) error {
	_ = r.DeleteTypesByCityIDAndOrganizationID(cityID, organizationID)

	query := fmt.Sprintf(`
		DELETE FROM preference_cities
		WHERE city_id = %s AND organization_id = %s
	`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
	)
	_, err := database.Exec(r.db, query, cityID, organizationID)
	return err
}

func (r *PreferenceCityRepository) GetAll(organizationID string, cityID *int) ([]model.PreferenceCity, error) {
	query := fmt.Sprintf(`
		SELECT preference_id, city_id, minimal_day, organization_id, created_at, created_by
		FROM preference_cities
		WHERE organization_id = %s
	`,
		r.getPlaceholder(1),
	)
	args := []interface{}{organizationID}
	argPos := 2

	if cityID != nil {
		query += fmt.Sprintf(" AND city_id = %s", r.getPlaceholder(argPos))
		args = append(args, *cityID)
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

func (r *PreferenceCityRepository) GetTypesByCityIDAndOrganizationID(cityID int, organizationID string) ([]int, error) {
	query := fmt.Sprintf(`
		SELECT service_type
		FROM preference_city_types
		WHERE city_id = %s AND organization_id = %s
	`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
	)
	rows, err := database.Query(r.db, query, cityID, organizationID)
	if err != nil {
		return []int{}, nil
	}
	defer rows.Close()

	var types []int
	for rows.Next() {
		var t int
		if err := rows.Scan(&t); err != nil {
			return []int{}, nil
		}
		types = append(types, t)
	}
	if err := rows.Err(); err != nil {
		return []int{}, nil
	}
	return types, nil
}
