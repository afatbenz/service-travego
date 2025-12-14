package repository

import (
	"database/sql"
	"fmt"
	"log"
	"service-travego/model"
	"strings"
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

	argsFleet := []interface{}{uuid, req.FleetName, req.FleetType, req.Capacity, req.ProductionYear, req.Engine, req.Body, req.Description, req.Thumbnail, now, createdBy, organizationID, req.Active}
	_, err = tx.Exec(fleetQuery, argsFleet...)
	if err != nil {
		log.Printf("[ERROR] Insert fleets failed - driver=%s, err=%v\nSQL: %s\nArgs: %#v", r.driver, err, fleetQuery, argsFleet)
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
			pu := uuid2()
			args := []interface{}{pu, uuid, cityID, createdBy, organizationID, now, createdBy, now}
			_, err = tx.Exec(pickupQuery, args...)
			if err != nil {
				log.Printf("[ERROR] Insert fleet_pickup failed - driver=%s, err=%v\nSQL: %s\nArgs: %#v", r.driver, err, pickupQuery, args)
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
			fu := uuid2()
			args := []interface{}{fu, uuid, facility, createdBy, organizationID, now, createdBy, now}
			_, err = tx.Exec(facQuery, args...)
			if err != nil {
				log.Printf("[ERROR] Insert fleet_facilities failed - driver=%s, err=%v\nSQL: %s\nArgs: %#v", r.driver, err, facQuery, args)
				return err
			}
		}
	}

	if len(req.Prices) > 0 {
		priceQuery := fmt.Sprintf(`
            INSERT INTO fleet_prices (uuid, fleet_id, duration, rent_type, price, created_by, organization_id, created_at, updated_by, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10),
		)
		for _, p := range req.Prices {
			pru := uuid2()
			args := []interface{}{pru, uuid, p.Duration, p.RentCategory, p.Price, createdBy, organizationID, now, createdBy, now}
			_, err = tx.Exec(priceQuery, args...)
			if err != nil {
				log.Printf("[ERROR] Insert fleet_prices failed - driver=%s, err=%v\nSQL: %s\nArgs: %#v", r.driver, err, priceQuery, args)
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
			au := uuid2()
			args := []interface{}{au, uuid, a.AddonName, a.Description, a.Price, createdBy, organizationID, now, createdBy, now}
			_, err = tx.Exec(addonQuery, args...)
			if err != nil {
				log.Printf("[ERROR] Insert fleet_addon failed - driver=%s, err=%v\nSQL: %s\nArgs: %#v", r.driver, err, addonQuery, args)
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
			iu := uuid2()
			args := []interface{}{iu, uuid, path}
			_, err = tx.Exec(imgQuery, args...)
			if err != nil {
				log.Printf("[ERROR] Insert fleet_images failed - driver=%s, err=%v\nSQL: %s\nArgs: %#v", r.driver, err, imgQuery, args)
				return err
			}
		}
	}

	err = tx.Commit()
	return err
}

func uuid2() string { return uuid.New().String() }

func (r *FleetRepository) ListFleets(req *model.ListFleetRequest) ([]model.FleetListItem, error) {
	base := `
        SELECT f.uuid AS fleet_id, ft.label AS fleet_type, f.fleet_name, f.capacity, f.engine, f.body, f.active, f.status, f.thumbnail
        FROM fleets f INNER JOIN fleet_types ft ON f.fleet_type = ft.id
    `
	where := make([]string, 0, 4)
	args := make([]interface{}, 0, 4)
	pos := 1
	if req.OrganizationID != "" {
		where = append(where, fmt.Sprintf("f.organization_id = %s", r.getPlaceholder(pos)))
		args = append(args, req.OrganizationID)
		pos++
	}
	if req.FleetType != "" {
		where = append(where, fmt.Sprintf("f.fleet_type = %s", r.getPlaceholder(pos)))
		args = append(args, req.FleetType)
		pos++
	}
	if req.FleetBody != "" {
		where = append(where, fmt.Sprintf("f.body = %s", r.getPlaceholder(pos)))
		args = append(args, req.FleetBody)
		pos++
	}
	if req.FleetEngine != "" {
		where = append(where, fmt.Sprintf("f.engine = %s", r.getPlaceholder(pos)))
		args = append(args, req.FleetEngine)
		pos++
	}
	if req.PickupLocation > 0 {
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM fleet_pickup fp WHERE fp.fleet_id = f.uuid AND fp.city_id = %s)", r.getPlaceholder(pos)))
		args = append(args, req.PickupLocation)
		pos++
	}
	query := base
	if len(where) > 0 {
		query = query + " WHERE " + strings.Join(where, " AND ")
	}
	query = query + " ORDER BY f.created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.FleetListItem, 0)
	for rows.Next() {
		var it model.FleetListItem
		if err := rows.Scan(&it.FleetID, &it.FleetType, &it.FleetName, &it.Capacity, &it.Engine, &it.Body, &it.Active, &it.Status, &it.Thumbnail); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) GetFleetDetailMeta(orgID, fleetID string) (*model.FleetDetailMeta, error) {
	query := `
        SELECT f.uuid AS fleet_id, ft.label AS fleet_type, f.fleet_name, f.capacity, f.engine, f.body,
               COALESCE(f.description, '') AS description, f.thumbnail,
               f.created_at, u.fullname AS created_by, f.updated_at, COALESCE(u2.fullname, '') AS updated_by
        FROM fleets f
        INNER JOIN fleet_types ft ON f.fleet_type = ft.id
        INNER JOIN users u ON u.user_id = f.created_by
        LEFT JOIN users u2 ON u2.user_id = f.updated_by
        WHERE f.organization_id = %s AND f.uuid = %s
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
	row := r.db.QueryRow(query, orgID, fleetID)
	var meta model.FleetDetailMeta
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	var createdBy string
	var updatedBy string
	err := row.Scan(&meta.FleetID, &meta.FleetType, &meta.FleetName, &meta.Capacity, &meta.Engine, &meta.Body, &meta.Description, &meta.Thumbnail,
		&createdAt, &createdBy, &updatedAt, &updatedBy)
	if err != nil {
		return nil, err
	}
	if createdAt.Valid {
		meta.CreatedAt = createdAt.Time.Format(time.RFC3339)
	}
	meta.CreatedBy = createdBy
	if updatedAt.Valid {
		meta.UpdatedAt = updatedAt.Time.Format(time.RFC3339)
	} else {
		meta.UpdatedAt = ""
	}
	meta.UpdatedBy = updatedBy
	return &meta, nil
}

func (r *FleetRepository) GetFleetFacilities(fleetID string) ([]string, error) {
	query := `
        SELECT COALESCE(facility, '') AS facility FROM fleet_facilities WHERE fleet_id = %s
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1))
	rows, err := r.db.Query(query, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]string, 0)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) GetFleetPickup(orgID, fleetID string) ([]model.FleetPickupItem, error) {
	query := `
        SELECT COALESCE(uuid, '') AS uuid, COALESCE(city_id, 0) AS city_id
        FROM fleet_pickup WHERE organization_id = %s AND fleet_id = %s
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
	rows, err := r.db.Query(query, orgID, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.FleetPickupItem, 0)
	for rows.Next() {
		var it model.FleetPickupItem
		if err := rows.Scan(&it.UUID, &it.CityID); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) GetFleetAddon(orgID, fleetID string) ([]model.FleetAddonItem, error) {
	query := `
        SELECT uuid,
               COALESCE(addon_name, '') AS addon_name,
               COALESCE(addon_desc, '') AS addon_desc,
               COALESCE(addon_price, 0) AS addon_price
        FROM fleet_addon WHERE organization_id = %s AND fleet_id = %s
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
	rows, err := r.db.Query(query, orgID, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.FleetAddonItem, 0)
	for rows.Next() {
		var it model.FleetAddonItem
		if err := rows.Scan(&it.UUID, &it.AddonName, &it.AddonDesc, &it.AddonPrice); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) GetFleetPrices(orgID, fleetID string) ([]model.FleetPriceItem, error) {
	query := `
        SELECT uuid, duration, rent_type, price,
               COALESCE(disc_amount, 0) AS disc_amount,
               COALESCE(disc_price, 0)  AS disc_price
        FROM fleet_prices WHERE organization_id = %s AND fleet_id = %s
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
	rows, err := r.db.Query(query, orgID, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.FleetPriceItem, 0)
	for rows.Next() {
		var it model.FleetPriceItem
		if err := rows.Scan(&it.UUID, &it.Duration, &it.RentType, &it.Price, &it.DiscAmount, &it.DiscPrice); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) GetFleetImages(orgID, fleetID string) ([]model.FleetImageItem, error) {
	query := `
        SELECT COALESCE(uuid, '') AS uuid, COALESCE(path_file, '') AS path_file
        FROM fleet_images WHERE organization_id = %s AND fleet_id = %s
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
	rows, err := r.db.Query(query, orgID, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.FleetImageItem, 0)
	for rows.Next() {
		var it model.FleetImageItem
		if err := rows.Scan(&it.UUID, &it.PathFile); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
