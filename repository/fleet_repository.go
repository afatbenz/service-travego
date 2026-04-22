package repository

import (
	"database/sql"
	"fmt"
	"service-travego/configs"
	"service-travego/database"
	"service-travego/model"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FleetRepository struct {
	db     *sql.DB
	driver string
}

const softDeleteFleetPostgres = `
UPDATE fleets
SET status = 0, updated_at = $1, updated_by = $2
WHERE uuid = $3 AND organization_id::text = $4
`

const softDeleteFleetMySQL = `
UPDATE fleets
SET status = 0, updated_at = ?, updated_by = ?
WHERE uuid = ? AND organization_id = ?
`

const listFleetsForUnitPostgres = `
SELECT uuid, fleet_name
FROM fleets
WHERE organization_id::text = $1
ORDER BY fleet_name
`

const listFleetsForUnitPostgresSearch = `
SELECT uuid, fleet_name
FROM fleets
WHERE organization_id::text = $1 AND fleet_name ILIKE '%' || $2 || '%'
ORDER BY fleet_name
`

const listFleetsForUnitMySQL = `
SELECT uuid, fleet_name
FROM fleets
WHERE organization_id = ?
ORDER BY fleet_name
`

const listFleetsForUnitMySQLSearch = `
SELECT uuid, fleet_name
FROM fleets
WHERE organization_id = ? AND fleet_name LIKE CONCAT('%', ?, '%')
ORDER BY fleet_name
`

func NewFleetRepository(db *sql.DB, driver string) *FleetRepository {
	return &FleetRepository{
		db:     db,
		driver: driver,
	}
}

func uuid2() string { return uuid.New().String() }

func (r *FleetRepository) ListFleets(req *model.ListFleetRequest) ([]model.FleetListItem, error) {
	totalUnitExpr := "COALESCE((SELECT COUNT(*) FROM fleet_units fu WHERE fu.fleet_id = f.uuid AND fu.status = 1), 0)"
	if r.driver == "postgres" || r.driver == "pgx" {
		totalUnitExpr = "COALESCE((SELECT COUNT(*) FROM fleet_units fu WHERE fu.fleet_id::text = f.uuid::text AND fu.status = 1), 0)"
	}
	base := `
        SELECT f.uuid AS fleet_id, ft.label AS fleet_type, f.fleet_name, f.capacity, f.engine, f.body, %s as total_unit, f.active, f.status, f.thumbnail
        FROM fleets f INNER JOIN fleet_types ft ON f.fleet_type = ft.id
    `
	base = fmt.Sprintf(base, totalUnitExpr)
	where := make([]string, 0, 4)
	args := make([]interface{}, 0, 4)
	pos := 1
	where = append(where, "f.status > 0")
	if req.OrganizationID != "" {
		orgExpr := fmt.Sprintf("f.organization_id = %s", r.getPlaceholder(pos))
		if r.driver == "postgres" || r.driver == "pgx" {
			orgExpr = fmt.Sprintf("f.organization_id::text = %s", r.getPlaceholder(pos))
		}
		where = append(where, orgExpr)
		args = append(args, req.OrganizationID)
		pos++
	}
	if req.FleetType != "" {
		where = append(where, fmt.Sprintf("f.fleet_type = %s", r.getPlaceholder(pos)))
		args = append(args, req.FleetType)
		pos++
	}
	if req.FleetName != "" {
		likeExpr := "f.fleet_name LIKE " + r.getPlaceholder(pos)
		if r.driver == "postgres" || r.driver == "pgx" {
			likeExpr = "f.fleet_name ILIKE " + r.getPlaceholder(pos)
		}
		where = append(where, likeExpr)
		args = append(args, "%"+req.FleetName+"%")
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

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.FleetListItem
	for rows.Next() {
		var item model.FleetListItem
		var fleetType sql.NullString
		var engine sql.NullString
		var body sql.NullString
		var thumbnail sql.NullString
		var totalUnit int64
		if err := rows.Scan(&item.FleetID, &fleetType, &item.FleetName, &item.Capacity, &engine, &body, &totalUnit, &item.Active, &item.Status, &thumbnail); err != nil {
			return nil, err
		}
		if fleetType.Valid {
			item.FleetType = fleetType.String
		}
		if engine.Valid {
			item.Engine = engine.String
		}
		if body.Valid {
			item.Body = body.String
		}
		item.TotalUnit = int(totalUnit)
		if thumbnail.Valid {
			item.Thumbnail = thumbnail.String
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *FleetRepository) CreateFleet(req *model.CreateFleetRequest) (string, error) {
	id := uuid2()
	now := time.Now()
	query := `
        INSERT INTO fleets (uuid, organization_id, fleet_type, fleet_name, capacity, production_year, engine, body, fuel_type, description, thumbnail, active, created_at, created_by, status)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9),
		r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14), r.getPlaceholder(15))

	// Status default 1 (Active/Draft?)
	_, err := database.Exec(r.db, query,
		id,
		req.OrganizationID,
		req.FleetType,
		req.FleetName,
		req.Capacity,
		req.ProductionYear,
		req.Engine,
		req.Body,
		req.FuelType,
		req.Description,
		req.Thumbnail,
		req.Active,
		now,
		req.CreatedBy,
		1,
	)

	if err != nil {
		return "", err
	}

	// Insert facilities
	if len(req.Facilities) > 0 {
		fQuery := fmt.Sprintf("INSERT INTO fleet_facilities (uuid, fleet_id, facility) VALUES (%s, %s, %s)",
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
		for _, fac := range req.Facilities {
			fID := uuid2()
			_, err := database.Exec(r.db, fQuery, fID, id, fac)
			if err != nil {
				return "", err
			}
		}
	}

	// Insert pickup
	if len(req.Pickup) > 0 {
		pQuery := fmt.Sprintf("INSERT INTO fleet_pickup (uuid, fleet_id, organization_id, city_id) VALUES (%s, %s, %s, %s)",
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))
		for _, p := range req.Pickup {
			pID := uuid2()
			_, err := database.Exec(r.db, pQuery, pID, id, req.OrganizationID, p.CityID)
			if err != nil {
				return "", err
			}
		}
	}

	// Insert addon
	if len(req.Addon) > 0 {
		aQuery := fmt.Sprintf("INSERT INTO fleet_addon (uuid, fleet_id, organization_id, addon_name, addon_desc, addon_price) VALUES (%s, %s, %s, %s, %s, %s)",
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
		for _, a := range req.Addon {
			aID := uuid2()
			_, err := database.Exec(r.db, aQuery, aID, id, req.OrganizationID, a.AddonName, a.AddonDesc, a.AddonPrice)
			if err != nil {
				return "", err
			}
		}
	}

	// Insert pricing
	if len(req.Pricing) > 0 {
		prQuery := fmt.Sprintf("INSERT INTO fleet_prices (uuid, fleet_id, organization_id, duration, rent_type, price, disc_amount, disc_price, uom) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)",
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9))
		for _, pr := range req.Pricing {
			prID := uuid2()
			_, err := database.Exec(r.db, prQuery, prID, id, req.OrganizationID, pr.Duration, pr.RentType, pr.Price, pr.DiscAmount, pr.DiscPrice, pr.Uom)
			if err != nil {
				return "", err
			}
		}
	}

	// Insert images
	if len(req.Images) > 0 {
		iQuery := fmt.Sprintf("INSERT INTO fleet_images (uuid, fleet_id, path_file) VALUES (%s, %s, %s)",
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
		for _, img := range req.Images {
			iID := uuid2()
			_, err := database.Exec(r.db, iQuery, iID, id, img.PathFile)
			if err != nil {
				return "", err
			}
		}
	}

	return id, nil
}

func (r *FleetRepository) UpdateFleet(req *model.UpdateFleetRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	updateFleetQuery := fmt.Sprintf(
		`UPDATE fleets SET fleet_type = %s, fleet_name = %s, capacity = %s, production_year = %s, engine = %s, body = %s, fuel_type = %s, description = %s, thumbnail = %s, active = %s, updated_at = %s, updated_by = %s WHERE uuid = %s AND organization_id = %s`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
		r.getPlaceholder(3),
		r.getPlaceholder(4),
		r.getPlaceholder(5),
		r.getPlaceholder(6),
		r.getPlaceholder(7),
		r.getPlaceholder(8),
		r.getPlaceholder(9),
		r.getPlaceholder(10),
		r.getPlaceholder(11),
		r.getPlaceholder(12),
		r.getPlaceholder(13),
		r.getPlaceholder(14),
	)

	res, err := database.TxExec(
		tx,
		updateFleetQuery,
		req.FleetType,
		req.FleetName,
		req.Capacity,
		req.ProductionYear,
		req.Engine,
		req.Body,
		req.FuelType,
		req.Description,
		req.Thumbnail,
		req.Active,
		now,
		req.UpdatedBy,
		req.FleetID,
		req.OrganizationID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return sql.ErrNoRows
	}

	if req.Facilities != nil {
		keepIDs := make([]string, 0, len(req.Facilities))
		for _, it := range req.Facilities {
			if it.UUID == "" {
				newID := uuid2()
				insertQuery := fmt.Sprintf("INSERT INTO fleet_facilities (uuid, fleet_id, facility) VALUES (%s, %s, %s)", r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
				if _, err := database.TxExec(tx, insertQuery, newID, req.FleetID, it.Facility); err != nil {
					return err
				}
				keepIDs = append(keepIDs, newID)
				continue
			}
			updateQuery := fmt.Sprintf("UPDATE fleet_facilities SET facility = %s WHERE uuid = %s AND fleet_id = %s", r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
			if _, err := database.TxExec(tx, updateQuery, it.Facility, it.UUID, req.FleetID); err != nil {
				return err
			}
			keepIDs = append(keepIDs, it.UUID)
		}

		if len(keepIDs) == 0 {
			delQuery := fmt.Sprintf("DELETE FROM fleet_facilities WHERE fleet_id = %s", r.getPlaceholder(1))
			if _, err := database.TxExec(tx, delQuery, req.FleetID); err != nil {
				return err
			}
		} else {
			in := make([]string, 0, len(keepIDs))
			args := make([]interface{}, 0, 1+len(keepIDs))
			args = append(args, req.FleetID)
			for i, id := range keepIDs {
				in = append(in, r.getPlaceholder(i+2))
				args = append(args, id)
			}
			delQuery := fmt.Sprintf("DELETE FROM fleet_facilities WHERE fleet_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), strings.Join(in, ","))
			if _, err := database.TxExec(tx, delQuery, args...); err != nil {
				return err
			}
		}
	}

	if req.Pickup != nil {
		keepIDs := make([]string, 0, len(req.Pickup))
		for _, it := range req.Pickup {
			if it.UUID == "" {
				newID := uuid2()
				insertQuery := fmt.Sprintf("INSERT INTO fleet_pickup (uuid, fleet_id, organization_id, city_id) VALUES (%s, %s, %s, %s)", r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))
				if _, err := database.TxExec(tx, insertQuery, newID, req.FleetID, req.OrganizationID, it.CityID); err != nil {
					return err
				}
				keepIDs = append(keepIDs, newID)
				continue
			}
			updateQuery := fmt.Sprintf("UPDATE fleet_pickup SET city_id = %s WHERE uuid = %s AND fleet_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))
			if _, err := database.TxExec(tx, updateQuery, it.CityID, it.UUID, req.FleetID, req.OrganizationID); err != nil {
				return err
			}
			keepIDs = append(keepIDs, it.UUID)
		}

		if len(keepIDs) == 0 {
			delQuery := fmt.Sprintf("DELETE FROM fleet_pickup WHERE fleet_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := database.TxExec(tx, delQuery, req.FleetID, req.OrganizationID); err != nil {
				return err
			}
		} else {
			in := make([]string, 0, len(keepIDs))
			args := make([]interface{}, 0, 2+len(keepIDs))
			args = append(args, req.FleetID, req.OrganizationID)
			for i, id := range keepIDs {
				in = append(in, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			delQuery := fmt.Sprintf("DELETE FROM fleet_pickup WHERE fleet_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(in, ","))
			if _, err := database.TxExec(tx, delQuery, args...); err != nil {
				return err
			}
		}
	}

	if req.Addon != nil {
		keepIDs := make([]string, 0, len(req.Addon))
		for _, it := range req.Addon {
			if it.UUID == "" {
				newID := uuid2()
				insertQuery := fmt.Sprintf("INSERT INTO fleet_addon (uuid, fleet_id, organization_id, addon_name, addon_desc, addon_price) VALUES (%s, %s, %s, %s, %s, %s)",
					r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
				if _, err := database.TxExec(tx, insertQuery, newID, req.FleetID, req.OrganizationID, it.AddonName, it.AddonDesc, it.AddonPrice); err != nil {
					return err
				}
				keepIDs = append(keepIDs, newID)
				continue
			}
			updateQuery := fmt.Sprintf("UPDATE fleet_addon SET addon_name = %s, addon_desc = %s, addon_price = %s WHERE uuid = %s AND fleet_id = %s AND organization_id = %s",
				r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
			if _, err := database.TxExec(tx, updateQuery, it.AddonName, it.AddonDesc, it.AddonPrice, it.UUID, req.FleetID, req.OrganizationID); err != nil {
				return err
			}
			keepIDs = append(keepIDs, it.UUID)
		}

		if len(keepIDs) == 0 {
			delQuery := fmt.Sprintf("DELETE FROM fleet_addon WHERE fleet_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := database.TxExec(tx, delQuery, req.FleetID, req.OrganizationID); err != nil {
				return err
			}
		} else {
			in := make([]string, 0, len(keepIDs))
			args := make([]interface{}, 0, 2+len(keepIDs))
			args = append(args, req.FleetID, req.OrganizationID)
			for i, id := range keepIDs {
				in = append(in, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			delQuery := fmt.Sprintf("DELETE FROM fleet_addon WHERE fleet_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(in, ","))
			if _, err := database.TxExec(tx, delQuery, args...); err != nil {
				return err
			}
		}
	}

	if req.Pricing != nil {
		keepIDs := make([]string, 0, len(req.Pricing))
		for _, it := range req.Pricing {
			if it.UUID == "" {
				newID := uuid2()
				insertQuery := fmt.Sprintf("INSERT INTO fleet_prices (uuid, fleet_id, organization_id, duration, rent_type, price, disc_amount, disc_price, uom) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)",
					r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9))
				if _, err := database.TxExec(tx, insertQuery, newID, req.FleetID, req.OrganizationID, it.Duration, it.RentType, it.Price, it.DiscAmount, it.DiscPrice, it.Uom); err != nil {
					return err
				}
				keepIDs = append(keepIDs, newID)
				continue
			}
			updateQuery := fmt.Sprintf("UPDATE fleet_prices SET duration = %s, rent_type = %s, price = %s, disc_amount = %s, disc_price = %s, uom = %s WHERE uuid = %s AND fleet_id = %s AND organization_id = %s",
				r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9))
			if _, err := database.TxExec(tx, updateQuery, it.Duration, it.RentType, it.Price, it.DiscAmount, it.DiscPrice, it.Uom, it.UUID, req.FleetID, req.OrganizationID); err != nil {
				return err
			}
			keepIDs = append(keepIDs, it.UUID)
		}

		if len(keepIDs) == 0 {
			delQuery := fmt.Sprintf("DELETE FROM fleet_prices WHERE fleet_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := database.TxExec(tx, delQuery, req.FleetID, req.OrganizationID); err != nil {
				return err
			}
		} else {
			in := make([]string, 0, len(keepIDs))
			args := make([]interface{}, 0, 2+len(keepIDs))
			args = append(args, req.FleetID, req.OrganizationID)
			for i, id := range keepIDs {
				in = append(in, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			delQuery := fmt.Sprintf("DELETE FROM fleet_prices WHERE fleet_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(in, ","))
			if _, err := database.TxExec(tx, delQuery, args...); err != nil {
				return err
			}
		}
	}

	if req.Images != nil {
		keepIDs := make([]string, 0, len(req.Images))
		for _, it := range req.Images {
			if it.UUID == "" {
				newID := uuid2()
				insertQuery := fmt.Sprintf("INSERT INTO fleet_images (uuid, fleet_id, path_file) VALUES (%s, %s, %s)", r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
				if _, err := database.TxExec(tx, insertQuery, newID, req.FleetID, it.PathFile); err != nil {
					return err
				}
				keepIDs = append(keepIDs, newID)
				continue
			}
			updateQuery := fmt.Sprintf("UPDATE fleet_images SET path_file = %s WHERE uuid = %s AND fleet_id = %s", r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
			if _, err := database.TxExec(tx, updateQuery, it.PathFile, it.UUID, req.FleetID); err != nil {
				return err
			}
			keepIDs = append(keepIDs, it.UUID)
		}

		if len(keepIDs) == 0 {
			delQuery := fmt.Sprintf("DELETE FROM fleet_images WHERE fleet_id = %s", r.getPlaceholder(1))
			if _, err := database.TxExec(tx, delQuery, req.FleetID); err != nil {
				return err
			}
		} else {
			in := make([]string, 0, len(keepIDs))
			args := make([]interface{}, 0, 1+len(keepIDs))
			args = append(args, req.FleetID)
			for i, id := range keepIDs {
				in = append(in, r.getPlaceholder(i+2))
				args = append(args, id)
			}
			delQuery := fmt.Sprintf("DELETE FROM fleet_images WHERE fleet_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), strings.Join(in, ","))
			if _, err := database.TxExec(tx, delQuery, args...); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *FleetRepository) GetFleetDetail(id, orgID string) (*model.FleetDetailResponse, error) {
	// Main fleet data
	query := fmt.Sprintf(`
        SELECT fleet_name, fleet_type, capacity, engine, body, description, active, status, thumbnail
        FROM fleets
        WHERE uuid = %s AND organization_id = %s
    `, r.getPlaceholder(1), r.getPlaceholder(2))

	var res model.FleetDetailResponse
	res.Meta.FleetID = id
	err := database.QueryRow(r.db, query, id, orgID).Scan(
		&res.Meta.FleetName, &res.Meta.FleetType, &res.Meta.Capacity, &res.Meta.Engine,
		&res.Meta.Body, &res.Meta.Description, &res.Meta.Active, &res.Meta.Status, &res.Meta.Thumbnail,
	)
	if err != nil {
		return nil, err
	}

	// Facilities
	res.Facilities, _ = r.GetFleetFacilities(id)

	// Pickup
	res.Pickup, _ = r.GetFleetPickup(orgID, id)

	// Pricing
	res.Pricing, _ = r.GetFleetPricing(orgID, id)

	// Addon
	res.Addon, _ = r.GetFleetAddon(orgID, id)

	// Images
	res.Images, _ = r.GetFleetImages(id)

	return &res, nil
}

func (r *FleetRepository) GetFleetFacilities(fleetID string) ([]string, error) {
	query := fmt.Sprintf("SELECT facility FROM fleet_facilities WHERE fleet_id = %s", r.getPlaceholder(1))
	rows, err := database.Query(r.db, query, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var facilities []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err == nil {
			facilities = append(facilities, f)
		}
	}
	return facilities, nil
}

func (r *FleetRepository) GetFleetPricing(orgID, fleetID string) ([]model.FleetPriceItem, error) {
	query := `
        SELECT uuid, duration, rent_type, price, disc_amount, disc_price, uom
        FROM fleet_prices
        WHERE fleet_id = %s
    `
	args := []interface{}{fleetID}
	if orgID != "" {
		query += " AND organization_id = %s"
		query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
		args = append(args, orgID)
	} else {
		query = fmt.Sprintf(query, r.getPlaceholder(1))
	}

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.FleetPriceItem, 0)
	for rows.Next() {
		var it model.FleetPriceItem
		if err := rows.Scan(&it.UUID, &it.Duration, &it.RentType, &it.Price, &it.DiscAmount, &it.DiscPrice, &it.Uom); err != nil {
			return nil, err
		}
		it.RentTypeLabel = configs.RentType(it.RentType).String()
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) CreateOrder(req *model.CreateOrderRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	orderID := req.OrderID
	totalAmount := req.TotalAmount

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()

	// 1. Insert fleet_order
	orderQueryFull := fmt.Sprintf(`
		INSERT INTO fleet_orders (order_id, fleet_id, start_date, end_date, pickup_city_id, pickup_location, unit_qty, price_id, created_at, total_amount, additional_amount, status, payment_status, organization_id, additional_request)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 2, %d, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), configs.PaymentStatusWaitingPayment, r.getPlaceholder(12), r.getPlaceholder(13))

	_, _ = database.TxExec(tx, "SAVEPOINT sp_orders")
	_, err = database.TxExec(tx, orderQueryFull, orderID, req.FleetID, req.StartDate, req.EndDate, req.PickupCityID, req.PickupLocation, req.Qty, req.PriceID, now, totalAmount, req.AdditionalAmount, req.OrganizationID, req.AdditionalRequest)
	if err != nil {
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "unknown column") || strings.Contains(errMsg, "does not exist") {
			_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_orders")
			orderQueryWithRequest := fmt.Sprintf(`
				INSERT INTO fleet_orders (order_id, fleet_id, start_date, end_date, pickup_city_id, pickup_location, unit_qty, price_id, created_at, total_amount, status, payment_status, organization_id, additional_request)
				VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 2, %d, %s, %s)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
				r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), configs.PaymentStatusWaitingPayment, r.getPlaceholder(11), r.getPlaceholder(12))

			_, _ = database.TxExec(tx, "SAVEPOINT sp_orders_2")
			_, err = database.TxExec(tx, orderQueryWithRequest, orderID, req.FleetID, req.StartDate, req.EndDate, req.PickupCityID, req.PickupLocation, req.Qty, req.PriceID, now, totalAmount, req.OrganizationID, req.AdditionalRequest)
			if err != nil {
				errMsg2 := strings.ToLower(err.Error())
				if strings.Contains(errMsg2, "unknown column") || strings.Contains(errMsg2, "does not exist") {
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_orders_2")
					orderQueryLegacy := fmt.Sprintf(`
						INSERT INTO fleet_orders (order_id, fleet_id, start_date, end_date, pickup_city_id, pickup_location, unit_qty, price_id, created_at, total_amount, status, payment_status, organization_id)
						VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 2, %d, %s)
					`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
						r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), configs.PaymentStatusWaitingPayment, r.getPlaceholder(11))

					_, err = database.TxExec(tx, orderQueryLegacy, orderID, req.FleetID, req.StartDate, req.EndDate, req.PickupCityID, req.PickupLocation, req.Qty, req.PriceID, now, totalAmount, req.OrganizationID)
					if err != nil {
						fmt.Println("error create orders legacy", err)
						return err
					}
				} else {
					fmt.Println("error create orders fallback", err)
					return err
				}
			}
			_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_orders_2")
		} else {
			fmt.Println("error create orders full", err)
			return err
		}
	}
	_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_orders")

	// 2. Insert fleet_orders_customers
	custID := uuid2()
	custQuery := fmt.Sprintf(`
		INSERT INTO fleet_order_customers (customer_id, order_id, customer_name, customer_phone, customer_email, customer_address, created_at, organization_id)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8))

	_, err = database.TxExec(tx, custQuery, custID, orderID, req.Fullname, req.Phone, req.Email, req.Address, now, req.OrganizationID)
	if err != nil {
		fmt.Println("error create customer orders", err)
		return err
	}

	// 3. Insert fleet_orders_addon
	if len(req.Addons) > 0 {
		addonQuery := fmt.Sprintf(`
			INSERT INTO fleet_order_addons (order_addon_id, order_id, addon_id, addon_price, created_at)
			SELECT %s, %s, uuid, addon_price, %s FROM fleet_addon WHERE uuid = %s
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))
		for _, addonID := range req.Addons {
			id := uuid2()
			res, err := database.TxExec(tx, addonQuery, id, orderID, now, addonID)
			if err != nil {
				fmt.Println("error create addon orders", err)
				return err
			}
			rows, _ := res.RowsAffected()
			if rows == 0 {
				return fmt.Errorf("addon not found: %s", addonID)
			}
		}
	}

	// 4. Insert fleet_order_destinations
	if len(req.Destinations) > 0 {
		destQuery := fmt.Sprintf(`
			INSERT INTO fleet_order_destinations (uuid, order_id, city_id, location, created_at)
			VALUES (%s, %s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5))
		for _, dest := range req.Destinations {
			id := uuid2()
			_, err = database.TxExec(tx, destQuery, id, orderID, dest.CityID, dest.Location, now)
			if err != nil {
				fmt.Println("error create dest orders", err)
				return err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (r *FleetRepository) CreatePartnerOrder(orderID, fleetID, startDate, endDate, pickupCityID, pickupLocation string, qty int, priceID string, totalAmount, additionalAmount, discount, priceSum float64, customerID, orgID, createdBy string, itinerary []model.FleetOrderItineraryItem, addons []model.FleetOrderAddonItem, additionalRequest string, fleets []model.FleetItemRequest) error {
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

	insertWithCreatedBy := fmt.Sprintf(`
		INSERT INTO fleet_orders (order_id, fleet_id, start_date, end_date, pickup_city_id, pickup_location, unit_qty, price_id, created_at, total_amount, additional_amount, discount, price, status, payment_status, organization_id, created_by, additional_request)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 2, %d, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), configs.PaymentStatusWaitingPayment, r.getPlaceholder(14), r.getPlaceholder(15), r.getPlaceholder(16))

	_, _ = database.TxExec(tx, "SAVEPOINT sp_orders")
	_, err = database.TxExec(tx, insertWithCreatedBy, orderID, fleetID, startDate, endDate, pickupCityID, pickupLocation, qty, priceID, now, totalAmount, additionalAmount, discount, priceSum, orgID, createdBy, additionalRequest)
	if err != nil {
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "unknown column") || strings.Contains(errMsg, "does not exist") {
			_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_orders")
			insertWithoutCreatedBy := fmt.Sprintf(`
				INSERT INTO fleet_orders (order_id, fleet_id, start_date, end_date, pickup_city_id, pickup_location, unit_qty, price_id, created_at, total_amount, additional_amount, discount, price, status, payment_status, organization_id, additional_request)
				VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 2, %d, %s, %s)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
				r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), configs.PaymentStatusWaitingPayment, r.getPlaceholder(14), r.getPlaceholder(15))

			_, _ = database.TxExec(tx, "SAVEPOINT sp_orders_2")
			_, err = database.TxExec(tx, insertWithoutCreatedBy, orderID, fleetID, startDate, endDate, pickupCityID, pickupLocation, qty, priceID, now, totalAmount, additionalAmount, discount, priceSum, orgID, additionalRequest)
			if err != nil {
				errMsg2 := strings.ToLower(err.Error())
				if strings.Contains(errMsg2, "additional_request") {
					// Fallback if additional_request missing
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_orders_2")
					insertLegacy := fmt.Sprintf(`
						INSERT INTO fleet_orders (order_id, fleet_id, start_date, end_date, pickup_city_id, pickup_location, unit_qty, price_id, created_at, total_amount, additional_amount, discount, price, status, payment_status, organization_id)
						VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 2, %d, %s)
					`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
						r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), configs.PaymentStatusWaitingPayment, r.getPlaceholder(14))
					_, err = database.TxExec(tx, insertLegacy, orderID, fleetID, startDate, endDate, pickupCityID, pickupLocation, qty, priceID, now, totalAmount, additionalAmount, discount, priceSum, orgID)
					if err != nil {
						return fmt.Errorf("insert fleet_orders legacy: %w", err)
					}
				} else {
					return fmt.Errorf("insert fleet_orders without created_by: %w", err)
				}
			}
			_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_orders_2")
		} else {
			return fmt.Errorf("insert fleet_orders full: %w", err)
		}
	}
	_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_orders")

	// Insert into fleet_order_items
	if len(fleets) > 0 {
		itemQuery := fmt.Sprintf(`
			INSERT INTO fleet_order_items (order_item_id, organization_id, order_id, fleet_id, price_id, quantity, charge_amount, discount, sub_total, create_at, created_by, status)
			VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11))

		for _, f := range fleets {
			itemID := uuid2()
			// Fetch price from DB to be sure, although service should have done it
			dbPrice, _, _ := r.GetPriceByID(f.PriceID)
			subTotal := (dbPrice * float64(f.Qty)) + f.AdditionalAmount - f.Discount
			_, err = database.TxExec(tx, itemQuery, itemID, orgID, orderID, f.ArmadaID, f.PriceID, f.Qty, f.AdditionalAmount, f.Discount, subTotal, now, createdBy)
			if err != nil {
				return fmt.Errorf("insert fleet_order_items: %w", err)
			}
		}
	}

	custOrderWithCreatedBy := fmt.Sprintf(`
		INSERT INTO customer_orders (order_id, customer_id, order_type, created_at, created_by, organization_id)
		VALUES (%s, %s, 1, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5))
	_, _ = database.TxExec(tx, "SAVEPOINT sp_custorders")
	_, err = database.TxExec(tx, custOrderWithCreatedBy, orderID, customerID, now, createdBy, orgID)
	if err != nil {
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "unknown column") || strings.Contains(errMsg, "does not exist") {
			custOrderWithoutCreatedBy := fmt.Sprintf(`
				INSERT INTO customer_orders (order_id, customer_id, order_type, created_at, organization_id)
				VALUES (%s, %s, 1, %s, %s)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))
			_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_custorders")
			_, err = database.TxExec(tx, custOrderWithoutCreatedBy, orderID, customerID, now, orgID)
			if err != nil {
				return fmt.Errorf("insert customer_orders: %w", err)
			}
		} else {
			return fmt.Errorf("insert customer_orders: %w", err)
		}
	}
	_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_custorders")

	if len(itinerary) > 0 {
		mode := 0
		itineraryWithCreatedBy := fmt.Sprintf(`
			INSERT INTO fleet_order_itinerary (fleet_itinerary_id, order_id, day_num, city_id, location, organization_id, created_at, created_by)
			VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8))
		itineraryWithoutCreatedBy := fmt.Sprintf(`
			INSERT INTO fleet_order_itinerary (fleet_itinerary_id, order_id, day_num, city_id, location, organization_id, created_at)
			VALUES (%s, %s, %s, %s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7))
		destQuery := fmt.Sprintf(`
			INSERT INTO fleet_order_destinations (uuid, order_id, city_id, location, created_at)
			VALUES (%s, %s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5))

		for _, it := range itinerary {
			id := uuid2()
			for {
				_, _ = database.TxExec(tx, "SAVEPOINT sp_it")
				if mode == 0 {
					_, err = database.TxExec(tx, itineraryWithCreatedBy, id, orderID, it.Day, it.CityID, it.Destination, orgID, now, createdBy)
				} else if mode == 1 {
					_, err = database.TxExec(tx, itineraryWithoutCreatedBy, id, orderID, it.Day, it.CityID, it.Destination, orgID, now)
				} else {
					_, err = database.TxExec(tx, destQuery, id, orderID, it.CityID, it.Destination, now)
				}
				if err == nil {
					_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_it")
					break
				}
				errMsg := strings.ToLower(err.Error())
				if mode == 0 && (strings.Contains(errMsg, "unknown column") || strings.Contains(errMsg, "does not exist")) {
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_it")
					mode = 1
					continue
				}
				if mode != 2 && (strings.Contains(errMsg, "doesn't exist") || strings.Contains(errMsg, "does not exist") || strings.Contains(errMsg, "relation") || strings.Contains(errMsg, "unknown table")) {
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_it")
					mode = 2
					continue
				}
				return fmt.Errorf("insert itinerary: %w", err)
			}
		}
	}

	if len(addons) > 0 {
		mode := 0
		addonWithCreatedBy := fmt.Sprintf(`
			INSERT INTO fleet_order_addons (order_addon_id, order_id, organization_id, addon_id, addon_price, addon_qty, created_at, created_by)
			SELECT %s, %s, %s, uuid, addon_price, %s, %s, %s FROM fleet_addon WHERE uuid = %s
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7))
		addonWithoutCreatedBy := fmt.Sprintf(`
			INSERT INTO fleet_order_addons (order_addon_id, order_id, organization_id, addon_id, addon_price, addon_qty, created_at)
			SELECT %s, %s, %s, uuid, addon_price, %s, %s FROM fleet_addon WHERE uuid = %s
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
		addonLegacy := fmt.Sprintf(`
			INSERT INTO fleet_order_addons (order_addon_id, order_id, addon_id, addon_price, created_at)
			SELECT %s, %s, uuid, addon_price, %s FROM fleet_addon WHERE uuid = %s
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))

		for _, a := range addons {
			if a.AddonID == "" {
				continue
			}
			addonQty := a.Quantity
			if addonQty <= 0 {
				addonQty = 1
			}
			id := uuid2()
			var res sql.Result
			for {
				_, _ = database.TxExec(tx, "SAVEPOINT sp_addon")
				if mode == 0 {
					res, err = database.TxExec(tx, addonWithCreatedBy, id, orderID, orgID, addonQty, now, createdBy, a.AddonID)
				} else if mode == 1 {
					res, err = database.TxExec(tx, addonWithoutCreatedBy, id, orderID, orgID, addonQty, now, a.AddonID)
				} else {
					res, err = database.TxExec(tx, addonLegacy, id, orderID, now, a.AddonID)
				}
				if err == nil {
					_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_addon")
					break
				}
				errMsg := strings.ToLower(err.Error())
				if mode == 0 && (strings.Contains(errMsg, "unknown column") || strings.Contains(errMsg, "does not exist")) {
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_addon")
					mode = 1
					continue
				}
				if mode == 1 && (strings.Contains(errMsg, "unknown column") || strings.Contains(errMsg, "does not exist")) {
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_addon")
					mode = 2
					continue
				}
				return fmt.Errorf("insert addons: %w", err)
			}
			rows, _ := res.RowsAffected()
			if rows == 0 {
				return fmt.Errorf("addon not found: %s", a.AddonID)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (r *FleetRepository) GetFleetOrderSummary(fleetID, priceID string) (*model.OrderFleetSummaryResponse, error) {
	query := fmt.Sprintf(`
		SELECT f.fleet_name, f.capacity, f.engine, f.body, f.description, f.active, f.thumbnail,
		       fp.duration, fp.rent_type, fp.price, fp.uom
		FROM fleets f
		JOIN fleet_prices fp ON fp.fleet_id = f.uuid
		WHERE f.uuid = %s AND fp.uuid = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	res := &model.OrderFleetSummaryResponse{}

	err := database.QueryRow(r.db, query, fleetID, priceID).Scan(
		&res.FleetName, &res.Capacity, &res.Engine, &res.Body, &res.Description, &res.Active, &res.Thumbnail,
		&res.Duration, &res.RentType, &res.Price, &res.Uom,
	)
	if err != nil {
		return nil, err
	}

	// Facilities
	fQuery := fmt.Sprintf("SELECT facility FROM fleet_facilities WHERE fleet_id = %s", r.getPlaceholder(1))
	rows, err := database.Query(r.db, fQuery, fleetID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var f string
			if err := rows.Scan(&f); err == nil {
				res.Facilities = append(res.Facilities, f)
			}
		}
	}

	// Pickup Points
	pQuery := fmt.Sprintf("SELECT city_id FROM fleet_pickup WHERE fleet_id = %s", r.getPlaceholder(1))
	pRows, err := database.Query(r.db, pQuery, fleetID)
	if err == nil {
		defer pRows.Close()
		for pRows.Next() {
			var cityID int
			if err := pRows.Scan(&cityID); err == nil {
				res.PickupPoints = append(res.PickupPoints, model.PickupPoint{CityID: cityID})
			}
		}
	}

	return res, nil
}

func (r *FleetRepository) GetFleetPrices(orgID, fleetID string) ([]model.FleetPriceItem, error) {
	query := fmt.Sprintf(`
		SELECT uuid, duration, rent_type, price, disc_amount, disc_price, uom
		FROM fleet_prices
		WHERE fleet_id = %s ORDER BY price
	`, r.getPlaceholder(1))

	rows, err := database.Query(r.db, query, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.FleetPriceItem
	for rows.Next() {
		var it model.FleetPriceItem
		var discAmount, discPrice sql.NullFloat64
		if err := rows.Scan(&it.UUID, &it.Duration, &it.RentType, &it.Price, &discAmount, &discPrice, &it.Uom); err == nil {
			if discAmount.Valid {
				it.DiscAmount = discAmount.Float64
			}
			if discPrice.Valid {
				it.DiscPrice = discPrice.Float64
			}
			items = append(items, it)
		}
	}
	return items, nil
}

func (r *FleetRepository) GetFleetPriceListByRentType(fleetID string, rentType int) ([]model.FleetPriceListItem, error) {
	query := fmt.Sprintf(`
		SELECT uuid, fleet_id, duration, rent_type, price
		FROM fleet_prices
		WHERE fleet_id = %s AND rent_type = %s
		ORDER BY price
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	rows, err := database.Query(r.db, query, fleetID, rentType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.FleetPriceListItem, 0)
	for rows.Next() {
		var it model.FleetPriceListItem
		if err := rows.Scan(&it.PriceID, &it.FleetID, &it.Duration, &it.RentType, &it.Price); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

// GetCities returns city IDs
// Map cities in Service layer
func (r *FleetRepository) GetFleetPickup(orgID, fleetID string) ([]model.FleetPickupItem, error) {
	query := `
        SELECT uuid, city_id, '' as city_name 
        FROM fleet_pickup
        WHERE fleet_id = %s
    `
	args := []interface{}{fleetID}
	if orgID != "" {
		query += " AND organization_id = %s"
		query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
		args = append(args, orgID)
	} else {
		query = fmt.Sprintf(query, r.getPlaceholder(1))
	}

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.FleetPickupItem
	for rows.Next() {
		var it model.FleetPickupItem
		if err := rows.Scan(&it.UUID, &it.CityID, &it.CityName); err == nil {
			items = append(items, it)
		}
	}
	return items, nil
}

func (r *FleetRepository) GetOrderList(req *model.GetOrderListRequest) ([]model.OrderListItem, int, error) {
	offset := (req.Page - 1) * req.Limit
	query := fmt.Sprintf(`
		SELECT fo.order_id, fo.created_at, fo.total_amount, fo.status, fo.payment_status,
		       COALESCE(foc.customer_name, '') as customer_name,
		       COALESCE(foc.customer_email, '') as customer_email,
		       COALESCE(foc.customer_phone, '') as customer_phone
		FROM fleet_orders fo
		LEFT JOIN fleet_order_customers foc ON fo.order_id = foc.order_id
		WHERE fo.organization_id = %s
	`, r.getPlaceholder(1))

	args := []interface{}{req.OrganizationID}

	if req.Status != 0 {
		query += fmt.Sprintf(" AND fo.status = %s", r.getPlaceholder(len(args)+1))
		args = append(args, req.Status)
	}

	query += fmt.Sprintf(" ORDER BY fo.created_at DESC LIMIT %s OFFSET %s", r.getPlaceholder(len(args)+1), r.getPlaceholder(len(args)+2))
	args = append(args, req.Limit, offset)

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []model.OrderListItem
	for rows.Next() {
		var it model.OrderListItem
		var createdAt time.Time
		if err := rows.Scan(&it.OrderID, &createdAt, &it.TotalAmount, &it.Status, &it.PaymentStatus, &it.CustomerName, &it.CustomerEmail, &it.CustomerPhone); err == nil {
			it.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
			items = append(items, it)
		}
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM fleet_orders WHERE organization_id = %s", r.getPlaceholder(1))
	countArgs := []interface{}{req.OrganizationID}
	if req.Status != 0 {
		countQuery += fmt.Sprintf(" AND status = %s", r.getPlaceholder(len(countArgs)+1))
		countArgs = append(countArgs, req.Status)
	}
	var total int
	database.QueryRow(r.db, countQuery, countArgs...).Scan(&total)

	return items, total, nil
}

func (r *FleetRepository) GetOrderDetail(orderID, priceID, organizationID string) (*model.OrderDetailResponse, error) {
	// Reusing FindOrderDetail logic
	return r.FindOrderDetail(orderID, organizationID)
}

func (r *FleetRepository) GetFleetOrderTotalAmount(orderID, priceID, organizationID string) (float64, error) {
	query := fmt.Sprintf("SELECT total_amount FROM fleet_orders WHERE order_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
	var amount float64
	err := database.QueryRow(r.db, query, orderID, organizationID).Scan(&amount)
	return amount, err
}

func (r *FleetRepository) GetFleetOrderPaymentsByOrderID(orderID, organizationID string) ([]model.FleetOrderPayment, error) {
	query := fmt.Sprintf(`
		SELECT order_payment_id, payment_type, payment_percentage, payment_amount, total_amount, payment_remaining, status, created_at
		FROM fleet_order_payment
		WHERE order_id = %s AND organization_id = %s
		ORDER BY created_at ASC
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	rows, err := database.Query(r.db, query, orderID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.FleetOrderPayment
	for rows.Next() {
		var p model.FleetOrderPayment
		if err := rows.Scan(&p.OrderPaymentID, &p.PaymentType, &p.PaymentPercentage, &p.PaymentAmount, &p.TotalAmount, &p.PaymentRemaining, &p.Status, &p.CreatedAt); err == nil {
			p.OrderID = orderID
			p.OrganizationID = organizationID
			items = append(items, p)
		}
	}
	return items, nil
}

func (r *FleetRepository) CreateOrderPayment(payment *model.FleetOrderPayment, history *model.OrderPaymentHistory) error {
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

	// Insert Payment
	payQuery := fmt.Sprintf(`
		INSERT INTO fleet_order_payment (order_payment_id, order_id, organization_id, payment_method, payment_type, payment_percentage, payment_amount, total_amount, payment_remaining, status, created_at, bank_code, account_number, account_name, unique_code)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14), r.getPlaceholder(15))

	_, err = tx.Exec(payQuery, payment.OrderPaymentID, payment.OrderID, payment.OrganizationID, payment.PaymentMethod, payment.PaymentType, payment.PaymentPercentage, payment.PaymentAmount, payment.TotalAmount, payment.PaymentRemaining, payment.Status, now, payment.BankCode, payment.AccountNumber, payment.AccountName, payment.UniqueCode)
	if err != nil {
		return err
	}

	// Insert History
	histQuery := fmt.Sprintf(`
		INSERT INTO fleet_order_payment_history (payment_history_id, order_id, bank_code, bank_account_id, account_number, account_name, created_at, organization_id, payment_amount, unique_code)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10))

	_, err = tx.Exec(histQuery, history.PaymentHistoryID, history.OrderID, history.BankCode, history.BankAccountID, history.AccountNumber, history.AccountName, now, history.OrganizationID, history.PaymentAmount, history.UniqueCode)
	if err != nil {
		return err
	}

	// Update Order Payment Status
	return tx.Commit()
}

func (r *FleetRepository) UpdateFleetOrderPaymentStatus(orderID, organizationID string, oldStatus, newStatus int) error {
	query := fmt.Sprintf(`
		UPDATE fleet_order_payment
		SET status = %s
		WHERE order_id = %s AND organization_id = %s AND status = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))

	_, err := database.Exec(r.db, query, newStatus, orderID, organizationID, oldStatus)
	return err
}

func (r *FleetRepository) FindOrderDetail(orderID, organizationID string) (*model.OrderDetailResponse, error) {
	query := fmt.Sprintf(`
        SELECT 
            fo.order_id, fo.created_at, fo.price_id,
            f.fleet_name, 
            fp.rent_type, fp.duration, COALESCE(fp.uom, '') as duration_uom, fp.price, 
            fo.unit_qty, fo.total_amount, COALESCE(fo.additional_amount, 0) as additional_amount,
            fo.pickup_location, fo.pickup_city_id, fo.start_date, fo.end_date,
            COALESCE(foc.customer_name, '') as customer_name, COALESCE(foc.customer_phone, '') as customer_phone, COALESCE(foc.customer_email, '') as customer_email, COALESCE(foc.customer_address, '') as customer_address,
            COALESCE(fo.additional_request, '') as additional_request
        FROM fleet_orders fo
        JOIN fleets f ON fo.fleet_id = f.uuid
        JOIN fleet_prices fp ON fo.price_id = fp.uuid
        LEFT JOIN fleet_order_customers foc ON fo.order_id = foc.order_id
        WHERE fo.order_id = %s AND fo.organization_id = %s
    `, r.getPlaceholder(1), r.getPlaceholder(2))

	var res model.OrderDetailResponse
	var createdAt time.Time
	var pickupCityID string
	var startDate, endDate time.Time

	err := database.QueryRow(r.db, query, orderID, organizationID).Scan(
		&res.OrderID, &createdAt, &res.PriceID,
		&res.FleetName,
		&res.RentType, &res.Duration, &res.DurationUom, &res.Price,
		&res.Quantity, &res.TotalAmount, &res.AdditionalAmount,
		&res.Pickup.PickupLocation, &pickupCityID, &startDate, &endDate,
		&res.Customer.CustomerName, &res.Customer.CustomerPhone, &res.Customer.CustomerEmail, &res.Customer.CustomerAddress,
		&res.AdditionalRequest,
	)
	if err != nil {
		fmt.Println("Error querying order detail:", err)
		return nil, err
	}
	res.OrderDate = createdAt.Format("2006-01-02 15:04:05")
	res.Pickup.PickupCity = pickupCityID
	res.Pickup.StartDate = startDate.Format("2006-01-02")
	res.Pickup.EndDate = endDate.Format("2006-01-02")

	// Destinations
	destQuery := fmt.Sprintf(`SELECT city_id, location FROM fleet_order_destinations WHERE order_id = %s`, r.getPlaceholder(1))
	dRows, err := database.Query(r.db, destQuery, orderID)
	if err == nil {
		defer dRows.Close()
		for dRows.Next() {
			var d model.OrderDetailDest
			var cID string
			if err := dRows.Scan(&cID, &d.Location); err == nil {
				d.City = cID
				res.Destination = append(res.Destination, d)
			}
		}
	}

	// Addons
	addonQuery := fmt.Sprintf(`
        SELECT fa.addon_name, fa.addon_price
        FROM fleet_order_addons foa 
        JOIN fleet_addon fa ON foa.addon_id = fa.uuid 
        WHERE foa.order_id = %s
    `, r.getPlaceholder(1))
	aRows, err := database.Query(r.db, addonQuery, orderID)
	if err != nil {
		return nil, err
	}
	defer aRows.Close()
	for aRows.Next() {
		var a model.OrderDetailAddon
		if err := aRows.Scan(&a.AddonName, &a.AddonPrice); err == nil {
			res.Addon = append(res.Addon, a)
		}
	}

	// Payments
	paymentQuery := fmt.Sprintf(`
		SELECT 
			ba.bank_code, ba.account_name, ba.account_number, bl.name as bank_name, 
			op.payment_type, op.payment_percentage, op.payment_amount, op.total_amount, 
			op.payment_remaining, op.status, op.created_at, op.order_payment_id
		FROM fleet_order_payment op
		LEFT JOIN organization_bank_accounts ba ON op.bank_account_id = ba.bank_account_id
		LEFT JOIN bank_lists bl ON ba.bank_code = bl.code
		WHERE op.order_id = %s
		ORDER BY op.created_at DESC
	`, r.getPlaceholder(1))

	pRows, err := database.Query(r.db, paymentQuery, orderID)
	if err == nil {
		defer pRows.Close()
		var allStatus1 bool = true
		var hasPayment bool = false
		for pRows.Next() {
			hasPayment = true
			var pd model.PaymentDetail
			var bankCode, accName, accNum, bankName sql.NullString
			var createdAt time.Time
			var orderPaymentID string

			if err := pRows.Scan(
				&bankCode, &accName, &accNum, &bankName,
				&pd.PaymentType, &pd.PaymentPercentage, &pd.PaymentAmount, &pd.TotalAmount,
				&pd.PaymentRemaining, &pd.Status, &createdAt, &orderPaymentID,
			); err == nil {
				pd.BankCode = bankCode.String
				pd.AccountName = accName.String
				pd.AccountNumber = accNum.String
				pd.BankName = bankName.String
				pd.PaymentDate = createdAt.Format("2006-01-02 15:04:05")

				res.Payment = append(res.Payment, pd)

				if pd.Status != 1 {
					allStatus1 = false
				}
			}
		}

		// Determine overall payment status
		if !hasPayment {
			res.PaymentStatus = "Belum Bayar"
		} else if allStatus1 {
			res.PaymentStatus = "Lunas"
		}
	}

	return &res, nil
}

func (r *FleetRepository) UpdatePaymentEvidence(orderID, organizationID, filePath string) error {
	// Find latest payment ID
	subQuery := fmt.Sprintf(`
		SELECT order_payment_id 
		FROM fleet_order_payment 
		WHERE order_id = %s AND organization_id = %s 
		ORDER BY created_at DESC 
		LIMIT 1
	`, r.getPlaceholder(2), r.getPlaceholder(3))

	query := fmt.Sprintf(`
		UPDATE fleet_order_payment
		SET evidence_file = %s
		WHERE order_payment_id = (%s)
	`, r.getPlaceholder(1), subQuery)

	_, err := r.db.Exec(query, filePath, orderID, organizationID)
	return err
}

func (r *FleetRepository) GetOrderTotalAmountByType(orderType int, orderID, organizationID string) (float64, error) {
	var query string
	switch orderType {
	case 1: // fleet
		orgExpr := "organization_id = " + r.getPlaceholder(2)
		if r.driver == "postgres" || r.driver == "pgx" {
			orgExpr = "organization_id::text = " + r.getPlaceholder(2)
		}
		query = fmt.Sprintf("SELECT total_amount FROM fleet_orders WHERE order_id = %s AND %s", r.getPlaceholder(1), orgExpr)
	case 2: // package
		orgExpr := "organization_id = " + r.getPlaceholder(2)
		if r.driver == "postgres" || r.driver == "pgx" {
			orgExpr = "organization_id::text = " + r.getPlaceholder(2)
		}
		query = fmt.Sprintf("SELECT total_amount FROM package_orders WHERE order_id = %s AND %s", r.getPlaceholder(1), orgExpr)
	default:
		return 0, fmt.Errorf("invalid order_type")
	}

	var total float64
	err := database.QueryRow(r.db, query, orderID, organizationID).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (r *FleetRepository) GetServiceOrderPaymentStats(orderID, organizationID string) (*model.ServiceOrderPaymentStats, error) {
	orgExpr := "organization_id = " + r.getPlaceholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(2)
	}
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(payment_amount), 0) AS total_paid,
		       COALESCE(SUM(CASE WHEN payment_type = 1001 THEN 1 ELSE 0 END), 0) AS dp_count
		FROM payment_orders
		WHERE order_id = %s AND %s AND COALESCE(status, 0) > 0
	`, r.getPlaceholder(1), orgExpr)

	var totalPaid float64
	var dpCount int
	err := database.QueryRow(r.db, query, orderID, organizationID).Scan(&totalPaid, &dpCount)
	if err != nil {
		return nil, err
	}
	return &model.ServiceOrderPaymentStats{
		TotalPaid:      totalPaid,
		DownPaymentCnt: dpCount,
	}, nil
}

func (r *FleetRepository) InsertServiceOrderPayment(req *model.CreateServiceOrderPaymentRequest, totalAmount, remainingAmount float64) (string, error) {
	paymentID := uuid.New().String()
	now := time.Now()

	query := fmt.Sprintf(`
		INSERT INTO payment_orders
			(payment_id, order_type, order_id, organization_id, payment_type, payment_method, bank_id, bank_account, payment_amount, total_amount, remaining_amount, evidence_file, status, created_at, created_by)
		VALUES
			(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10),
		r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14))

	_, err := database.Exec(
		r.db,
		query,
		paymentID,
		req.OrderType,
		req.OrderID,
		req.OrganizationID,
		req.PaymentType,
		req.PaymentMethod,
		req.BankID,
		req.BankAccount,
		req.PaymentAmount,
		totalAmount,
		remainingAmount,
		req.EvidenceFile,
		now,
		req.CreatedBy,
	)
	if err != nil {
		return "", err
	}
	return paymentID, nil
}

func (r *FleetRepository) ListPaymentOrders(orderID string, orderType int, organizationID string) ([]model.PaymentOrderRow, error) {
	orgExpr := "organization_id = " + r.getPlaceholder(3)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(3)
	}
	query := fmt.Sprintf(`
		SELECT
			payment_id,
			order_type,
			order_id,
			organization_id,
			payment_type,
			payment_method,
			bank_id,
			bank_account,
			payment_amount,
			total_amount,
			remaining_amount,
			evidence_file,
			COALESCE(status, 0),
			created_at,
			created_by
		FROM payment_orders
		WHERE order_id = %s AND order_type = %s AND %s AND COALESCE(status, 0) > 0
		ORDER BY created_at DESC
	`, r.getPlaceholder(1), r.getPlaceholder(2), orgExpr)

	rows, err := database.Query(r.db, query, orderID, orderType, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.PaymentOrderRow, 0)
	for rows.Next() {
		var it model.PaymentOrderRow
		if err := rows.Scan(
			&it.PaymentID,
			&it.OrderType,
			&it.OrderID,
			&it.OrganizationID,
			&it.PaymentType,
			&it.PaymentMethod,
			&it.BankID,
			&it.BankAccount,
			&it.PaymentAmount,
			&it.TotalAmount,
			&it.RemainingAmount,
			&it.EvidenceFile,
			&it.Status,
			&it.CreatedAt,
			&it.CreatedBy,
		); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FleetRepository) GetLatestPaymentOrder(orderID string, orderType int, organizationID string) (*model.PaymentOrderRow, error) {
	orgExpr := "organization_id = " + r.getPlaceholder(3)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(3)
	}
	query := fmt.Sprintf(`
		SELECT
			payment_id,
			order_type,
			order_id,
			organization_id,
			payment_type,
			payment_method,
			bank_id,
			bank_account,
			payment_amount,
			total_amount,
			remaining_amount,
			evidence_file,
			COALESCE(status, 0),
			created_at,
			created_by
		FROM payment_orders
		WHERE order_id = %s AND order_type = %s AND %s AND COALESCE(status, 0) > 0
		ORDER BY created_at DESC
		LIMIT 1
	`, r.getPlaceholder(1), r.getPlaceholder(2), orgExpr)

	var it model.PaymentOrderRow
	err := database.QueryRow(r.db, query, orderID, orderType, organizationID).Scan(
		&it.PaymentID,
		&it.OrderType,
		&it.OrderID,
		&it.OrganizationID,
		&it.PaymentType,
		&it.PaymentMethod,
		&it.BankID,
		&it.BankAccount,
		&it.PaymentAmount,
		&it.TotalAmount,
		&it.RemainingAmount,
		&it.EvidenceFile,
		&it.Status,
		&it.CreatedAt,
		&it.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *FleetRepository) UpdateFleetOrderPaymentStatusOnOrder(orderID, organizationID string, paymentStatus int) error {
	orgExpr := "organization_id = " + r.getPlaceholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(2)
	}
	query := fmt.Sprintf(`
		UPDATE fleet_orders
		SET payment_status = %s
		WHERE order_id = %s AND %s AND COALESCE(status, 0) > 0
	`, r.getPlaceholder(1), r.getPlaceholder(3), orgExpr)

	res, err := database.Exec(r.db, query, paymentStatus, organizationID, orderID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *FleetRepository) ListServiceOrderFleet(orgID, processType string) ([]model.ServiceOrderListItem, error) {
	orgExpr := "organization_id = " + r.getPlaceholder(1)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(1)
	}

	query := fmt.Sprintf(`
		SELECT order_id, fleet_id, start_date
		FROM fleet_orders
		WHERE %s
		  AND payment_status IN (1, 4)
		  AND COALESCE(status, 0) > 0
	`, orgExpr)
	args := []interface{}{orgID}

	switch strings.ToLower(strings.TrimSpace(processType)) {
	case "ongoing":
		query += " AND CURRENT_DATE BETWEEN start_date AND end_date"
	case "upcoming":
		query += " AND start_date >= CURRENT_DATE"
	case "completed":
		query += " AND start_date < CURRENT_DATE AND end_date < CURRENT_DATE"
	}
	query += " ORDER BY start_date ASC"

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.ServiceOrderListItem, 0)
	for rows.Next() {
		var it model.ServiceOrderListItem
		var start time.Time
		if err := rows.Scan(&it.OrderID, &it.FleetID, &start); err != nil {
			return nil, err
		}
		it.StartDate = start.Format("2006-01-02")
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FleetRepository) GetPartnerOrderList(orgID string, filter *model.PartnerOrderListFilter) ([]model.PartnerOrderListItem, error) {
	base := `
        SELECT 
			fo.order_id, f.fleet_name,
			COALESCE(c.customer_name, '') as customer_name,
			COALESCE(c.customer_phone, '') as customer_phone,
			fo.start_date, fo.end_date, fo.unit_qty, fo.payment_status, 
			p.duration, p.uom, fo.total_amount, p.rent_type,
			COALESCE((
				SELECT po.payment_type
				FROM payment_orders po
				WHERE po.order_id = fo.order_id
				  AND po.order_type = 1
				  AND po.organization_id = f.organization_id
				  AND COALESCE(po.status, 0) > 0
				ORDER BY po.created_at DESC
				LIMIT 1
			), 0) as latest_payment_type
        FROM fleet_orders fo 
        INNER JOIN fleets f ON fo.fleet_id = f.uuid 
        INNER JOIN fleet_prices p ON p.uuid = fo.price_id 
		LEFT JOIN customer_orders co ON co.order_id = fo.order_id
		LEFT JOIN customers c ON c.customer_id = co.customer_id AND c.organization_id = f.organization_id
        WHERE f.organization_id = %[1]s
    `
	args := make([]interface{}, 0, 6)
	args = append(args, orgID)
	cond := ""
	if filter != nil {
		if strings.TrimSpace(filter.StartDateFrom) != "" {
			cond += fmt.Sprintf(" AND fo.start_date >= %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.StartDateFrom)
		}
		if strings.TrimSpace(filter.StartDateTo) != "" {
			cond += fmt.Sprintf(" AND fo.start_date <= %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.StartDateTo)
		}
		if strings.TrimSpace(filter.OrderDateFrom) != "" {
			cond += fmt.Sprintf(" AND fo.created_at >= %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.OrderDateFrom)
		}
		if strings.TrimSpace(filter.OrderDateTo) != "" {
			cond += fmt.Sprintf(" AND fo.created_at <= %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.OrderDateTo)
		}
		if filter.HasPaymentStatus {
			cond += fmt.Sprintf(" AND fo.payment_status = %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.PaymentStatus)
		}
	}
	query := fmt.Sprintf(base, r.getPlaceholder(1)) + cond + " ORDER BY fo.created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.PartnerOrderListItem, 0)
	for rows.Next() {
		var it model.PartnerOrderListItem
		var startDate, endDate time.Time
		var rentType int
		var latestPaymentType int
		if err := rows.Scan(
			&it.OrderID, &it.FleetName, &it.CustomerName, &it.CustomerPhone,
			&startDate, &endDate, &it.UnitQty, &it.PaymentStatus,
			&it.Duration, &it.Uom, &it.TotalAmount, &rentType,
			&latestPaymentType,
		); err != nil {
			return nil, err
		}
		it.StartDate = startDate.Format("2006-01-02")
		it.EndDate = endDate.Format("2006-01-02")
		it.LatestPaymentType = latestPaymentType

		switch rentType {
		case 1:
			it.RentType = "Cititour"
		case 2:
			it.RentType = "Overland"
		case 3:
			it.RentType = "Pickup / Drop"
		default:
			it.RentType = "Unknown"
		}

		items = append(items, it)
	}
	return items, nil
}

func (r *FleetRepository) GetPartnerOrderSummary(orgID string, filter *model.PartnerOrderListFilter) (*model.PartnerOrderSummary, error) {
	base := `
        SELECT 
            COUNT(*) AS total_orders,
            SUM(CASE WHEN fo.payment_status = 1 THEN 1 ELSE 0 END) AS paid,
            SUM(CASE WHEN fo.payment_status = 2 THEN 1 ELSE 0 END) AS unpaid,
            SUM(CASE WHEN fo.payment_status IN (3,4) THEN 1 ELSE 0 END) AS pending,
            SUM(CASE WHEN fo.payment_status = 1 THEN fo.total_amount ELSE 0 END) AS revenue,
            SUM(CASE WHEN fo.start_date <= CURRENT_DATE AND fo.end_date >= CURRENT_DATE THEN 1 ELSE 0 END) AS ongoing
        FROM fleet_orders fo
        INNER JOIN fleets f ON fo.fleet_id = f.uuid
        WHERE f.organization_id = %[1]s
    `
	args := make([]interface{}, 0, 6)
	args = append(args, orgID)
	cond := ""
	if filter != nil {
		if strings.TrimSpace(filter.StartDateFrom) != "" {
			cond += fmt.Sprintf(" AND fo.start_date >= %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.StartDateFrom)
		}
		if strings.TrimSpace(filter.StartDateTo) != "" {
			cond += fmt.Sprintf(" AND fo.start_date <= %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.StartDateTo)
		}
		if strings.TrimSpace(filter.OrderDateFrom) != "" {
			cond += fmt.Sprintf(" AND fo.created_at >= %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.OrderDateFrom)
		}
		if strings.TrimSpace(filter.OrderDateTo) != "" {
			cond += fmt.Sprintf(" AND fo.created_at <= %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.OrderDateTo)
		}
		if filter.HasPaymentStatus {
			cond += fmt.Sprintf(" AND fo.payment_status = %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.PaymentStatus)
		}
	}
	query := fmt.Sprintf(base, r.getPlaceholder(1)) + cond
	row := r.db.QueryRow(query, args...)
	var s model.PartnerOrderSummary
	if err := row.Scan(&s.TotalOrders, &s.Paid, &s.Unpaid, &s.Pending, &s.Revenue, &s.Ongoing); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *FleetRepository) GetPartnerOrderDetail(orderID, orgID string) (*model.OrderDetailResponse, error) {
	customerCityExpr := "COALESCE(c.customer_city, '')"
	if r.driver == "postgres" || r.driver == "pgx" {
		customerCityExpr = "COALESCE(c.customer_city::text, '')"
	} else if r.driver == "mysql" {
		customerCityExpr = "COALESCE(CAST(c.customer_city AS CHAR), '')"
	}

	query := fmt.Sprintf(`
        SELECT 
            fo.order_id, fo.fleet_id, fo.created_at, fo.price_id,
            f.fleet_name, 
            fp.rent_type, fp.duration, COALESCE(fp.uom, '') as duration_uom, fp.price, 
            fo.unit_qty, fo.total_amount, COALESCE(fo.additional_amount, 0) as additional_amount,
            fo.pickup_location, fo.pickup_city_id, fo.start_date, fo.end_date,
            COALESCE(c.customer_name, '') as customer_name,
			COALESCE(c.customer_phone, '') as customer_phone,
			COALESCE(c.customer_email, '') as customer_email,
			COALESCE(c.customer_address, '') as customer_address,
			%[1]s as customer_city,
			COALESCE(fo.additional_request, '') as additional_request
        FROM fleet_orders fo
        JOIN fleets f ON fo.fleet_id = f.uuid
        JOIN fleet_prices fp ON fo.price_id = fp.uuid
		LEFT JOIN customer_orders co ON co.order_id = fo.order_id AND co.organization_id = f.organization_id
		LEFT JOIN customers c ON c.customer_id = co.customer_id AND c.organization_id = f.organization_id
        WHERE fo.order_id = %s AND f.organization_id = %s
    `, customerCityExpr, r.getPlaceholder(1), r.getPlaceholder(2))

	var res model.OrderDetailResponse
	var createdAt time.Time
	var pickupCityID string
	var startDate, endDate time.Time

	err := r.db.QueryRow(query, orderID, orgID).Scan(
		&res.OrderID, &res.FleetID, &createdAt, &res.PriceID,
		&res.FleetName,
		&res.RentType, &res.Duration, &res.DurationUom, &res.Price,
		&res.Quantity, &res.TotalAmount, &res.AdditionalAmount,
		&res.Pickup.PickupLocation, &pickupCityID, &startDate, &endDate,
		&res.Customer.CustomerName, &res.Customer.CustomerPhone, &res.Customer.CustomerEmail, &res.Customer.CustomerAddress, &res.Customer.CustomerCity,
		&res.AdditionalRequest,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found or access denied")
		}
		fmt.Println("Error querying order detail:", err)
		return nil, err
	}
	res.OrderDate = createdAt.Format("2006-01-02 15:04:05")
	res.Pickup.PickupCity = pickupCityID
	res.Pickup.StartDate = startDate.Format("2006-01-02 15:04")
	res.Pickup.EndDate = endDate.Format("2006-01-02 15:04")

	// Destinations
	destQuery := fmt.Sprintf(`SELECT city_id, location FROM fleet_order_destinations WHERE order_id = %s`, r.getPlaceholder(1))
	dRows, err := r.db.Query(destQuery, orderID)
	if err == nil {
		defer dRows.Close()
		for dRows.Next() {
			var d model.OrderDetailDest
			var cID string
			if err := dRows.Scan(&cID, &d.Location); err == nil {
				d.City = cID
				res.Destination = append(res.Destination, d)
			}
		}
	}

	if len(res.Destination) > 0 {
		res.Itinerary = make([]model.FleetOrderItineraryItem, 0, len(res.Destination))
		for i := range res.Destination {
			res.Itinerary = append(res.Itinerary, model.FleetOrderItineraryItem{
				Day:         i + 1,
				CityID:      res.Destination[i].City,
				Destination: res.Destination[i].Location,
			})
		}
	}

	cityExpr := "city_id"
	if r.driver == "postgres" || r.driver == "pgx" {
		cityExpr = "city_id::text"
	} else if r.driver == "mysql" {
		cityExpr = "CAST(city_id AS CHAR)"
	}
	itQuery := fmt.Sprintf(`SELECT day_num, %s as city_id, location FROM fleet_order_itinerary WHERE order_id = %s AND organization_id = %s ORDER BY day_num`, cityExpr, r.getPlaceholder(1), r.getPlaceholder(2))
	iRows, itErr := r.db.Query(itQuery, orderID, orgID)
	if itErr == nil {
		defer iRows.Close()
		items := make([]model.FleetOrderItineraryItem, 0)
		for iRows.Next() {
			var it model.FleetOrderItineraryItem
			if err := iRows.Scan(&it.Day, &it.CityID, &it.Destination); err == nil {
				items = append(items, it)
			}
		}
		if len(items) > 0 {
			res.Itinerary = items
		}
	}

	// Addons
	addonQuery := fmt.Sprintf(`
        SELECT fa.addon_name, fa.addon_price
        FROM fleet_order_addons foa 
        JOIN fleet_addon fa ON foa.addon_id = fa.uuid 
        WHERE foa.order_id = %s
    `, r.getPlaceholder(1))
	aRows, err := r.db.Query(addonQuery, orderID)
	if err == nil {
		defer aRows.Close()
		for aRows.Next() {
			var a model.OrderDetailAddon
			if err := aRows.Scan(&a.AddonName, &a.AddonPrice); err == nil {
				res.Addon = append(res.Addon, a)
			}
		}
	}
	return &res, nil
}

func (r *FleetRepository) GetFleetAddon(orgID, fleetID string) ([]model.FleetAddonItem, error) {
	query := `SELECT uuid, addon_name, addon_desc, addon_price FROM fleet_addon WHERE fleet_id = %s`
	args := []interface{}{fleetID}
	if orgID != "" {
		query += " AND organization_id = %s"
		query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
		args = append(args, orgID)
	} else {
		query = fmt.Sprintf(query, r.getPlaceholder(1))
	}

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.FleetAddonItem
	for rows.Next() {
		var it model.FleetAddonItem
		if err := rows.Scan(&it.UUID, &it.AddonName, &it.AddonDesc, &it.AddonPrice); err == nil {
			items = append(items, it)
		}
	}
	return items, nil
}

func (r *FleetRepository) GetFleetImages(fleetID string) ([]model.FleetImageItem, error) {
	query := fmt.Sprintf("SELECT uuid, path_file FROM fleet_images WHERE fleet_id = %s", r.getPlaceholder(1))
	rows, err := database.Query(r.db, query, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.FleetImageItem
	for rows.Next() {
		var it model.FleetImageItem
		if err := rows.Scan(&it.UUID, &it.PathFile); err == nil {
			items = append(items, it)
		}
	}
	return items, nil
}

func (r *FleetRepository) GetServiceFleets(page, perPage int) ([]model.ServiceFleetItem, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage

	var groupConcat string
	if r.driver == "postgres" {
		groupConcat = "STRING_AGG(CAST(city_id AS VARCHAR), ',')"
	} else {
		groupConcat = "GROUP_CONCAT(city_id)"
	}

	query := fmt.Sprintf(`
        SELECT f.uuid, f.fleet_name, f.fleet_type, f.capacity, f.production_year, f.engine, f.body, f.description, f.thumbnail, f.created_at,
        (SELECT MIN(price) FROM fleet_prices WHERE fleet_id = f.uuid) as price,
        (SELECT uom FROM fleet_prices WHERE fleet_id = f.uuid ORDER BY price ASC LIMIT 1) as uom,
        (SELECT %s FROM fleet_pickup WHERE fleet_id = f.uuid) as cities
        FROM fleets f
        WHERE f.active = true
        ORDER BY f.created_at DESC
        LIMIT %d OFFSET %d
    `, groupConcat, perPage, offset)

	rows, err := database.Query(r.db, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.ServiceFleetItem
	for rows.Next() {
		var it model.ServiceFleetItem
		var price sql.NullFloat64
		var uom sql.NullString
		var cities sql.NullString

		if err := rows.Scan(
			&it.FleetID, &it.FleetName, &it.FleetType, &it.Capacity, &it.ProductionYear, &it.Engine, &it.Body, &it.Description, &it.Thumbnail, &it.CreatedAt,
			&price, &uom, &cities,
		); err != nil {
			return nil, err
		}

		if price.Valid {
			it.OriginalPrice = price.Float64
		}
		if uom.Valid {
			it.Uom = uom.String
		}
		if cities.Valid {
			it.Cities = strings.Split(cities.String, ",")
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *FleetRepository) GetAvailableCities(orgID string) ([]int, error) {
	query := fmt.Sprintf(`
		SELECT city_id
		FROM fleet_pickup
		WHERE organization_id = %s
		GROUP BY city_id
	`, r.getPlaceholder(1))

	rows, err := database.Query(r.db, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cities []int
	for rows.Next() {
		var cityID int
		if err := rows.Scan(&cityID); err == nil {
			cities = append(cities, cityID)
		}
	}
	return cities, nil
}

func (r *FleetRepository) GetFleetOrgID(fleetID string) (string, error) {
	query := fmt.Sprintf("SELECT organization_id FROM fleets WHERE uuid = %s", r.getPlaceholder(1))
	var orgID string
	err := database.QueryRow(r.db, query, fleetID).Scan(&orgID)
	return orgID, err
}

func (r *FleetRepository) GetFleetDetailMeta(orgID, fleetID string) (*model.FleetDetailMeta, error) {
	fleetTypeExpr := "CAST(f.fleet_type AS CHAR)"
	createdByExpr := "COALESCE(u.fullname, CAST(f.created_by AS CHAR))"
	if r.driver == "postgres" || r.driver == "pgx" {
		fleetTypeExpr = "f.fleet_type::text"
		createdByExpr = "COALESCE(u.fullname, f.created_by::text)"
	}

	query := fmt.Sprintf(`
        SELECT 
			f.uuid,
			%s AS fleet_type,
			COALESCE(ft.label, '') AS fleet_type_label,
			f.fleet_name,
			f.capacity,
			f.production_year,
			f.engine,
			f.body,
			COALESCE(f.fuel_type, '') AS fuel_type,
			COALESCE(f.transmission, '') AS transmission,
			f.description,
			f.thumbnail,
			f.active,
			f.status,
			f.created_at,
			%s AS created_by,
			f.updated_at,
			f.updated_by
        FROM fleets f
		LEFT JOIN fleet_types ft ON f.fleet_type = ft.id
		LEFT JOIN users u ON u.user_id = f.created_by
        WHERE f.uuid = %s
    `, fleetTypeExpr, createdByExpr, r.getPlaceholder(1))

	args := []interface{}{fleetID}
	if orgID != "" {
		query += " AND f.organization_id = %s"
		query = fmt.Sprintf(query, r.getPlaceholder(2))
		args = append(args, orgID)
	}

	var meta model.FleetDetailMeta
	// Note: using explicit fields
	// Handle potential nulls
	var createdAt time.Time
	var updatedAt sql.NullTime
	var updatedBy sql.NullString
	var createdBy sql.NullString
	var fleetType string
	var fleetTypeLabel sql.NullString
	var fuelType sql.NullString
	var transmission sql.NullString
	// FleetDetailMeta: CreatedAt string `json:"created_at"`

	err := database.QueryRow(r.db, query, args...).Scan(
		&meta.FleetID,
		&fleetType,
		&fleetTypeLabel,
		&meta.FleetName,
		&meta.Capacity,
		&meta.ProductionYear,
		&meta.Engine,
		&meta.Body,
		&fuelType,
		&transmission,
		&meta.Description,
		&meta.Thumbnail,
		&meta.Active,
		&meta.Status,
		&createdAt,
		&createdBy,
		&updatedAt,
		&updatedBy,
	)
	if err != nil {
		return nil, err
	}
	meta.FleetType = fleetType
	if fleetTypeLabel.Valid {
		meta.FleetTypeLabel = fleetTypeLabel.String
	}
	if fuelType.Valid {
		meta.FuelType = fuelType.String
	}
	if transmission.Valid {
		meta.Transmission = transmission.String
	}
	meta.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
	if createdBy.Valid {
		meta.CreatedBy = createdBy.String
	}
	if updatedAt.Valid {
		meta.UpdatedAt = updatedAt.Time.Format("2006-01-02 15:04:05")
	}
	if updatedBy.Valid {
		meta.UpdatedBy = updatedBy.String
	}

	return &meta, nil
}

func (r *FleetRepository) GetPriceByID(priceID string) (float64, int, error) {
	query := fmt.Sprintf("SELECT price, rent_type FROM fleet_prices WHERE uuid = %s", r.getPlaceholder(1))
	var price float64
	var rentType int
	err := database.QueryRow(r.db, query, priceID).Scan(&price, &rentType)
	return price, rentType, err
}

func (r *FleetRepository) GetAddonPrices(addonIDs []string) (map[string]float64, error) {
	res := make(map[string]float64)
	if len(addonIDs) == 0 {
		return res, nil
	}
	unique := make(map[string]struct{}, len(addonIDs))
	ids := make([]string, 0, len(addonIDs))
	for _, id := range addonIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := unique[id]; ok {
			continue
		}
		unique[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return res, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = r.getPlaceholder(i + 1)
		args[i] = id
	}
	query := fmt.Sprintf("SELECT uuid, addon_price FROM fleet_addon WHERE uuid IN (%s)", strings.Join(placeholders, ","))
	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var price float64
		if err := rows.Scan(&id, &price); err != nil {
			return nil, err
		}
		res[id] = price
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		if _, ok := res[id]; !ok {
			return nil, fmt.Errorf("addon not found: %s", id)
		}
	}
	return res, nil
}

func (r *FleetRepository) GetAddonPriceSum(addonIDs []string) (float64, error) {
	if len(addonIDs) == 0 {
		return 0, nil
	}
	// Create placeholders for IN clause
	placeholders := make([]string, len(addonIDs))
	args := make([]interface{}, len(addonIDs))
	for i, id := range addonIDs {
		placeholders[i] = r.getPlaceholder(i + 1)
		args[i] = id
	}
	query := fmt.Sprintf("SELECT COALESCE(SUM(addon_price), 0) FROM fleet_addon WHERE uuid IN (%s)", strings.Join(placeholders, ","))
	var total float64
	err := database.QueryRow(r.db, query, args...).Scan(&total)
	return total, err
}

func (r *FleetRepository) GetOrderCountByOrgID(orgID string) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM fleet_orders WHERE organization_id = %s", r.getPlaceholder(1))
	var count int
	err := database.QueryRow(r.db, query, orgID).Scan(&count)
	return count, err
}

func (r *FleetRepository) GetOrganizationCodeByOrgID(orgID string) (string, error) {
	query := fmt.Sprintf("SELECT organization_code FROM organizations WHERE organization_id = %s", r.getPlaceholder(1))
	var code string
	err := database.QueryRow(r.db, query, orgID).Scan(&code)
	return code, err
}

func (r *FleetRepository) getPlaceholder(pos int) string {
	if r.driver == "postgres" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}

func (r *FleetRepository) ListFleetsForUnit(orgID, searchFor string) ([]model.FleetUnitSearchItem, error) {
	var query string
	args := make([]interface{}, 0, 2)
	args = append(args, orgID)

	if r.driver == "postgres" || r.driver == "pgx" {
		query = listFleetsForUnitPostgres
		if strings.TrimSpace(searchFor) != "" {
			query = listFleetsForUnitPostgresSearch
			args = append(args, searchFor)
		}
	} else {
		query = listFleetsForUnitMySQL
		if strings.TrimSpace(searchFor) != "" {
			query = listFleetsForUnitMySQLSearch
			args = append(args, searchFor)
		}
	}

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.FleetUnitSearchItem, 0)
	for rows.Next() {
		var it model.FleetUnitSearchItem
		if err := rows.Scan(&it.FleetID, &it.FleetName); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) SoftDeleteFleet(orgID, userID, fleetID string) error {
	query := softDeleteFleetMySQL
	if r.driver == "postgres" || r.driver == "pgx" {
		query = softDeleteFleetPostgres
	}
	res, err := database.Exec(r.db, query, time.Now(), userID, fleetID, orgID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
