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

type GarageRepository struct {
	db     *sql.DB
	driver string
}

func NewGarageRepository(db *sql.DB, driver string) *GarageRepository {
	return &GarageRepository{
		db:     db,
		driver: driver,
	}
}

func (r *GarageRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

func (r *GarageRepository) GetAll(organizationID string) ([]model.Garage, error) {
	query := fmt.Sprintf(`
		SELECT garage_id, organization_id, garage_name, garage_address, garage_city,
		       created_at, created_by, updated_at, updated_by
		FROM garage
		WHERE organization_id = %s
		ORDER BY created_at DESC
	`, r.getPlaceholder(1))

	rows, err := database.Query(r.db, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var garages []model.Garage
	for rows.Next() {
		var g model.Garage
		err := rows.Scan(
			&g.GarageID,
			&g.OrganizationID,
			&g.GarageName,
			&g.GarageAddress,
			&g.GarageCity,
			&g.CreatedAt,
			&g.CreatedBy,
			&g.UpdatedAt,
			&g.UpdatedBy,
		)
		if err != nil {
			return nil, err
		}
		garages = append(garages, g)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return garages, nil
}

func (r *GarageRepository) GetByID(garageID, organizationID string) (*model.Garage, error) {
	query := fmt.Sprintf(`
		SELECT garage_id, organization_id, garage_name, garage_address, garage_city,
		       created_at, created_by, updated_at, updated_by
		FROM garage
		WHERE garage_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	var g model.Garage
	err := database.QueryRow(r.db, query, garageID, organizationID).Scan(
		&g.GarageID,
		&g.OrganizationID,
		&g.GarageName,
		&g.GarageAddress,
		&g.GarageCity,
		&g.CreatedAt,
		&g.CreatedBy,
		&g.UpdatedAt,
		&g.UpdatedBy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &g, nil
}

func (r *GarageRepository) Create(garage *model.Garage) error {
	garage.GarageID = uuid.New().String()
	now := time.Now()
	garage.CreatedAt = now
	garage.UpdatedAt = now

	query := fmt.Sprintf(`
		INSERT INTO garage (
			organization_id, garage_id, garage_name, garage_address, garage_city,
			created_at, created_by, updated_at, updated_by
		) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9),
	)

	_, err := database.Exec(r.db, query,
		garage.OrganizationID,
		garage.GarageID,
		garage.GarageName,
		garage.GarageAddress,
		garage.GarageCity,
		garage.CreatedAt,
		garage.CreatedBy,
		garage.UpdatedAt,
		garage.UpdatedBy,
	)

	return err
}

func (r *GarageRepository) Update(garageID, organizationID string, updates map[string]interface{}) error {
	now := time.Now()
	updates["updated_at"] = now

	var setParts []string
	var args []interface{}
	pos := 1

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = %s", key, r.getPlaceholder(pos)))
		args = append(args, value)
		pos++
	}

	query := fmt.Sprintf("UPDATE garage SET %s WHERE garage_id = %s AND organization_id = %s",
		strings.Join(setParts, ", "),
		r.getPlaceholder(pos),
		r.getPlaceholder(pos+1),
	)

	args = append(args, garageID, organizationID)

	_, err := database.Exec(r.db, query, args...)
	return err
}

func (r *GarageRepository) Delete(garageID, organizationID string) error {
	query := fmt.Sprintf("UPDATE garage SET status = 0 WHERE garage_id = %s AND organization_id = %s",
		r.getPlaceholder(1), r.getPlaceholder(2))

	result, err := database.Exec(r.db, query, garageID, organizationID)
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
