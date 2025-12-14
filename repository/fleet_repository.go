package repository

import (
	"database/sql"
	"fmt"
	"service-travego/model"
	"time"

	"github.com/google/uuid"
)

type FleetRepository struct {
	db     *sql.DB
	driver string
}

func NewFleetRepository(db *sql.DB, driver string) *FleetRepository {
	return &FleetRepository{db: db, driver: driver}
}

func (r *FleetRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

func (r *FleetRepository) CreateFleetWithDetails(uuid, createdBy, organizationID string, req *model.CreateFleetRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()

	fleetQuery := fmt.Sprintf(`
        INSERT INTO fleets (uuid, fleet_name, fleet_type, capacity, production_year, engine, body, description, thumbnail, created_at, created_by, organization_id, active, status)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1)
    `,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10),
		r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13),
	)

	_, err = tx.Exec(
		fleetQuery,
		uuid,
		req.FleetName,
		req.FleetType,
		req.Capacity,
		req.ProductionYear,
		req.Engine,
		req.Body,
		req.Description,
		req.Thumbnail,
		now,
		createdBy,
		organizationID,
		req.Active,
	)
	if err != nil {
		return err
	}

	if len(req.PickupPoint) > 0 {
		pickupQuery := fmt.Sprintf(`
            INSERT INTO fleet_pickup (uuid, fleet_id, city_id, created_by, organization_id, created_at, updated_by, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		)
		for _, cityID := range req.PickupPoint {
			_, err = tx.Exec(pickupQuery, uuid, uuid, cityID, createdBy, organizationID, now, createdBy, now)
			if err != nil {
				return err
			}
		}
	}

	if len(req.Facilities) > 0 {
		facQuery := fmt.Sprintf(`
            INSERT INTO fleet_facilities (uuid, fleet_id, facility, created_by, organization_id, created_at, updated_by, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		)
		for _, facility := range req.Facilities {
			_, err = tx.Exec(facQuery, uuid, uuid, facility, createdBy, organizationID, now, createdBy, now)
			if err != nil {
				return err
			}
		}
	}

	if len(req.Prices) > 0 {
		priceQuery := fmt.Sprintf(`
            INSERT INTO fleet_price (uuid, fleet_id, duration, rent_type, price, created_by, organization_id, created_at, updated_by, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10),
		)
		for _, p := range req.Prices {
			_, err = tx.Exec(priceQuery, uuid, uuid, p.Duration, p.RentCategory, p.Price, createdBy, organizationID, now, createdBy, now)
			if err != nil {
				return err
			}
		}
	}

	if len(req.Addon) > 0 {
		addonQuery := fmt.Sprintf(`
            INSERT INTO fleet_addon (uuid, fleet_id, addon_name, addon_desc, addon_price, created_by, organization_id, created_at, updated_by, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10),
		)
		for _, a := range req.Addon {
			_, err = tx.Exec(addonQuery, uuid, uuid, a.AddonName, a.Description, a.Price, createdBy, organizationID, now, createdBy, now)
			if err != nil {
				return err
			}
		}
	}

	if len(req.BodyImages) > 0 {
		imgQuery := fmt.Sprintf(`
            INSERT INTO fleet_images (uuid, fleet_id, path_file)
            VALUES (%s, %s, %s)
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3),
		)
		for _, path := range req.BodyImages {
			if path == "" {
				continue
			}
			_, err = tx.Exec(imgQuery, uuid2(), uuid, path)
			if err != nil {
				return err
			}
		}
	}

	err = tx.Commit()
	return err
}

func uuid2() string { return uuid.New().String() }
