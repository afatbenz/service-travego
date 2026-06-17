package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"service-travego/configs"
	"service-travego/database"
	"service-travego/model"
	"service-travego/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	fuelTypeLabelOnce sync.Once
	fuelTypeLabelMap  map[string]string

	citiesOnce sync.Once
	citiesMap  map[string]string
)

func getCitiesMap() map[string]string {
	citiesOnce.Do(func() {
		citiesMap = map[string]string{}
		f, err := os.Open("config/location.json")
		if err != nil {
			return
		}
		defer f.Close()
		var loc model.Location
		if err := json.NewDecoder(f).Decode(&loc); err != nil {
			return
		}
		for _, c := range loc.Cities {
			citiesMap[strings.TrimSpace(c.ID)] = c.Name
		}
	})
	return citiesMap
}

func getFuelTypeLabelMap() map[string]string {
	fuelTypeLabelOnce.Do(func() {
		fuelTypeLabelMap = map[string]string{}
		f, err := os.Open("config/fleet-config.json")
		if err != nil {
			return
		}
		defer f.Close()
		var cfg model.FleetConfig
		if err := json.NewDecoder(f).Decode(&cfg); err != nil {
			return
		}
		for _, it := range cfg.FuelType {
			if it.ID != "" && it.Label != "" {
				fuelTypeLabelMap[it.ID] = it.Label
			}
		}
	})
	return fuelTypeLabelMap
}

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

const setFleetActivePostgres = `
UPDATE fleets
SET active = $1, updated_at = $2, updated_by = $3
WHERE uuid = $4 AND organization_id::text = $5
`

const setFleetActiveMySQL = `
UPDATE fleets
SET active = ?, updated_at = ?, updated_by = ?
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
	totalUnitExpr := "COALESCE((SELECT COUNT(*) FROM fleet_units fu WHERE fu.fleet_id::text = f.uuid::text AND fu.status = 1), 0)"
	base := `
        SELECT f.uuid AS fleet_id, ft.label AS fleet_type, f.fleet_name, f.capacity, f.engine, f.body, %s as total_unit, f.active, f.status, f.thumbnail, STRING_AGG(DISTINCT fu.engine::text, ', ') AS engines, STRING_AGG(DISTINCT fu.capacity::text, ', ') AS capacities
        FROM fleets f INNER JOIN fleet_types ft ON f.fleet_type = ft.id
		INNER JOIN fleet_units fu ON fu.fleet_id::text = f.uuid::text
    `
	base = fmt.Sprintf(base, totalUnitExpr)
	where := make([]string, 0, 4)
	args := make([]interface{}, 0, 4)
	pos := 1
	where = append(where, "f.status > 0")
	if req.OrganizationID != "" {
		orgExpr := fmt.Sprintf("f.organization_id::text = %s", r.getPlaceholder(pos))
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
		likeExpr := "f.fleet_name ILIKE " + r.getPlaceholder(pos)
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
	query = query + " GROUP BY f.uuid, ft.label, f.fleet_name, f.capacity, f.engine, f.body, f.active, f.status, f.thumbnail, f.created_at ORDER BY f.created_at DESC"
	fmt.Println(query)

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
		var engines sql.NullString
		var capacities sql.NullString
		var body sql.NullString
		var thumbnail sql.NullString
		var totalUnit int64
		if err := rows.Scan(&item.FleetID, &fleetType, &item.FleetName, &item.Capacity, &engine, &body, &totalUnit, &item.Active, &item.Status, &thumbnail, &engines, &capacities); err != nil {
			return nil, err
		}
		if fleetType.Valid {
			item.FleetType = fleetType.String
		}
		if engine.Valid {
			item.Engine = engine.String
		}
		if engines.Valid {
			item.Engines = engines.String
		}
		if capacities.Valid {
			item.Capacities = capacities.String
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
        INSERT INTO fleets (uuid, organization_id, fleet_type, fleet_name, capacity, production_year, engine, body, fuel_type, description, thumbnail, active, is_public, created_at, created_by, status)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9),
		r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14), r.getPlaceholder(15), r.getPlaceholder(16))

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
		req.IsPublic,
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

	// 2. Check and Insert customers
	var custID string
	checkQuery := fmt.Sprintf(`
		SELECT customer_id FROM customers 
		WHERE organization_id = %s AND (customer_email = %s AND customer_phone = %s)
		LIMIT 1
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	err = database.TxQueryRow(tx, checkQuery, req.OrganizationID, req.Email, req.Phone).Scan(&custID)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("error checking existing customer", err)
		return err
	}

	if err == sql.ErrNoRows {
		custID = uuid2()
		customerQuery := fmt.Sprintf(`
			INSERT INTO customers (customer_id, organization_id, customer_name, customer_email, customer_address, customer_city, customer_company, created_at, customer_phone)
			VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9))

		_, err = database.TxExec(tx, customerQuery, custID, req.OrganizationID, req.Fullname, req.Email, req.Address, req.CityID, req.CompanyName, now, req.Phone)
		if err != nil {
			fmt.Println("error insert customers", err)
			return err
		}
	}

	// 3. Insert customer_orders
	custOrderQuery := fmt.Sprintf(`
		INSERT INTO customer_orders (order_id, customer_id, order_type, created_at, organization_id)
		VALUES (%s, %s, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5))

	_, err = database.TxExec(tx, custOrderQuery, orderID, custID, req.OrderType, now, req.OrganizationID)
	if err != nil {
		fmt.Println("error insert customer_orders", err)
		return err
	}

	// 4. Insert fleet_order_items
	// Get price again or pass it from service. Since we are in repo, let's query it.
	var price float64
	priceQuery := fmt.Sprintf("SELECT price FROM fleet_prices WHERE uuid = %s", r.getPlaceholder(1))
	err = database.TxQueryRow(tx, priceQuery, req.PriceID).Scan(&price)
	if err != nil {
		fmt.Println("error get price for order items", err)
		return err
	}

	// Get addon amount
	var addonAmount float64
	if len(req.Addons) > 0 {
		placeholders := make([]string, len(req.Addons))
		args := make([]interface{}, len(req.Addons))
		for i, id := range req.Addons {
			placeholders[i] = r.getPlaceholder(i + 1)
			args[i] = id
		}
		addonSumQuery := fmt.Sprintf("SELECT COALESCE(SUM(addon_price), 0) FROM fleet_addon WHERE uuid IN (%s)", strings.Join(placeholders, ","))
		err = database.TxQueryRow(tx, addonSumQuery, args...).Scan(&addonAmount)
		if err != nil {
			fmt.Println("error get addon sum for order items", err)
			return err
		}
	}

	subTotal := (float64(req.Qty) * price) + (float64(req.Qty) * addonAmount)
	orderItemID := uuid2()
	itemQuery := fmt.Sprintf(`
		INSERT INTO fleet_order_items (order_item_id, organization_id, order_id, fleet_id, price_id, quantity, sub_total, create_at, status, addon_amount)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, 1, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9))

	_, err = database.TxExec(tx, itemQuery, orderItemID, req.OrganizationID, orderID, req.FleetID, req.PriceID, req.Qty, subTotal, now, addonAmount)
	if err != nil {
		fmt.Println("error insert fleet_order_items", err)
		return err
	}

	// 5. Insert fleet_orders_addon (existing logic, keeping it but it might be redundant now)
	if len(req.Addons) > 0 {
		addonQuery := fmt.Sprintf(`
			INSERT INTO fleet_order_addons (order_addon_id, order_id, order_item_id, organization_id, addon_id, addon_price, created_at)
			SELECT %s, %s, %s, %s, uuid, addon_price, %s FROM fleet_addon WHERE uuid = %s
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
		for _, addonID := range req.Addons {
			id := uuid2()
			res, err := database.TxExec(tx, addonQuery, id, orderID, orderItemID, req.OrganizationID, now, addonID)
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

	// 6. Insert fleet_order_itinerary
	if len(req.Destinations) > 0 {
		mode := 0
		itineraryWithOrg := fmt.Sprintf(`
			INSERT INTO fleet_order_itinerary (fleet_itinerary_id, order_id, day_num, city_id, location, organization_id, created_at)
			VALUES (%s, %s, %s, %s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7))
		itineraryWithoutOrg := fmt.Sprintf(`
			INSERT INTO fleet_order_itinerary (fleet_itinerary_id, order_id, day_num, city_id, location, created_at)
			VALUES (%s, %s, %s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
		destQuery := fmt.Sprintf(`
			INSERT INTO fleet_order_destinations (order_id, city_id, location, created_at)
			VALUES (%s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))
		destQueryWithID := fmt.Sprintf(`
			INSERT INTO fleet_order_destinations (uuid, order_id, city_id, location, created_at)
			VALUES (%s, %s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5))

		for i, dest := range req.Destinations {
			id := uuid2()
			dayNum := i + 1
			for {
				_, _ = database.TxExec(tx, "SAVEPOINT sp_dest")
				switch mode {
				case 0:
					_, err = database.TxExec(tx, itineraryWithOrg, id, orderID, dayNum, dest.CityID, dest.Location, req.OrganizationID, now)
				case 1:
					_, err = database.TxExec(tx, itineraryWithoutOrg, id, orderID, dayNum, dest.CityID, dest.Location, now)
				case 2:
					_, err = database.TxExec(tx, destQuery, orderID, dest.CityID, dest.Location, now)
				default:
					_, err = database.TxExec(tx, destQueryWithID, id, orderID, dest.CityID, dest.Location, now)
				}

				if err == nil {
					_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_dest")
					break
				}

				errMsg := strings.ToLower(err.Error())
				if mode == 0 && (strings.Contains(errMsg, "unknown column") || strings.Contains(errMsg, "does not exist")) {
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_dest")
					mode = 1
					continue
				}
				if (mode == 0 || mode == 1) && (strings.Contains(errMsg, "doesn't exist") || strings.Contains(errMsg, "does not exist") || strings.Contains(errMsg, "relation") || strings.Contains(errMsg, "unknown table")) {
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_dest")
					mode = 2
					continue
				}
				if mode == 2 && (strings.Contains(errMsg, "unknown column \"uuid\"") || strings.Contains(errMsg, "column \"uuid\" of relation") || strings.Contains(errMsg, "does not exist")) {
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_dest")
					mode = 3
					continue
				}

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

func (r *FleetRepository) CreatePartnerOrder(orderID, fleetID, startDate, endDate, pickupCityID, pickupLocation string, qty int, priceID string, totalAmount, additionalAmount float64, customerID, orgID, createdBy string, itinerary []model.FleetOrderItineraryItem, addons []model.FleetOrderAddonItem, additionalRequest string, fleets []model.FleetOrderFleetItem) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	_ = addons

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()

	insertWithCreatedBy := fmt.Sprintf(`
		INSERT INTO fleet_orders (order_id, fleet_id, start_date, end_date, pickup_city_id, pickup_location, unit_qty, price_id, created_at, total_amount, additional_amount, status, payment_status, organization_id, created_by, additional_request)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1, %d, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), configs.PaymentStatusWaitingPayment, r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14))

	_, _ = database.TxExec(tx, "SAVEPOINT sp_orders")
	_, err = database.TxExec(tx, insertWithCreatedBy, orderID, fleetID, startDate, endDate, pickupCityID, pickupLocation, qty, priceID, now, totalAmount, additionalAmount, orgID, createdBy, additionalRequest)
	if err != nil {
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "unknown column") || strings.Contains(errMsg, "does not exist") {
			_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_orders")
			insertWithoutCreatedBy := fmt.Sprintf(`
				INSERT INTO fleet_orders (order_id, fleet_id, start_date, end_date, pickup_city_id, pickup_location, unit_qty, price_id, created_at, total_amount, status, payment_status, organization_id, additional_request)
				VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1, %d, %s, %s)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
				r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), configs.PaymentStatusWaitingPayment, r.getPlaceholder(11), r.getPlaceholder(12))

			_, _ = database.TxExec(tx, "SAVEPOINT sp_orders_2")
			_, err = database.TxExec(tx, insertWithoutCreatedBy, orderID, fleetID, startDate, endDate, pickupCityID, pickupLocation, qty, priceID, now, totalAmount, orgID, additionalRequest)
			if err != nil {
				errMsg2 := strings.ToLower(err.Error())
				if strings.Contains(errMsg2, "additional_request") {
					// Fallback if additional_request missing
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_orders_2")
					insertLegacy := fmt.Sprintf(`
						INSERT INTO fleet_orders (order_id, fleet_id, start_date, end_date, pickup_city_id, pickup_location, unit_qty, price_id, created_at, total_amount, status, payment_status, organization_id)
						VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1, %d, %s)
					`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
						r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), configs.PaymentStatusWaitingPayment, r.getPlaceholder(11))
					_, err = database.TxExec(tx, insertLegacy, orderID, fleetID, startDate, endDate, pickupCityID, pickupLocation, qty, priceID, now, totalAmount, orgID)
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
				switch mode {
				case 0:
					_, err = database.TxExec(tx, itineraryWithCreatedBy, id, orderID, it.Day, it.CityID, it.Destination, orgID, now, createdBy)
				case 1:
					_, err = database.TxExec(tx, itineraryWithoutCreatedBy, id, orderID, it.Day, it.CityID, it.Destination, orgID, now)
				default:
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

	// Insert fleet_order_items
	if err := r.CreateFleetOrderItems(tx, orderID, orgID, createdBy, fleets); err != nil {
		return err
	}

	_, _ = r.RecalculateFleetOrderTotal(tx, orderID, orgID)

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (r *FleetRepository) CreateFleetOrderItems(tx *sql.Tx, orderID, orgID, createdBy string, fleets []model.FleetOrderFleetItem) error {
	if len(fleets) == 0 {
		return nil
	}

	now := time.Now()

	priceIDs := make([]string, 0, len(fleets))
	addonIDs := make([]string, 0)
	for _, f := range fleets {
		if strings.TrimSpace(f.PriceID) != "" {
			priceIDs = append(priceIDs, f.PriceID)
		}
		for _, a := range f.Addons {
			a = strings.TrimSpace(a)
			if a != "" {
				addonIDs = append(addonIDs, a)
			}
		}
		if strings.TrimSpace(f.AddonID) != "" {
			addonIDs = append(addonIDs, strings.TrimSpace(f.AddonID))
		}
	}
	priceMap, _ := r.GetFleetPricesByIDs(priceIDs)
	addonPriceMap, err := r.GetAddonPrices(addonIDs)
	if err != nil {
		return err
	}

	// Try with all columns first
	insertWithAll := fmt.Sprintf(`
		INSERT INTO fleet_order_items (order_item_id, organization_id, order_id, fleet_id, price_id, quantity, charge_amount, addon_amount, discount, sub_total, create_at, created_by, status)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12))

	// Fallback without created_by
	insertWithoutCreatedBy := fmt.Sprintf(`
		INSERT INTO fleet_order_items (order_item_id, organization_id, order_id, fleet_id, price_id, quantity, charge_amount, addon_amount, discount, sub_total, create_at, status)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11))

	insertWithAllNoAddon := fmt.Sprintf(`
		INSERT INTO fleet_order_items (order_item_id, organization_id, order_id, fleet_id, price_id, quantity, charge_amount, discount, sub_total, create_at, created_by, status)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11))

	insertWithoutCreatedByNoAddon := fmt.Sprintf(`
		INSERT INTO fleet_order_items (order_item_id, organization_id, order_id, fleet_id, price_id, quantity, charge_amount, discount, sub_total, create_at, status)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10))

	mode := 0
	for _, f := range fleets {
		id := uuid2()
		q := f.Qty
		if q <= 0 {
			q = 1
		}
		unitPrice := 0.0
		if strings.TrimSpace(f.PriceID) != "" {
			if p, ok := priceMap[strings.TrimSpace(f.PriceID)]; ok {
				unitPrice = p
			} else {
				p, _, e := r.GetPriceByID(strings.TrimSpace(f.PriceID))
				if e == nil {
					unitPrice = p
				}
			}
		}

		addonIDsForItem := normalizeAddonIDs(f.Addons, f.AddonID)
		addonAmount := 0.0
		for _, a := range addonIDsForItem {
			addonAmount += addonPriceMap[a]
		}

		subTotal := (unitPrice * float64(q)) + (f.BiayaLain * float64(q)) + (addonAmount * float64(q)) - (f.Discount * float64(q))
		if subTotal < 0 {
			subTotal = 0
		}

		for {
			_, _ = database.TxExec(tx, "SAVEPOINT sp_fleet_items")
			var err error
			switch mode {
			case 0:
				_, err = database.TxExec(tx, insertWithAll, id, orgID, orderID, f.ArmadaID, f.PriceID, q, f.BiayaLain, addonAmount, f.Discount, subTotal, now, createdBy)
			case 1:
				_, err = database.TxExec(tx, insertWithoutCreatedBy, id, orgID, orderID, f.ArmadaID, f.PriceID, q, f.BiayaLain, addonAmount, f.Discount, subTotal, now)
			case 2:
				_, err = database.TxExec(tx, insertWithAllNoAddon, id, orgID, orderID, f.ArmadaID, f.PriceID, q, f.BiayaLain, f.Discount, subTotal, now, createdBy)
			default:
				_, err = database.TxExec(tx, insertWithoutCreatedByNoAddon, id, orgID, orderID, f.ArmadaID, f.PriceID, q, f.BiayaLain, f.Discount, subTotal, now)
			}
			if err == nil {
				_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_fleet_items")
				break
			}
			errMsg := strings.ToLower(err.Error())
			if mode < 3 && (strings.Contains(errMsg, "unknown column") || strings.Contains(errMsg, "does not exist")) {
				_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_fleet_items")
				mode++
				continue
			}
			return fmt.Errorf("insert fleet_order_items: %w", err)
		}

		if err := r.replaceFleetOrderItemAddons(tx, orderID, orgID, createdBy, id, addonIDsForItem, now, false); err != nil {
			return err
		}
	}

	return nil
}

func normalizeAddonIDs(addons []string, addonID string) []string {
	seen := make(map[string]struct{}, len(addons)+1)
	out := make([]string, 0, len(addons)+1)
	for _, a := range addons {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		out = append(out, a)
	}
	addonID = strings.TrimSpace(addonID)
	if addonID != "" {
		if _, ok := seen[addonID]; !ok {
			out = append(out, addonID)
		}
	}
	return out
}

func (r *FleetRepository) replaceFleetOrderItemAddons(tx *sql.Tx, orderID, orgID, createdBy, orderItemID string, addonIDs []string, now time.Time, deleteFirst bool) error {
	if len(addonIDs) == 0 {
		if !deleteFirst {
			return nil
		}
	}

	orgExpr := "organization_id = " + r.getPlaceholder(3)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(3)
	}

	if deleteFirst {
		delCandidates := []struct {
			query string
			args  []interface{}
		}{
			{
				query: fmt.Sprintf(`DELETE FROM fleet_order_addons WHERE order_item_id = %s AND order_id = %s AND %s`, r.getPlaceholder(1), r.getPlaceholder(2), orgExpr),
				args:  []interface{}{orderItemID, orderID, orgID},
			},
			{
				query: fmt.Sprintf(`DELETE FROM fleet_order_addons WHERE order_item_id = %s AND order_id = %s`, r.getPlaceholder(1), r.getPlaceholder(2)),
				args:  []interface{}{orderItemID, orderID},
			},
		}
		var deleted bool
		for _, c := range delCandidates {
			_, _ = database.TxExec(tx, "SAVEPOINT sp_del_addons_item")
			_, e := database.TxExec(tx, c.query, c.args...)
			if e == nil {
				_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_del_addons_item")
				deleted = true
				break
			}
			_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_del_addons_item")
			msg := strings.ToLower(e.Error())
			if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
				continue
			}
			return e
		}
		_ = deleted
	}

	if len(addonIDs) == 0 {
		return nil
	}

	addonWithCreatedBy := fmt.Sprintf(`
		INSERT INTO fleet_order_addons (order_addon_id, order_id, order_item_id, organization_id, addon_id, addon_price, created_at, created_by)
		SELECT %s, %s, %s, %s, uuid, addon_price, %s, %s FROM fleet_addon WHERE uuid = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7))
	addonWithoutCreatedBy := fmt.Sprintf(`
		INSERT INTO fleet_order_addons (order_addon_id, order_id, order_item_id, organization_id, addon_id, addon_price, created_at)
		SELECT %s, %s, %s, %s, uuid, addon_price, %s FROM fleet_addon WHERE uuid = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))

	addonWithCreatedByQty := fmt.Sprintf(`
		INSERT INTO fleet_order_addons (order_addon_id, order_id, order_item_id, organization_id, addon_id, addon_price, addon_qty, created_at, created_by)
		SELECT %s, %s, %s, %s, uuid, addon_price, %s, %s, %s FROM fleet_addon WHERE uuid = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8))
	addonWithoutCreatedByQty := fmt.Sprintf(`
		INSERT INTO fleet_order_addons (order_addon_id, order_id, order_item_id, organization_id, addon_id, addon_price, addon_qty, created_at)
		SELECT %s, %s, %s, %s, uuid, addon_price, %s, %s FROM fleet_addon WHERE uuid = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7))

	for _, a := range addonIDs {
		if strings.TrimSpace(a) == "" {
			continue
		}
		id := uuid2()
		_, _ = database.TxExec(tx, "SAVEPOINT sp_ins_addon_item")
		res, execErr := database.TxExec(tx, addonWithCreatedBy, id, orderID, orderItemID, orgID, now, createdBy, a)
		if execErr != nil {
			msg := strings.ToLower(execErr.Error())
			if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
				_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_ins_addon_item")
				res2, execErr2 := database.TxExec(tx, addonWithoutCreatedBy, id, orderID, orderItemID, orgID, now, a)
				if execErr2 == nil {
					_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_ins_addon_item")
					rows2, _ := res2.RowsAffected()
					if rows2 == 0 {
						return fmt.Errorf("addon not found: %s", a)
					}
					continue
				}

				msg2 := strings.ToLower(execErr2.Error())
				if strings.Contains(msg2, "unknown column") || strings.Contains(msg2, "does not exist") || strings.Contains(msg2, "column") {
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_ins_addon_item")
					addonQty := 1
					res3, execErr3 := database.TxExec(tx, addonWithCreatedByQty, id, orderID, orderItemID, orgID, addonQty, now, createdBy, a)
					if execErr3 == nil {
						_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_ins_addon_item")
						rows3, _ := res3.RowsAffected()
						if rows3 == 0 {
							return fmt.Errorf("addon not found: %s", a)
						}
						continue
					}
					msg3 := strings.ToLower(execErr3.Error())
					if strings.Contains(msg3, "unknown column") || strings.Contains(msg3, "does not exist") || strings.Contains(msg3, "column") {
						_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_ins_addon_item")
						res4, execErr4 := database.TxExec(tx, addonWithoutCreatedByQty, id, orderID, orderItemID, orgID, addonQty, now, a)
						if execErr4 != nil {
							return fmt.Errorf("insert addons: %w", execErr4)
						}
						_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_ins_addon_item")
						rows4, _ := res4.RowsAffected()
						if rows4 == 0 {
							return fmt.Errorf("addon not found: %s", a)
						}
						continue
					}
					return fmt.Errorf("insert addons: %w", execErr3)
				}
				return fmt.Errorf("insert addons: %w", execErr2)
			}
			return fmt.Errorf("insert addons: %w", execErr)
		}
		_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_ins_addon_item")
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return fmt.Errorf("addon not found: %s", a)
		}
	}
	return nil
}

func (r *FleetRepository) updateFleetOrderAddonPrice(tx *sql.Tx, orderID, orgID, updatedBy, addonID string, now time.Time) error {
	addonID = strings.TrimSpace(addonID)
	if addonID == "" {
		return nil
	}

	orgExpr := "organization_id = " + r.getPlaceholder(4)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(4)
	}

	candidates := []struct {
		query string
		args  []interface{}
	}{
		{
			query: fmt.Sprintf(`
				UPDATE fleet_order_addons
				SET addon_price = (SELECT addon_price FROM fleet_addon WHERE uuid = %s),
				    updated_at = %s, updated_by = %s
				WHERE order_id = %s AND addon_id = %s AND %s
				  AND EXISTS (SELECT 1 FROM fleet_addon WHERE uuid = %s)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), orgExpr, r.getPlaceholder(6)),
			args: []interface{}{addonID, now, updatedBy, orderID, addonID, orgID, addonID},
		},
		{
			query: fmt.Sprintf(`
				UPDATE fleet_order_addons
				SET addon_price = (SELECT addon_price FROM fleet_addon WHERE uuid = %s)
				WHERE order_id = %s AND addon_id = %s AND %s
				  AND EXISTS (SELECT 1 FROM fleet_addon WHERE uuid = %s)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), orgExpr, r.getPlaceholder(4)),
			args: []interface{}{addonID, orderID, addonID, orgID, addonID},
		},
		{
			query: fmt.Sprintf(`
				UPDATE fleet_order_addons
				SET addon_price = (SELECT addon_price FROM fleet_addon WHERE uuid = %s),
				    updated_at = %s, updated_by = %s
				WHERE order_id = %s AND addon_id = %s
				  AND EXISTS (SELECT 1 FROM fleet_addon WHERE uuid = %s)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6)),
			args: []interface{}{addonID, now, updatedBy, orderID, addonID, addonID},
		},
		{
			query: fmt.Sprintf(`
				UPDATE fleet_order_addons
				SET addon_price = (SELECT addon_price FROM fleet_addon WHERE uuid = %s)
				WHERE order_id = %s AND addon_id = %s
				  AND EXISTS (SELECT 1 FROM fleet_addon WHERE uuid = %s)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4)),
			args: []interface{}{addonID, orderID, addonID, addonID},
		},
	}

	for _, c := range candidates {
		_, _ = database.TxExec(tx, "SAVEPOINT sp_upd_addon_price")
		_, e := database.TxExec(tx, c.query, c.args...)
		if e == nil {
			_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_upd_addon_price")
			return nil
		}
		_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_upd_addon_price")
		msg := strings.ToLower(e.Error())
		if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
			continue
		}
		return e
	}
	return nil
}

func (r *FleetRepository) getFleetOrderAddonIDsByItem(tx *sql.Tx, orderID, orgID, orderItemID string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	orderItemID = strings.TrimSpace(orderItemID)
	if orderItemID == "" {
		return out, nil
	}

	orgExpr := "organization_id = " + r.getPlaceholder(3)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(3)
	}

	candidates := []struct {
		query string
		args  []interface{}
	}{
		{
			query: fmt.Sprintf(`SELECT addon_id FROM fleet_order_addons WHERE order_item_id = %s AND order_id = %s AND %s`, r.getPlaceholder(1), r.getPlaceholder(2), orgExpr),
			args:  []interface{}{orderItemID, orderID, orgID},
		},
		{
			query: fmt.Sprintf(`SELECT addon_id FROM fleet_order_addons WHERE order_item_id = %s AND order_id = %s`, r.getPlaceholder(1), r.getPlaceholder(2)),
			args:  []interface{}{orderItemID, orderID},
		},
	}

	for _, c := range candidates {
		rows, err := database.TxQuery(tx, c.query, c.args...)
		if err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
				continue
			}
			return nil, err
		}
		for rows.Next() {
			var id sql.NullString
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				return nil, err
			}
			if id.Valid && strings.TrimSpace(id.String) != "" {
				out[strings.TrimSpace(id.String)] = struct{}{}
			}
		}
		_ = rows.Close()
		return out, nil
	}

	return out, nil
}

func (r *FleetRepository) RecalculateFleetOrderTotal(tx *sql.Tx, orderID, organizationID string) (float64, error) {
	orgExpr := "organization_id = " + r.getPlaceholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(2)
	}

	sumQuery := fmt.Sprintf(`SELECT COALESCE(SUM(sub_total), 0), COUNT(1) FROM fleet_order_items WHERE order_id = %s AND %s`, r.getPlaceholder(1), orgExpr)
	var sumSubTotal float64
	var count int
	if err := database.TxQueryRow(tx, sumQuery, orderID, organizationID).Scan(&sumSubTotal, &count); err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, sql.ErrNoRows
	}
	if sumSubTotal < 0 {
		sumSubTotal = 0
	}
	total := sumSubTotal
	if total < 0 {
		total = 0
	}

	orgExprUpdate := "organization_id = " + r.getPlaceholder(3)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExprUpdate = "organization_id::text = " + r.getPlaceholder(3)
	}

	updateQuery := fmt.Sprintf(`UPDATE fleet_orders SET total_amount = %s WHERE order_id = %s AND %s`, r.getPlaceholder(1), r.getPlaceholder(2), orgExprUpdate)
	if _, err := database.TxExec(tx, updateQuery, total, orderID, organizationID); err != nil {
		return 0, err
	}
	return total, nil
}

type UpdatePartnerOrderInput struct {
	OrderID           string
	OrganizationID    string
	UpdatedBy         string
	FleetID           string
	PriceID           string
	StartDate         string
	EndDate           string
	PickupCityID      string
	PickupLocation    string
	UnitQty           int
	CustomerID        string
	TotalAmount       float64
	AdditionalAmount  float64
	DiscountAmount    float64
	DiscountTotal     float64
	AdditionalRequest string
	Fleets            []UpdatePartnerOrderFleetItem
	Itinerary         []UpdatePartnerOrderItineraryItem
}

type UpdatePartnerOrderFleetItem struct {
	OrderItemID  string
	FleetID      string
	PriceID      string
	Qty          int
	ChargeAmount float64
	AddonAmount  float64
	Discount     float64
	SubTotal     float64
	Addons       []string
}

type UpdatePartnerOrderItineraryItem struct {
	FleetItineraryID string
	Day              int
	CityID           string
	Location         string
}

func (r *FleetRepository) UpdatePartnerOrder(in UpdatePartnerOrderInput) (err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now()

	orgExpr2 := "organization_id = " + r.getPlaceholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr2 = "organization_id::text = " + r.getPlaceholder(2)
	}
	lockQuery := fmt.Sprintf("SELECT total_amount FROM fleet_orders WHERE order_id = %s AND %s FOR UPDATE", r.getPlaceholder(1), orgExpr2)
	var oldTotal float64
	if err := database.TxQueryRow(tx, lockQuery, in.OrderID, in.OrganizationID).Scan(&oldTotal); err != nil {
		return err
	}

	orgWhereAt := func(pos int) string {
		expr := "organization_id = " + r.getPlaceholder(pos)
		if r.driver == "postgres" || r.driver == "pgx" {
			expr = "organization_id::text = " + r.getPlaceholder(pos)
		}
		return expr
	}
	updateCandidates := []struct {
		query string
		args  []interface{}
	}{
		{
			query: fmt.Sprintf(`
				UPDATE fleet_orders
				SET fleet_id = %s, start_date = %s, end_date = %s, pickup_city_id = %s, pickup_location = %s,
				    unit_qty = %s, price_id = %s, total_amount = %s, additional_amount = %s, discount_amount = %s,
				    additional_request = %s, updated_at = %s, updated_by = %s
				WHERE order_id = %s AND %s
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
				r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10),
				r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14), orgWhereAt(15)),
			args: []interface{}{in.FleetID, in.StartDate, in.EndDate, in.PickupCityID, in.PickupLocation, in.UnitQty, in.PriceID, in.TotalAmount, in.AdditionalAmount, in.DiscountTotal, in.AdditionalRequest, now, in.UpdatedBy, in.OrderID, in.OrganizationID},
		},
		{
			query: fmt.Sprintf(`
				UPDATE fleet_orders
				SET fleet_id = %s, start_date = %s, end_date = %s, pickup_city_id = %s, pickup_location = %s,
				    unit_qty = %s, price_id = %s, total_amount = %s, additional_amount = %s, discount = %s,
				    additional_request = %s, updated_at = %s, updated_by = %s
				WHERE order_id = %s AND %s
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
				r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10),
				r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14), orgWhereAt(15)),
			args: []interface{}{in.FleetID, in.StartDate, in.EndDate, in.PickupCityID, in.PickupLocation, in.UnitQty, in.PriceID, in.TotalAmount, in.AdditionalAmount, in.DiscountTotal, in.AdditionalRequest, now, in.UpdatedBy, in.OrderID, in.OrganizationID},
		},
		{
			query: fmt.Sprintf(`
				UPDATE fleet_orders
				SET fleet_id = %s, start_date = %s, end_date = %s, pickup_city_id = %s, pickup_location = %s,
				    unit_qty = %s, price_id = %s, total_amount = %s, additional_amount = %s,
				    additional_request = %s, updated_at = %s, updated_by = %s
				WHERE order_id = %s AND %s
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
				r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9),
				r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), orgWhereAt(14)),
			args: []interface{}{in.FleetID, in.StartDate, in.EndDate, in.PickupCityID, in.PickupLocation, in.UnitQty, in.PriceID, in.TotalAmount, in.AdditionalAmount, in.AdditionalRequest, now, in.UpdatedBy, in.OrderID, in.OrganizationID},
		},
		{
			query: fmt.Sprintf(`
				UPDATE fleet_orders
				SET fleet_id = %s, start_date = %s, end_date = %s, pickup_city_id = %s, pickup_location = %s,
				    unit_qty = %s, price_id = %s, total_amount = %s, additional_amount = %s, updated_at = %s, updated_by = %s
				WHERE order_id = %s AND %s
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
				r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9),
				r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12), orgWhereAt(13)),
			args: []interface{}{in.FleetID, in.StartDate, in.EndDate, in.PickupCityID, in.PickupLocation, in.UnitQty, in.PriceID, in.TotalAmount, in.AdditionalAmount, now, in.UpdatedBy, in.OrderID, in.OrganizationID},
		},
		{
			query: fmt.Sprintf(`
				UPDATE fleet_orders
				SET fleet_id = %s, start_date = %s, end_date = %s, pickup_city_id = %s, pickup_location = %s,
				    unit_qty = %s, price_id = %s, total_amount = %s, additional_amount = %s,
				    additional_request = %s
				WHERE order_id = %s AND %s
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
				r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9),
				r.getPlaceholder(10), r.getPlaceholder(11), orgWhereAt(12)),
			args: []interface{}{in.FleetID, in.StartDate, in.EndDate, in.PickupCityID, in.PickupLocation, in.UnitQty, in.PriceID, in.TotalAmount, in.AdditionalAmount, in.AdditionalRequest, in.OrderID, in.OrganizationID},
		},
		{
			query: fmt.Sprintf(`
				UPDATE fleet_orders
				SET fleet_id = %s, start_date = %s, end_date = %s, pickup_city_id = %s, pickup_location = %s,
				    unit_qty = %s, price_id = %s, total_amount = %s, additional_amount = %s
				WHERE order_id = %s AND %s
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
				r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9),
				r.getPlaceholder(10), orgWhereAt(11)),
			args: []interface{}{in.FleetID, in.StartDate, in.EndDate, in.PickupCityID, in.PickupLocation, in.UnitQty, in.PriceID, in.TotalAmount, in.AdditionalAmount, in.OrderID, in.OrganizationID},
		},
	}

	var updated bool
	for _, c := range updateCandidates {
		_, _ = database.TxExec(tx, "SAVEPOINT sp_upd_orders")
		res, e := database.TxExec(tx, c.query, c.args...)
		if e == nil {
			_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_upd_orders")
			aff, _ := res.RowsAffected()
			if aff == 0 {
				return sql.ErrNoRows
			}
			updated = true
			break
		}
		_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_upd_orders")
		msg := strings.ToLower(e.Error())
		if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
			continue
		}
		return e
	}
	if !updated {
		return fmt.Errorf("failed to update fleet_orders")
	}

	if strings.TrimSpace(in.CustomerID) != "" {
		custUpdateCandidates := []struct {
			query string
			args  []interface{}
		}{
			{
				query: fmt.Sprintf(`
					UPDATE customer_orders
					SET customer_id = %s, updated_at = %s, updated_by = %s
					WHERE order_id = %s AND order_type = 1 AND %s
				`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), orgWhereAt(5)),
				args: []interface{}{in.CustomerID, now, in.UpdatedBy, in.OrderID, in.OrganizationID},
			},
			{
				query: fmt.Sprintf(`
					UPDATE customer_orders
					SET customer_id = %s
					WHERE order_id = %s AND order_type = 1 AND %s
				`, r.getPlaceholder(1), r.getPlaceholder(2), orgWhereAt(3)),
				args: []interface{}{in.CustomerID, in.OrderID, in.OrganizationID},
			},
		}
		var affected int64
		var lastErr error
		for _, c := range custUpdateCandidates {
			_, _ = database.TxExec(tx, "SAVEPOINT sp_upd_cust")
			res, e := database.TxExec(tx, c.query, c.args...)
			if e == nil {
				_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_upd_cust")
				affected, _ = res.RowsAffected()
				lastErr = nil
				break
			}
			_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_upd_cust")
			msg := strings.ToLower(e.Error())
			if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
				lastErr = e
				continue
			}
			return e
		}
		if lastErr != nil && affected == 0 {
			affected = 0
		}
		if affected == 0 {
			insertCustWithCreatedBy := fmt.Sprintf(`
				INSERT INTO customer_orders (order_id, customer_id, order_type, created_at, created_by, organization_id)
				VALUES (%s, %s, 1, %s, %s, %s)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5))
			insertCustWithoutCreatedBy := fmt.Sprintf(`
				INSERT INTO customer_orders (order_id, customer_id, order_type, created_at, organization_id)
				VALUES (%s, %s, 1, %s, %s)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))

			_, _ = database.TxExec(tx, "SAVEPOINT sp_ins_cust")
			_, e := database.TxExec(tx, insertCustWithCreatedBy, in.OrderID, in.CustomerID, now, in.UpdatedBy, in.OrganizationID)
			if e != nil {
				msg := strings.ToLower(e.Error())
				if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_ins_cust")
					_, e2 := database.TxExec(tx, insertCustWithoutCreatedBy, in.OrderID, in.CustomerID, now, in.OrganizationID)
					if e2 != nil {
						return e2
					}
				} else {
					return e
				}
			}
			_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_ins_cust")
		}
	}

	for _, it := range in.Itinerary {
		day := it.Day
		if day <= 0 {
			day = 1
		}
		if strings.TrimSpace(it.FleetItineraryID) == "" {
			id := uuid2()
			insertWithCreatedBy := fmt.Sprintf(`
				INSERT INTO fleet_order_itinerary (fleet_itinerary_id, order_id, day_num, city_id, location, organization_id, created_at, created_by)
				VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8))
			insertWithoutCreatedBy := fmt.Sprintf(`
				INSERT INTO fleet_order_itinerary (fleet_itinerary_id, order_id, day_num, city_id, location, organization_id, created_at)
				VALUES (%s, %s, %s, %s, %s, %s, %s)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7))

			_, _ = database.TxExec(tx, "SAVEPOINT sp_ins_it")
			_, e := database.TxExec(tx, insertWithCreatedBy, id, in.OrderID, day, it.CityID, it.Location, in.OrganizationID, now, in.UpdatedBy)
			if e != nil {
				msg := strings.ToLower(e.Error())
				if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_ins_it")
					_, e2 := database.TxExec(tx, insertWithoutCreatedBy, id, in.OrderID, day, it.CityID, it.Location, in.OrganizationID, now)
					if e2 != nil {
						return e2
					}
				} else {
					return e
				}
			}
			_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_ins_it")
			continue
		}

		updateItCandidates := []struct {
			query string
			args  []interface{}
		}{
			{
				query: fmt.Sprintf(`
					UPDATE fleet_order_itinerary
					SET day_num = %s, city_id = %s, location = %s, updated_at = %s, updated_by = %s
					WHERE fleet_itinerary_id = %s AND order_id = %s AND %s
				`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), orgWhereAt(8)),
				args: []interface{}{day, it.CityID, it.Location, now, in.UpdatedBy, it.FleetItineraryID, in.OrderID, in.OrganizationID},
			},
			{
				query: fmt.Sprintf(`
					UPDATE fleet_order_itinerary
					SET day_num = %s, city_id = %s, location = %s
					WHERE fleet_itinerary_id = %s AND order_id = %s AND %s
				`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), orgWhereAt(6)),
				args: []interface{}{day, it.CityID, it.Location, it.FleetItineraryID, in.OrderID, in.OrganizationID},
			},
		}
		var itUpdated bool
		for _, c := range updateItCandidates {
			_, _ = database.TxExec(tx, "SAVEPOINT sp_upd_it")
			res, e := database.TxExec(tx, c.query, c.args...)
			if e == nil {
				_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_upd_it")
				aff, _ := res.RowsAffected()
				if aff > 0 {
					itUpdated = true
				}
				break
			}
			_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_upd_it")
			msg := strings.ToLower(e.Error())
			if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
				continue
			}
			return e
		}
		_ = itUpdated
	}

	for _, f := range in.Fleets {
		q := f.Qty
		if q <= 0 {
			q = 1
		}
		addonIDsForItem := normalizeAddonIDs(f.Addons, "")
		if strings.TrimSpace(f.OrderItemID) == "" {
			id := uuid2()
			insertWithAll := fmt.Sprintf(`
				INSERT INTO fleet_order_items (order_item_id, organization_id, order_id, fleet_id, price_id, quantity, charge_amount, addon_amount, discount, sub_total, create_at, created_by, status)
				VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
				r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12))
			insertWithoutCreatedBy := fmt.Sprintf(`
				INSERT INTO fleet_order_items (order_item_id, organization_id, order_id, fleet_id, price_id, quantity, charge_amount, addon_amount, discount, sub_total, create_at, status)
				VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
				r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11))
			insertWithAllNoAddon := fmt.Sprintf(`
				INSERT INTO fleet_order_items (order_item_id, organization_id, order_id, fleet_id, price_id, quantity, charge_amount, discount, sub_total, create_at, created_by, status)
				VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
				r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11))
			insertWithoutCreatedByNoAddon := fmt.Sprintf(`
				INSERT INTO fleet_order_items (order_item_id, organization_id, order_id, fleet_id, price_id, quantity, charge_amount, discount, sub_total, create_at, status)
				VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1)
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
				r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10))

			_, _ = database.TxExec(tx, "SAVEPOINT sp_ins_item")
			_, e := database.TxExec(tx, insertWithAll, id, in.OrganizationID, in.OrderID, f.FleetID, f.PriceID, q, f.ChargeAmount, f.AddonAmount, f.Discount, f.SubTotal, now, in.UpdatedBy)
			if e != nil {
				msg := strings.ToLower(e.Error())
				if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
					_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_ins_item")
					_, e2 := database.TxExec(tx, insertWithoutCreatedBy, id, in.OrganizationID, in.OrderID, f.FleetID, f.PriceID, q, f.ChargeAmount, f.AddonAmount, f.Discount, f.SubTotal, now)
					if e2 != nil {
						msg2 := strings.ToLower(e2.Error())
						if strings.Contains(msg2, "unknown column") || strings.Contains(msg2, "does not exist") || strings.Contains(msg2, "column") {
							_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_ins_item")
							_, e3 := database.TxExec(tx, insertWithAllNoAddon, id, in.OrganizationID, in.OrderID, f.FleetID, f.PriceID, q, f.ChargeAmount, f.Discount, f.SubTotal, now, in.UpdatedBy)
							if e3 != nil {
								msg3 := strings.ToLower(e3.Error())
								if strings.Contains(msg3, "unknown column") || strings.Contains(msg3, "does not exist") || strings.Contains(msg3, "column") {
									_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_ins_item")
									_, e4 := database.TxExec(tx, insertWithoutCreatedByNoAddon, id, in.OrganizationID, in.OrderID, f.FleetID, f.PriceID, q, f.ChargeAmount, f.Discount, f.SubTotal, now)
									if e4 != nil {
										return e4
									}
								} else {
									return e3
								}
							}
						} else {
							return e2
						}
					}
				} else {
					return e
				}
			}
			_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_ins_item")

			if err := r.replaceFleetOrderItemAddons(tx, in.OrderID, in.OrganizationID, in.UpdatedBy, id, addonIDsForItem, now, false); err != nil {
				return err
			}
			continue
		}

		if len(addonIDsForItem) > 0 {
			existingAddonSet, err := r.getFleetOrderAddonIDsByItem(tx, in.OrderID, in.OrganizationID, strings.TrimSpace(f.OrderItemID))
			if err != nil {
				return err
			}
			for _, a := range addonIDsForItem {
				if _, ok := existingAddonSet[a]; !ok {
					continue
				}
				if err := r.updateFleetOrderAddonPrice(tx, in.OrderID, in.OrganizationID, in.UpdatedBy, a, now); err != nil {
					return err
				}
			}
			missing := make([]string, 0, len(addonIDsForItem))
			for _, a := range addonIDsForItem {
				if _, ok := existingAddonSet[a]; ok {
					continue
				}
				missing = append(missing, a)
			}
			if len(missing) > 0 {
				if err := r.replaceFleetOrderItemAddons(tx, in.OrderID, in.OrganizationID, in.UpdatedBy, strings.TrimSpace(f.OrderItemID), missing, now, false); err != nil {
					return err
				}
			}
		}

		updateItemCandidates := []struct {
			query string
			args  []interface{}
		}{
			{
				query: fmt.Sprintf(`
					UPDATE fleet_order_items
					SET fleet_id = %s, price_id = %s, quantity = %s, charge_amount = %s, addon_amount = %s, discount = %s, sub_total = %s, updated_at = %s, updated_by = %s
					WHERE order_item_id = %s AND order_id = %s AND %s
				`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7),
					r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), orgWhereAt(12)),
				args: []interface{}{f.FleetID, f.PriceID, q, f.ChargeAmount, f.AddonAmount, f.Discount, f.SubTotal, now, in.UpdatedBy, f.OrderItemID, in.OrderID, in.OrganizationID},
			},
			{
				query: fmt.Sprintf(`
					UPDATE fleet_order_items
					SET fleet_id = %s, price_id = %s, quantity = %s, charge_amount = %s, addon_amount = %s, discount = %s, sub_total = %s
					WHERE order_item_id = %s AND order_id = %s AND %s
				`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7),
					r.getPlaceholder(8), r.getPlaceholder(9), orgWhereAt(10)),
				args: []interface{}{f.FleetID, f.PriceID, q, f.ChargeAmount, f.AddonAmount, f.Discount, f.SubTotal, f.OrderItemID, in.OrderID, in.OrganizationID},
			},
			{
				query: fmt.Sprintf(`
					UPDATE fleet_order_items
					SET fleet_id = %s, price_id = %s, quantity = %s, charge_amount = %s, discount = %s, sub_total = %s, updated_at = %s, updated_by = %s
					WHERE order_item_id = %s AND order_id = %s AND %s
				`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6),
					r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), orgWhereAt(11)),
				args: []interface{}{f.FleetID, f.PriceID, q, f.ChargeAmount, f.Discount, f.SubTotal, now, in.UpdatedBy, f.OrderItemID, in.OrderID, in.OrganizationID},
			},
			{
				query: fmt.Sprintf(`
					UPDATE fleet_order_items
					SET fleet_id = %s, price_id = %s, quantity = %s, charge_amount = %s, discount = %s, sub_total = %s
					WHERE order_item_id = %s AND order_id = %s AND %s
				`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6),
					r.getPlaceholder(7), r.getPlaceholder(8), orgWhereAt(9)),
				args: []interface{}{f.FleetID, f.PriceID, q, f.ChargeAmount, f.Discount, f.SubTotal, f.OrderItemID, in.OrderID, in.OrganizationID},
			},
		}
		var itemUpdated bool
		for _, c := range updateItemCandidates {
			_, _ = database.TxExec(tx, "SAVEPOINT sp_upd_item")
			res, e := database.TxExec(tx, c.query, c.args...)
			if e == nil {
				_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_upd_item")
				aff, _ := res.RowsAffected()
				if aff > 0 {
					itemUpdated = true
				}
				break
			}
			_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_upd_item")
			msg := strings.ToLower(e.Error())
			if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
				continue
			}
			return e
		}
		_ = itemUpdated
	}

	sumOrgExpr := "organization_id = " + r.getPlaceholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		sumOrgExpr = "organization_id::text = " + r.getPlaceholder(2)
	}
	sumQuery := fmt.Sprintf(`SELECT COALESCE(SUM(sub_total), 0), COALESCE(SUM(COALESCE(discount, 0) * COALESCE(quantity, 0)), 0) FROM fleet_order_items WHERE order_id = %s AND %s`, r.getPlaceholder(1), sumOrgExpr)
	var sumSubTotal, sumDiscount float64
	sumErr := database.TxQueryRow(tx, sumQuery, in.OrderID, in.OrganizationID).Scan(&sumSubTotal, &sumDiscount)
	finalTotal := in.TotalAmount
	finalDiscount := in.DiscountTotal
	if sumErr == nil {
		finalDiscount = in.DiscountAmount + sumDiscount
		finalTotal = sumSubTotal
		if finalTotal < 0 {
			finalTotal = 0
		}
	}

	totalUpdateCandidates := []struct {
		query string
		args  []interface{}
	}{
		{
			query: fmt.Sprintf(`
				UPDATE fleet_orders
				SET total_amount = %s, additional_amount = %s, discount_amount = %s, updated_at = %s, updated_by = %s
				WHERE order_id = %s AND %s
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), orgWhereAt(7)),
			args: []interface{}{finalTotal, in.AdditionalAmount, finalDiscount, now, in.UpdatedBy, in.OrderID, in.OrganizationID},
		},
		{
			query: fmt.Sprintf(`
				UPDATE fleet_orders
				SET total_amount = %s, additional_amount = %s, discount = %s, updated_at = %s, updated_by = %s
				WHERE order_id = %s AND %s
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), orgWhereAt(7)),
			args: []interface{}{finalTotal, in.AdditionalAmount, finalDiscount, now, in.UpdatedBy, in.OrderID, in.OrganizationID},
		},
		{
			query: fmt.Sprintf(`
				UPDATE fleet_orders
				SET total_amount = %s, additional_amount = %s, updated_at = %s, updated_by = %s
				WHERE order_id = %s AND %s
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), orgWhereAt(6)),
			args: []interface{}{finalTotal, in.AdditionalAmount, now, in.UpdatedBy, in.OrderID, in.OrganizationID},
		},
		{
			query: fmt.Sprintf(`
				UPDATE fleet_orders
				SET total_amount = %s, additional_amount = %s
				WHERE order_id = %s AND %s
			`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), orgWhereAt(4)),
			args: []interface{}{finalTotal, in.AdditionalAmount, in.OrderID, in.OrganizationID},
		},
	}
	for _, c := range totalUpdateCandidates {
		_, _ = database.TxExec(tx, "SAVEPOINT sp_upd_totals")
		_, e := database.TxExec(tx, c.query, c.args...)
		if e == nil {
			_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_upd_totals")
			break
		}
		_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_upd_totals")
		msg := strings.ToLower(e.Error())
		if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
			continue
		}
		return e
	}

	if finalTotal != oldTotal {
		payUpdateCandidates := []struct {
			query string
			args  []interface{}
		}{
			{
				query: fmt.Sprintf(`
					UPDATE payment_orders
					SET total_amount = %s,
					    remaining_amount = CASE WHEN COALESCE(payment_amount, 0) >= %s THEN 0 ELSE %s - COALESCE(payment_amount, 0) END,
					    updated_at = %s, updated_by = %s
					WHERE order_id = %s AND order_type = 1 AND %s AND COALESCE(status, 0) > 0
				`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), orgWhereAt(7)),
				args: []interface{}{finalTotal, finalTotal, finalTotal, now, in.UpdatedBy, in.OrderID, in.OrganizationID},
			},
			{
				query: fmt.Sprintf(`
					UPDATE payment_orders
					SET total_amount = %s,
					    remaining_amount = CASE WHEN COALESCE(payment_amount, 0) >= %s THEN 0 ELSE %s - COALESCE(payment_amount, 0) END
					WHERE order_id = %s AND order_type = 1 AND %s AND COALESCE(status, 0) > 0
				`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), orgWhereAt(5)),
				args: []interface{}{finalTotal, finalTotal, finalTotal, in.OrderID, in.OrganizationID},
			},
		}
		for _, c := range payUpdateCandidates {
			_, _ = database.TxExec(tx, "SAVEPOINT sp_upd_payment_orders")
			_, e := database.TxExec(tx, c.query, c.args...)
			if e == nil {
				_, _ = database.TxExec(tx, "RELEASE SAVEPOINT sp_upd_payment_orders")
				break
			}
			_, _ = database.TxExec(tx, "ROLLBACK TO SAVEPOINT sp_upd_payment_orders")
			msg := strings.ToLower(e.Error())
			if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
				continue
			}
			return e
		}
	}

	return tx.Commit()
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
            fo.order_id, fo.created_at, fo.price_id, fo.status, fo.payment_status,
            f.fleet_name, 
            fp.rent_type, fp.price, 
            fo.unit_qty, fo.total_amount, COALESCE(fo.additional_amount, 0) as additional_amount,
            fo.pickup_location, fo.pickup_city_id, fo.start_date, fo.end_date,
            COALESCE(c.customer_name, '') as customer_name, COALESCE(c.customer_phone, '') as customer_phone, COALESCE(c.customer_email, '') as customer_email, COALESCE(c.customer_address, '') as customer_address, COALESCE(c.customer_city, 0) as customer_city,
            COALESCE(fo.additional_request, '') as additional_request
        FROM fleet_orders fo
        JOIN fleets f ON fo.fleet_id = f.uuid
        JOIN fleet_prices fp ON fo.price_id = fp.uuid
        INNER JOIN customer_orders co ON fo.order_id = co.order_id
		INNER JOIN customers c ON c.customer_id = co.customer_id
        WHERE fo.order_id = %s AND fo.organization_id = %s
    `, r.getPlaceholder(1), r.getPlaceholder(2))

	var res model.OrderDetailResponse
	var createdAt time.Time
	var pickupCityID string
	var startDate, endDate time.Time

	err := database.QueryRow(r.db, query, orderID, organizationID).Scan(
		&res.OrderID, &createdAt, &res.PriceID, &res.Status, &res.PaymentStatus,
		&res.FleetName,
		&res.RentType, &res.Price,
		&res.Quantity, &res.TotalAmount, &res.AdditionalAmount,
		&res.Pickup.PickupLocation, &pickupCityID, &startDate, &endDate,
		&res.Customer.CustomerName, &res.Customer.CustomerPhone, &res.Customer.CustomerEmail, &res.Customer.CustomerAddress, &res.Customer.CustomerCity,
		&res.AdditionalRequest,
	)
	if err != nil {
		fmt.Println("Error querying order detail:", err)
		return nil, err
	}
	res.OrderDate = createdAt.Format("2006-01-02 15:04:05")
	res.StatusLabel = configs.OrderStatus(res.Status).String()
	res.Pickup.PickupCity = pickupCityID
	if cityLabel, ok := getCitiesMap()[strings.TrimSpace(pickupCityID)]; ok {
		res.Pickup.CityLabel = cityLabel
	}
	if res.Customer.CustomerCity != 0 {
		if cityLabel, ok := getCitiesMap()[strconv.Itoa(res.Customer.CustomerCity)]; ok {
			res.Customer.CityLabel = cityLabel
		}
	}
	res.Pickup.StartDate = startDate.Format("2006-01-02")
	res.Pickup.EndDate = endDate.Format("2006-01-02")

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
				d.ID = cID
				if cityLabel, ok := getCitiesMap()[strings.TrimSpace(cID)]; ok {
					d.CityLabel = cityLabel
				}
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

	cityExpr := "city_id::text"

	itQuery := fmt.Sprintf(`SELECT fleet_itinerary_id, day_num, %s as city_id, location FROM fleet_order_itinerary WHERE order_id = %s AND organization_id = %s ORDER BY day_num`, cityExpr, r.getPlaceholder(1), r.getPlaceholder(2))
	iRows, itErr := r.db.Query(itQuery, orderID, organizationID)
	if itErr == nil {
		defer iRows.Close()
		items := make([]model.FleetOrderItineraryItem, 0)
		for iRows.Next() {
			var it model.FleetOrderItineraryItem
			if err := iRows.Scan(&it.FleetItineraryID, &it.Day, &it.CityID, &it.Destination); err == nil {
				if cityLabel, ok := getCitiesMap()[strings.TrimSpace(it.CityID)]; ok {
					it.CityLabel = cityLabel
				}
				items = append(items, it)
			}
		}
		if len(items) > 0 {
			res.Itinerary = items
		}
	}

	// Addons
	addonQuery := fmt.Sprintf(`
        SELECT fa.addon_name, fa.addon_desc, fa.addon_price
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
		if err := aRows.Scan(&a.AddonName, &a.AddonDesc, &a.AddonPrice); err == nil {
			res.Addon = append(res.Addon, a)
		}
	}

	// Payments
	paymentQuery := fmt.Sprintf(`
		SELECT 
			ba.bank_code, ba.account_name, ba.account_number, bl.name as bank_name, 
			op.payment_type, 0 as payment_percentage, op.payment_amount, op.total_amount, 
			op.remaining_amount, op.status, op.created_at, op.payment_id
		FROM payment_orders op
		LEFT JOIN organization_bank_accounts ba ON op.bank_account = ba.bank_account_id
		LEFT JOIN bank_lists bl ON ba.bank_code = bl.code
		WHERE op.order_id = %s AND op.order_type = 1 AND COALESCE(op.status, 0) > 0
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
			} else {
				fmt.Println("Payment scan error:", err)
			}
		}

		// Determine overall payment status
		if !hasPayment {
			res.PaymentStatus = 2
		} else if allStatus1 {
			res.PaymentStatus = 1
		}
	}

	return &res, nil
}

func (r *FleetRepository) DeleteFleetOrderAddon(orderID, orderItemID, addonID, orgID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	orgExpr := "organization_id = " + r.getPlaceholder(4)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(4)
	}

	// 1. Recalculate addon_amount (excluding the to-be-deleted one)
	sumAddonQuery := fmt.Sprintf(`SELECT COALESCE(SUM(addon_price), 0) FROM fleet_order_addons WHERE order_id = %s AND order_item_id = %s AND addon_id != %s AND %s`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), orgExpr)
	var newAddonAmount float64
	err = database.TxQueryRow(tx, sumAddonQuery, orderID, orderItemID, addonID, orgID).Scan(&newAddonAmount)
	if err != nil && err != sql.ErrNoRows {
		sumAddonQueryLegacy := fmt.Sprintf(`SELECT COALESCE(SUM(addon_price), 0) FROM fleet_order_addons WHERE order_id = %s AND order_item_id = %s AND addon_id != %s`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
		err = database.TxQueryRow(tx, sumAddonQueryLegacy, orderID, orderItemID, addonID).Scan(&newAddonAmount)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
	}

	// 2. Update fleet_order_items.addon_amount
	orgExprItem := "organization_id = " + r.getPlaceholder(3)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExprItem = "organization_id::text = " + r.getPlaceholder(3)
	}
	updateItemQuery := fmt.Sprintf(`UPDATE fleet_order_items SET addon_amount = %s WHERE order_item_id = %s AND %s`, r.getPlaceholder(1), r.getPlaceholder(2), orgExprItem)
	_, err = database.TxExec(tx, updateItemQuery, newAddonAmount, orderItemID, orgID)
	if err != nil {
		updateItemQueryLegacy := fmt.Sprintf(`UPDATE fleet_order_items SET addon_amount = %s WHERE order_item_id = %s`, r.getPlaceholder(1), r.getPlaceholder(2))
		_, err = database.TxExec(tx, updateItemQueryLegacy, newAddonAmount, orderItemID)
		if err != nil {
			return err
		}
	}

	// 3. Recalculate sub_total
	var priceID string
	var quantity, chargeAmount, discount float64
	fetchItemQuery := fmt.Sprintf(`SELECT price_id, quantity, COALESCE(charge_amount, 0), COALESCE(discount, 0) FROM fleet_order_items WHERE order_item_id = %s`, r.getPlaceholder(1))
	err = database.TxQueryRow(tx, fetchItemQuery, orderItemID).Scan(&priceID, &quantity, &chargeAmount, &discount)
	if err != nil {
		return err
	}

	var originalPrice float64
	priceQuery := fmt.Sprintf(`SELECT price FROM fleet_prices WHERE uuid = %s`, r.getPlaceholder(1))
	err = database.TxQueryRow(tx, priceQuery, priceID).Scan(&originalPrice)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	newSubTotal := (originalPrice + chargeAmount + newAddonAmount - discount) * quantity
	if newSubTotal < 0 {
		newSubTotal = 0
	}

	updateSubTotalQuery := fmt.Sprintf(`UPDATE fleet_order_items SET sub_total = %s WHERE order_item_id = %s`, r.getPlaceholder(1), r.getPlaceholder(2))
	_, err = database.TxExec(tx, updateSubTotalQuery, newSubTotal, orderItemID)
	if err != nil {
		return err
	}

	// 4. Recalculate total_amount in fleet_orders
	_, err = r.RecalculateFleetOrderTotal(tx, orderID, orgID)
	if err != nil {
		return err
	}

	// 5. Delete from fleet_order_addons
	deleteAddonQuery := fmt.Sprintf(`DELETE FROM fleet_order_addons WHERE order_id = %s AND order_item_id = %s AND addon_id = %s AND %s`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), orgExpr)
	_, err = database.TxExec(tx, deleteAddonQuery, orderID, orderItemID, addonID, orgID)
	if err != nil {
		deleteAddonQueryLegacy := fmt.Sprintf(`DELETE FROM fleet_order_addons WHERE order_id = %s AND order_item_id = %s AND addon_id = %s`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
		_, err = database.TxExec(tx, deleteAddonQueryLegacy, orderID, orderItemID, addonID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
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

func (r *FleetRepository) SyncFleetOrderTotalAmountFromItems(orderID, organizationID string) (float64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	orderExpr := "order_id::text = " + r.getPlaceholder(1)
	orgExpr := "organization_id::text = " + r.getPlaceholder(2)

	lockQuery := fmt.Sprintf("SELECT total_amount FROM fleet_orders WHERE %s AND %s FOR UPDATE", orderExpr, orgExpr)
	var oldTotal float64
	if e := database.TxQueryRow(tx, lockQuery, orderID, organizationID).Scan(&oldTotal); e != nil {
		err = e
		return 0, err
	}

	countQuery := fmt.Sprintf("SELECT COUNT(1) FROM fleet_order_items WHERE %s AND %s AND COALESCE(status, 1) > 0", orderExpr, orgExpr)
	var itemCount int
	if e := database.TxQueryRow(tx, countQuery, orderID, organizationID).Scan(&itemCount); e != nil {
		err = e
		return 0, err
	}
	if itemCount == 0 {
		err = sql.ErrNoRows
		return 0, err
	}

	joinExpr := "fp.uuid::text = oi.price_id::text"
	var sumItems float64
	useLegacyAddonSum := false
	itemsQueryNew := fmt.Sprintf(`
		SELECT COALESCE(SUM(
			(COALESCE(fp.price, 0) * COALESCE(oi.quantity, 0)) +
			(COALESCE(oi.charge_amount, 0) * COALESCE(oi.quantity, 0)) +
			(COALESCE(oi.addon_amount, 0) * COALESCE(oi.quantity, 0)) -
			(COALESCE(oi.discount, 0) * COALESCE(oi.quantity, 0))
		), 0)
		FROM fleet_order_items oi
		LEFT JOIN fleet_prices fp ON %s
		WHERE %s AND %s AND COALESCE(oi.status, 1) > 0
	`, joinExpr, "oi."+orderExpr, "oi."+orgExpr)
	if e := database.TxQueryRow(tx, itemsQueryNew, orderID, organizationID).Scan(&sumItems); e != nil {
		msg := strings.ToLower(e.Error())
		if strings.Contains(msg, "unknown column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "column") {
			useLegacyAddonSum = true
			itemsQueryOld := fmt.Sprintf(`
				SELECT COALESCE(SUM((COALESCE(fp.price, 0) * COALESCE(oi.quantity, 0)) + COALESCE(oi.charge_amount, 0) - COALESCE(oi.discount, 0)), 0)
				FROM fleet_order_items oi
				LEFT JOIN fleet_prices fp ON %s
				WHERE %s AND %s AND COALESCE(oi.status, 1) > 0
			`, joinExpr, "oi."+orderExpr, "oi."+orgExpr)
			if e2 := database.TxQueryRow(tx, itemsQueryOld, orderID, organizationID).Scan(&sumItems); e2 != nil {
				err = e2
				return 0, err
			}
		} else {
			err = e
			return 0, err
		}
	}
	if sumItems < 0 {
		sumItems = 0
	}

	var sumAddons float64
	if useLegacyAddonSum {
		addonsQuery := fmt.Sprintf(`SELECT COALESCE(SUM(COALESCE(addon_price, 0) * COALESCE(addon_qty, 1)), 0) FROM fleet_order_addons WHERE %s AND %s`, orderExpr, orgExpr)
		_ = database.TxQueryRow(tx, addonsQuery, orderID, organizationID).Scan(&sumAddons)
		if sumAddons < 0 {
			sumAddons = 0
		}
	}

	total := sumItems + sumAddons
	if total < 0 {
		total = 0
	}

	diff := total - oldTotal
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.0001 {
		updOrderExpr := "order_id::text = " + r.getPlaceholder(2)
		updOrgExpr := "organization_id::text = " + r.getPlaceholder(3)

		updateQuery := fmt.Sprintf("UPDATE fleet_orders SET total_amount = %s WHERE %s AND %s", r.getPlaceholder(1), updOrderExpr, updOrgExpr)
		if _, e := database.TxExec(tx, updateQuery, total, orderID, organizationID); e != nil {
			err = e
			return 0, err
		}
	}

	if e := tx.Commit(); e != nil {
		err = e
		return 0, err
	}
	return total, nil
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
	orgExpr := "organization_id::text = " + r.getPlaceholder(2)

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

func (r *FleetRepository) generatePaymentOrderInvoiceNumber(tx *sql.Tx, orderType int, organizationID string, now time.Time) (string, error) {
	return utils.GenerateInvoiceNumberTx(tx, r.driver, organizationID, orderType, now)
}

func (r *FleetRepository) InsertServiceOrderPayment(req *model.CreateServiceOrderPaymentRequest, totalAmount, remainingAmount float64) (string, string, error) {
	paymentID := uuid.New().String()
	now := time.Now()
	tx, err := r.db.Begin()
	if err != nil {
		return "", "", err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	invoiceNumber, err := r.generatePaymentOrderInvoiceNumber(tx, req.OrderType, req.OrganizationID, now)
	if err != nil {
		return "", "", err
	}

	status := req.PaymentType
	if math.Abs(remainingAmount) < 0.0001 {
		status = 1003
	}

	var bankIDArg interface{}
	if req.BankID != nil {
		bankIDArg = *req.BankID
	}
	var bankAccountArg interface{}
	if req.BankAccount != "" {
		bankAccountArg = req.BankAccount
	}

	query := fmt.Sprintf(`
		INSERT INTO payment_orders
			(payment_id, invoice_number, order_type, order_id, organization_id, payment_type, payment_method, bank_id, bank_account, payment_amount, total_amount, remaining_amount, evidence_file, status, created_at, created_by)
		VALUES
			(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10),
		r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14), r.getPlaceholder(15))

	_, err = database.TxExec(
		tx,
		query,
		paymentID,
		invoiceNumber,
		req.OrderType,
		req.OrderID,
		req.OrganizationID,
		status,
		req.PaymentMethod,
		bankIDArg,
		bankAccountArg,
		req.PaymentAmount,
		totalAmount,
		remainingAmount,
		req.EvidenceFile,
		now,
		req.CreatedBy,
	)
	if err != nil {
		return "", "", err
	}

	transactionOrderType := 3
	switch req.OrderType {
	case 1:
		transactionOrderType = 1
	case 2:
		transactionOrderType = 2
	}

	transactionType := 1

	transactionCategory := strings.TrimSpace(req.TransactionCategory)
	if transactionCategory == "" {
		transactionCategory = "TRX01"
	}
	var transactionItemArg interface{}
	if strings.TrimSpace(req.TransactionItem) != "" {
		transactionItemArg = req.TransactionItem
	}

	description := ""
	if transactionOrderType == 1 || transactionOrderType == 2 {
		description = "Transaction with Order ID " + req.OrderID
	}

	transactionQuery := fmt.Sprintf(`
		INSERT INTO transactions
			(transaction_id, transaction_type, order_type, invoice_number, description, transaction_date, payment_type, amount, organization_id, transaction_category, transaction_item, payment_method, created_at, created_by, status, reference_id)
		VALUES
			(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6),
		r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14), r.getPlaceholder(15))

	_, err = database.TxExec(
		tx,
		transactionQuery,
		uuid.New().String(),
		transactionType,
		transactionOrderType,
		invoiceNumber,
		description,
		now,
		status,
		req.PaymentAmount,
		req.OrganizationID,
		transactionCategory,
		transactionItemArg,
		req.PaymentMethod,
		now,
		req.CreatedBy,
		req.OrderID,
	)
	if err != nil {
		return "", "", err
	}
	if err := tx.Commit(); err != nil {
		return "", "", err
	}
	return paymentID, invoiceNumber, nil
}

func (r *FleetRepository) FindFleetOrderIDByPrefix(prefix, organizationID string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", sql.ErrNoRows
	}

	orgExpr := "organization_id::text = " + r.getPlaceholder(2)

	likeExpr := "order_id ILIKE " + r.getPlaceholder(1)

	query := fmt.Sprintf(`
		SELECT order_id
		FROM fleet_orders
		WHERE %s AND %s
		ORDER BY order_id
		LIMIT 2
	`, likeExpr, orgExpr)

	rows, err := database.Query(r.db, query, prefix+"%", organizationID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", sql.ErrNoRows
	}
	if len(ids) > 1 {
		return "", fmt.Errorf("multiple orders match prefix")
	}
	return ids[0], nil
}

func (r *FleetRepository) ListFleetOrderPaymentHistory(orderID, organizationID string) ([]model.PaymentOrderHistoryRow, error) {
	orgExpr := "organization_id::text = " + r.getPlaceholder(2)

	query := fmt.Sprintf(`
		SELECT
			payment_type,
			payment_method,
			payment_amount,
			total_amount,
			remaining_amount,
			COALESCE(status, 0),
			created_at,
			settled_at,
			COALESCE(invoice_number, ''),
			COALESCE(notes, '')
		FROM payment_orders
		WHERE order_id = %s AND order_type = 1 AND %s AND COALESCE(status, 0) > 0
		ORDER BY created_at ASC
	`, r.getPlaceholder(1), orgExpr)

	rows, err := database.Query(r.db, query, orderID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.PaymentOrderHistoryRow, 0)
	for rows.Next() {
		var it model.PaymentOrderHistoryRow
		if err := rows.Scan(
			&it.PaymentType,
			&it.PaymentMethod,
			&it.PaymentAmount,
			&it.TotalAmount,
			&it.RemainingAmount,
			&it.Status,
			&it.CreatedAt,
			&it.SettledAt,
			&it.InvoiceNumber,
			&it.Notes,
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

func (r *FleetRepository) ListPaymentOrders(orderID string, orderType int, organizationID string) ([]model.PaymentOrderRow, error) {
	orgExpr := "po.organization_id::text = " + r.getPlaceholder(3)
	query := fmt.Sprintf(`
		SELECT
			po.payment_id,
			po.order_type,
			po.order_id,
			po.organization_id,
			po.payment_type,
			po.payment_method,
			po.bank_id,
			COALESCE(bl.name, '') AS bank_name,
			po.bank_account,
			po.payment_amount,
			po.total_amount,
			po.remaining_amount,
			po.evidence_file,
			COALESCE(po.status, 0),
			po.created_at,
			po.created_by
		FROM payment_orders po
		LEFT JOIN bank_list bl ON bl.code::text = po.bank_id::text
		WHERE po.order_id = %s AND po.order_type = %s AND %s AND COALESCE(po.status, 0) > 0
		ORDER BY po.created_at DESC
	`, r.getPlaceholder(1), r.getPlaceholder(2), orgExpr)

	rows, err := database.Query(r.db, query, orderID, orderType, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.PaymentOrderRow, 0)
	for rows.Next() {
		var it model.PaymentOrderRow
		var bankName sql.NullString
		if err := rows.Scan(
			&it.PaymentID,
			&it.OrderType,
			&it.OrderID,
			&it.OrganizationID,
			&it.PaymentType,
			&it.PaymentMethod,
			&it.BankID,
			&bankName,
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
		fmt.Println("bankName:", bankName)
		if !bankName.Valid || bankName.String == "" {
			it.BankName = sql.NullString{}
		} else {
			it.BankName.Valid = true
			it.BankName = bankName
		}
		fmt.Println("it:", it)
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FleetRepository) GetLatestPaymentOrder(orderID string, orderType int, organizationID string) (*model.PaymentOrderRow, error) {
	orgExpr := "organization_id::text = " + r.getPlaceholder(3)
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
	fmt.Println("it:", it)
	return &it, nil
}

func (r *FleetRepository) GetFleetOrderItemTotals(orderID, organizationID string) (float64, float64, float64, float64, error) {
	orgExpr := "foi.organization_id::text = " + r.getPlaceholder(2)

	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(COALESCE(addon_amount, 0)), 0) as total_addon,
			COALESCE(SUM(COALESCE(discount, 0)), 0) as total_discount,
			COALESCE(SUM(COALESCE(charge_amount, 0)), 0) as total_charge,
			COALESCE(SUM(COALESCE(payment_amount, 0)), 0) as total_payment
		FROM fleet_order_items foi
		LEFT JOIN payment_orders po ON foi.order_id = po.order_id AND po.order_type = 1 AND po.status = 1
		WHERE foi.order_id = %s AND %s AND po.status > 0
	`, r.getPlaceholder(1), orgExpr)

	var totalAddon, totalDiscount, totalCharge, totalPayment float64
	if err := database.QueryRow(r.db, query, orderID, organizationID).Scan(&totalAddon, &totalDiscount, &totalCharge, &totalPayment); err != nil {
		return 0, 0, 0, 0, err
	}
	return totalAddon, totalDiscount, totalCharge, totalPayment, nil
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
	orgExpr := r.getPlaceholder(1)
	scheduleIDExpr := "s.schedule_id"
	if r.driver == "postgres" || r.driver == "pgx" {
		scheduleIDExpr = "s.schedule_id::text"
	}

	base := fmt.Sprintf(`
        SELECT 
			fo.order_id, f.fleet_name, f.thumbnail,
			COALESCE((
				SELECT c.customer_name
				FROM customer_orders co
				INNER JOIN customers c ON c.customer_id = co.customer_id AND c.organization_id = f.organization_id
				WHERE co.order_id = fo.order_id
				ORDER BY co.created_at DESC
				LIMIT 1
			), '') as customer_name,
			COALESCE((
				SELECT c.customer_phone
				FROM customer_orders co
				INNER JOIN customers c ON c.customer_id = co.customer_id AND c.organization_id = f.organization_id
				WHERE co.order_id = fo.order_id
				ORDER BY co.created_at DESC
				LIMIT 1
			), '') as customer_phone,
			fo.start_date, fo.end_date, fo.unit_qty, fo.payment_status, fo.status,
			p.duration, p.uom, fo.total_amount, p.rent_type, fo.created_at as order_date,
			COALESCE((
				SELECT po.payment_type
				FROM payment_orders po
				WHERE po.order_id = fo.order_id
				  AND po.order_type = 1
				  AND po.organization_id = f.organization_id
				  AND COALESCE(po.status, 0) > 0
				ORDER BY po.created_at DESC
				LIMIT 1
			), 0) as latest_payment_type,
			COALESCE((
				SELECT %s
				FROM schedules s
				WHERE s.order_id = fo.order_id
				ORDER BY s.created_at DESC
				LIMIT 1
			), '') as schedule_id
        FROM fleet_orders fo 
        INNER JOIN fleets f ON fo.fleet_id = f.uuid 
        INNER JOIN fleet_prices p ON p.uuid = fo.price_id 
        WHERE f.organization_id = %s
    `, scheduleIDExpr, orgExpr)
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
			cond += fmt.Sprintf(" AND fo.created_at < %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.OrderDateTo)
		}
		if filter.HasPaymentStatus {
			cond += fmt.Sprintf(" AND fo.payment_status = %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.PaymentStatus)
		}
		if v := strings.TrimSpace(filter.Search); v != "" {
			op := "LIKE"
			if r.driver == "postgres" || r.driver == "pgx" {
				op = "ILIKE"
			}
			like := "%" + v + "%"
			pos := len(args) + 1
			cond += fmt.Sprintf(
				" AND (fo.order_id %s %s OR f.fleet_name %s %s OR EXISTS (SELECT 1 FROM customer_orders co INNER JOIN customers c ON c.customer_id = co.customer_id AND c.organization_id = f.organization_id WHERE co.order_id = fo.order_id AND c.customer_name %s %s))",
				op, r.getPlaceholder(pos),
				op, r.getPlaceholder(pos+1),
				op, r.getPlaceholder(pos+2),
			)
			args = append(args, like, like, like)
		}
	}
	query := base + cond + " ORDER BY fo.created_at DESC"

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
		var scheduleID sql.NullString
		var orderDate, createdAt time.Time
		if err := rows.Scan(
			&it.OrderID, &it.FleetName, &it.Thumbnail, &it.CustomerName, &it.CustomerPhone,
			&startDate, &endDate, &it.UnitQty, &it.PaymentStatus, &it.Status,
			&it.Duration, &it.Uom, &it.TotalAmount, &rentType, &orderDate,
			&latestPaymentType, &scheduleID,
		); err != nil {
			return nil, err
		}
		it.StartDate = startDate.Format("2006-01-02")
		it.EndDate = endDate.Format("2006-01-02")
		it.LatestPaymentType = latestPaymentType
		if !scheduleID.Valid || scheduleID.String == "" {
			it.ScheduleID = ""
		} else {
			it.ScheduleID = scheduleID.String
		}
		it.OrderDate = orderDate.Format("2006-01-02")
		it.CreatedAt = createdAt.Format("2006-01-02")
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
	orgExpr := r.getPlaceholder(1)
	base := fmt.Sprintf(`
        SELECT 
            COUNT(DISTINCT fo.order_id) AS total_orders,
            COUNT(DISTINCT CASE WHEN fo.payment_status = 1 THEN fo.order_id END) AS paid,
            COUNT(DISTINCT CASE WHEN fo.payment_status = 2 THEN fo.order_id END) AS unpaid,
            COUNT(DISTINCT CASE WHEN fo.payment_status IN (3,4) THEN fo.order_id END) AS pending,
            COALESCE(SUM(CASE WHEN fo.payment_status IN (1,4) THEN fo.total_amount ELSE 0 END), 0) AS revenue,
            COUNT(DISTINCT CASE WHEN fo.start_date <= CURRENT_DATE AND fo.end_date >= CURRENT_DATE THEN fo.order_id END) AS ongoing
        FROM fleet_orders fo
        INNER JOIN fleets f ON fo.fleet_id = f.uuid
        WHERE f.organization_id = %s
    `, orgExpr)
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
			cond += fmt.Sprintf(" AND fo.created_at < %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.OrderDateTo)
		}
		if filter.HasPaymentStatus {
			cond += fmt.Sprintf(" AND fo.payment_status = %s", r.getPlaceholder(len(args)+1))
			args = append(args, filter.PaymentStatus)
		}
		if v := strings.TrimSpace(filter.Search); v != "" {
			op := "LIKE"
			if r.driver == "postgres" || r.driver == "pgx" {
				op = "ILIKE"
			}
			like := "%" + v + "%"
			pos := len(args) + 1
			cond += fmt.Sprintf(
				" AND (fo.order_id %s %s OR f.fleet_name %s %s OR EXISTS (SELECT 1 FROM customer_orders co INNER JOIN customers c ON c.customer_id = co.customer_id AND c.organization_id = f.organization_id WHERE co.order_id = fo.order_id AND c.customer_name %s %s))",
				op, r.getPlaceholder(pos),
				op, r.getPlaceholder(pos+1),
				op, r.getPlaceholder(pos+2),
			)
			args = append(args, like, like, like)
		}
	}
	query := base + cond
	row := r.db.QueryRow(query, args...)
	var s model.PartnerOrderSummary
	if err := row.Scan(&s.TotalOrders, &s.Paid, &s.Unpaid, &s.Pending, &s.Revenue, &s.Ongoing); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *FleetRepository) GetPartnerOrderFleetItems(organizationId, orderId string) ([]model.OrderDetailFleetItem, error) {
	priceJoinExpr := "p.uuid::text = oi.price_id::text"
	fleetJoinExpr := "f.uuid::text = oi.fleet_id::text"

	query := fmt.Sprintf(`
		SELECT oi.order_item_id, oi.order_id, oi.fleet_id, f.fleet_name,
		tp.label as fleet_type, oi.price_id, p.price, oi.quantity,
		COALESCE(oi.charge_amount, 0) as charge_amount, COALESCE(oi.discount, 0) as discount, COALESCE(oi.sub_total, 0) as sub_total
		FROM fleet_order_items oi
		INNER JOIN fleet_orders o ON oi.order_id = o.order_id
		INNER JOIN fleet_prices p ON %[3]s
		INNER JOIN fleets f ON %[4]s
		INNER JOIN fleet_types tp ON tp.id = f.fleet_type
		WHERE oi.organization_id = %[1]s AND oi.order_id = %[2]s
	`, r.getPlaceholder(1), r.getPlaceholder(2), priceJoinExpr, fleetJoinExpr)

	rows, err := database.Query(r.db, query, organizationId, orderId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.OrderDetailFleetItem
	for rows.Next() {
		var item model.OrderDetailFleetItem
		var fleetType sql.NullString
		if err := rows.Scan(
			&item.OrderItemID,
			&item.OrderID,
			&item.FleetID,
			&item.FleetName,
			&fleetType,
			&item.PriceID,
			&item.Price,
			&item.Quantity,
			&item.ChargeAmount,
			&item.Discount,
			&item.SubTotal,
		); err != nil {
			return nil, err
		}
		if fleetType.Valid {
			item.FleetType = fleetType.String
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return items, nil
	}

	orgExpr := "oi.organization_id::text = " + r.getPlaceholder(1)
	orderExpr := "oi.order_id::text = " + r.getPlaceholder(2)
	itemJoinExpr := "oi.order_id::text = foa.order_id::text"
	addonJoinExpr := "foa.addon_id::text = a.uuid::text"

	addonsQuery := fmt.Sprintf(`
		SELECT foa.order_addon_id, COALESCE(a.addon_name, ''), COALESCE(a.addon_desc, ''), COALESCE(foa.addon_price, 0)
		FROM fleet_orders oi
		INNER JOIN fleet_order_addons foa ON %s
		INNER JOIN fleet_addon a ON %s
		WHERE %s AND %s
	`, itemJoinExpr, addonJoinExpr, orgExpr, orderExpr)

	aRows, err := database.Query(r.db, addonsQuery, organizationId, orderId)
	if err != nil {
		return nil, err
	}
	defer aRows.Close()

	addonsByItem := make(map[string][]model.OrderDetailAddon)
	addonAmountByItem := make(map[string]float64)
	for aRows.Next() {
		var orderItemID string
		var a model.OrderDetailAddon
		if err := aRows.Scan(&orderItemID, &a.AddonName, &a.AddonDesc, &a.AddonPrice); err != nil {
			return nil, err
		}
		addonsByItem[orderItemID] = append(addonsByItem[orderItemID], a)
		addonAmountByItem[orderItemID] += a.AddonPrice
	}
	if err := aRows.Err(); err != nil {
		return nil, err
	}

	for i := range items {
		if v, ok := addonsByItem[items[i].OrderItemID]; ok {
			items[i].Addons = v
		}
		if v, ok := addonAmountByItem[items[i].OrderItemID]; ok {
			items[i].AddonAmount = v
		}
	}

	return items, nil
}

func (r *FleetRepository) GetPartnerOrderDetail(orderID, orgID string) (*model.OrderDetailResponse, error) {
	customerCityExpr := "COALESCE(c.customer_city::text, '')"
	customerIDExpr := "COALESCE(c.customer_id::text, '')"

	fleetJoinExpr := "fo.fleet_id::text = f.uuid::text"
	priceJoinExpr := "fo.price_id::text = fp.uuid::text"
	customerOrderOrgExpr := "co.organization_id::text = f.organization_id::text"
	customerJoinExpr := "c.customer_id::text = co.customer_id::text"
	customerOrgExpr := "c.organization_id::text = f.organization_id::text"
	orgWhereExpr := "f.organization_id::text = " + r.getPlaceholder(2)

	query := fmt.Sprintf(`
        SELECT 
            fo.order_id, fo.fleet_id, fo.created_at, fo.price_id, fo.payment_status, fo.status,
            f.fleet_name, 
            fp.rent_type, fp.price, 
            fo.unit_qty, fo.total_amount, COALESCE(fo.additional_amount, 0) as additional_amount,
            fo.pickup_location, fo.pickup_city_id, fo.start_date, fo.end_date,
			%[8]s as customer_id,
            COALESCE(c.customer_name, '') as customer_name,
			COALESCE(c.customer_phone, '') as customer_phone,
			COALESCE(c.customer_email, '') as customer_email,
			COALESCE(c.customer_address, '') as customer_address,
			%[1]s as customer_city,
			COALESCE(fo.additional_request, '') as additional_request,
			fo.updated_at
        FROM fleet_orders fo
        JOIN fleets f ON %[2]s
        JOIN fleet_prices fp ON %[3]s
		LEFT JOIN customer_orders co ON co.order_id = fo.order_id AND %[4]s
		LEFT JOIN customers c ON %[9]s AND %[5]s
        WHERE fo.order_id = %[6]s AND %[7]s
    `, customerCityExpr, fleetJoinExpr, priceJoinExpr, customerOrderOrgExpr, customerOrgExpr, r.getPlaceholder(1), orgWhereExpr, customerIDExpr, customerJoinExpr)

	var res model.OrderDetailResponse
	var createdAt time.Time
	var pickupCityID string
	var startDate, endDate time.Time
	var updatedAt sql.NullTime

	err := r.db.QueryRow(query, orderID, orgID).Scan(
		&res.OrderID, &res.FleetID, &createdAt, &res.PriceID, &res.PaymentStatus, &res.Status,
		&res.FleetName,
		&res.RentType, &res.Price,
		&res.Quantity, &res.TotalAmount, &res.AdditionalAmount,
		&res.Pickup.PickupLocation, &pickupCityID, &startDate, &endDate,
		&res.Customer.CustomerID, &res.Customer.CustomerName, &res.Customer.CustomerPhone, &res.Customer.CustomerEmail, &res.Customer.CustomerAddress, &res.Customer.CustomerCity,
		&res.AdditionalRequest, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found or access denied")
		}
		return nil, err
	}
	res.OrderDate = createdAt.Format("2006-01-02 15:04:05")
	if updatedAt.Valid {
		res.UpdatedAt = updatedAt.Time.Format("2006-01-02 15:04:05")
	} else {
		res.UpdatedAt = ""
	}
	res.StatusLabel = configs.OrderStatus(res.Status).String()
	res.Pickup.PickupCity = pickupCityID
	if cityLabel, ok := getCitiesMap()[strings.TrimSpace(pickupCityID)]; ok {
		res.Pickup.CityLabel = cityLabel
	}
	if res.Customer.CustomerCity != 0 {
		if cityLabel, ok := getCitiesMap()[strconv.Itoa(res.Customer.CustomerCity)]; ok {
			res.Customer.CityLabel = cityLabel
		}
	}
	res.StartDate = startDate.Format("2006-01-02")
	res.EndDate = endDate.Format("2006-01-02")
	res.Pickup.StartDate = startDate.Format("2006-01-02 15:00")
	res.Pickup.EndDate = endDate.Format("2006-01-02 15:00")

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
				d.ID = cID
				if cityLabel, ok := getCitiesMap()[strings.TrimSpace(cID)]; ok {
					d.CityLabel = cityLabel
				}
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

	cityExpr := "city_id::text"

	itQuery := fmt.Sprintf(`SELECT fleet_itinerary_id, day_num, %s as city_id, location FROM fleet_order_itinerary WHERE order_id = %s AND organization_id = %s ORDER BY day_num`, cityExpr, r.getPlaceholder(1), r.getPlaceholder(2))
	iRows, itErr := r.db.Query(itQuery, orderID, orgID)
	if itErr == nil {
		defer iRows.Close()
		items := make([]model.FleetOrderItineraryItem, 0)
		for iRows.Next() {
			var it model.FleetOrderItineraryItem
			if err := iRows.Scan(&it.FleetItineraryID, &it.Day, &it.CityID, &it.Destination); err == nil {
				if cityLabel, ok := getCitiesMap()[strings.TrimSpace(it.CityID)]; ok {
					it.CityLabel = cityLabel
				}
				items = append(items, it)
			}
		}
		if len(items) > 0 {
			res.Itinerary = items
		}
	}

	// Addons
	addonQuery := fmt.Sprintf(`
        SELECT fa.uuid as addon_id, fa.addon_name, fa.addon_desc, foa.addon_price, foa.order_item_id
        FROM fleet_order_addons foa 
        JOIN fleet_addon fa ON foa.addon_id = fa.uuid 
		INNER JOIN fleet_order_items fai ON fai.order_item_id = foa.order_item_id
        WHERE foa.order_id = %s
    `, r.getPlaceholder(1))
	aRows, err := r.db.Query(addonQuery, orderID)
	if err == nil {
		defer aRows.Close()
		for aRows.Next() {
			var a model.OrderDetailAddon
			if err := aRows.Scan(&a.AddonID, &a.AddonName, &a.AddonDesc, &a.AddonPrice, &a.OrderItemID); err == nil {
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

	groupConcat := "STRING_AGG(CAST(city_id AS VARCHAR), ',')"

	query := fmt.Sprintf(`
        SELECT f.uuid, f.fleet_name, f.fleet_type, ft.label as fleet_type_label, f.capacity, COALESCE(f.production_year, 0) as production_year, f.engine, f.body, f.description, COALESCE(f.thumbnail, '') as thumbnail, f.created_at,
        (SELECT MIN(price) FROM fleet_prices WHERE fleet_id = f.uuid) as price,
        (SELECT uom FROM fleet_prices WHERE fleet_id = f.uuid ORDER BY price ASC LIMIT 1) as uom,
        (SELECT %s FROM fleet_pickup WHERE fleet_id = f.uuid) as cities,
		STRING_AGG(DISTINCT fu.capacity::text, ', ') AS capacities
        FROM fleets f
		INNER JOIN fleet_types ft ON ft.id = f.fleet_type
		LEFT JOIN fleet_units fu ON fu.fleet_id::text = f.uuid::text
        WHERE f.active = true AND f.is_public = 1
		GROUP BY f.uuid, f.fleet_name, f.fleet_type, ft.label, f.capacity, f.production_year, f.engine, f.body, f.description, f.thumbnail, f.created_at
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
		var fleetTypeLabel sql.NullString
		var capacities sql.NullString
		if err := rows.Scan(
			&it.FleetID, &it.FleetName, &it.FleetType, &it.FleetTypeLabel, &it.Capacity, &it.ProductionYear, &it.Engine, &it.Body, &it.Description, &it.Thumbnail, &it.CreatedAt,
			&price, &uom, &cities, &capacities,
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
		if fleetTypeLabel.Valid {
			it.FleetTypeLabel = fleetTypeLabel.String
		}
		if capacities.Valid {
			it.Capacities = capacities.String
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
			f.updated_by,
			COALESCE(STRING_AGG(DISTINCT fu.engine::text, ', '), f.engine::text) AS engines,
			COALESCE(STRING_AGG(DISTINCT fu.capacity::text, ', '), f.capacity::text) AS capacities
        FROM fleets f
		LEFT JOIN fleet_types ft ON f.fleet_type = ft.id
		LEFT JOIN users u ON u.user_id = f.created_by
		LEFT JOIN fleet_units fu ON fu.fleet_id::text = f.uuid::text
        WHERE f.uuid = %s
    `, fleetTypeExpr, createdByExpr, r.getPlaceholder(1))

	groupBy := ` GROUP BY f.uuid, f.fleet_type, ft.label, f.fleet_name, f.capacity, f.production_year, f.engine, f.body, f.fuel_type, f.transmission, f.description, f.thumbnail, f.active, f.status, f.created_at, u.fullname, f.created_by, f.updated_at, f.updated_by`

	args := []interface{}{fleetID}
	if orgID != "" {
		orgExpr := "f.organization_id::text = " + r.getPlaceholder(2)
		query += " AND " + orgExpr + groupBy
		args = append(args, orgID)
	} else {
		query += groupBy
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
	var engines sql.NullString
	var capacities sql.NullString
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
		&engines,
		&capacities,
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
		if label, ok := getFuelTypeLabelMap()[strings.TrimSpace(meta.FuelType)]; ok {
			meta.FuelTypeLabel = label
		}
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
	if engines.Valid {
		meta.Engines = engines.String
	}
	if capacities.Valid {
		meta.Capacities = capacities.String
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

func (r *FleetRepository) GetFleetPricesByIDs(priceIDs []string) (map[string]float64, error) {
	res := make(map[string]float64)
	if len(priceIDs) == 0 {
		return res, nil
	}
	unique := make(map[string]struct{}, len(priceIDs))
	ids := make([]string, 0, len(priceIDs))
	for _, id := range priceIDs {
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
	query := fmt.Sprintf("SELECT uuid, price FROM fleet_prices WHERE uuid IN (%s)", strings.Join(placeholders, ","))
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
	return res, nil
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

type FleetAvailibilityItem struct {
	FleetID        string `json:"fleet_id"`
	FleetName      string `json:"fleet_name"`
	TotalUnit      int    `json:"total_unit"`
	TotalAvailable int    `json:"total_available"`
}

func (r *FleetRepository) GetFleetAvailibility(orgID string, reqStart time.Time, reqEnd time.Time, fleetID string) ([]FleetAvailibilityItem, error) {
	if orgID == "" {
		return nil, fmt.Errorf("missing organization_id")
	}

	args := []interface{}{orgID, reqStart, reqEnd}
	isPostgres := r.driver == "postgres" || r.driver == "pgx"
	orgExpr := "f.organization_id = " + r.getPlaceholder(1)
	bookOrgExpr := "sf.organization_id = " + r.getPlaceholder(1)
	if isPostgres {
		orgExpr = "f.organization_id::text = " + r.getPlaceholder(1)
		bookOrgExpr = "sf.organization_id::text = " + r.getPlaceholder(1)
	}

	fleetFilterExpr := ""
	if fleetID != "" {
		fleetFilterExpr = " AND f.uuid = " + r.getPlaceholder(4)
		if isPostgres {
			fleetFilterExpr = " AND f.uuid::text = " + r.getPlaceholder(4)
		}
		args = append(args, fleetID)
	}

	var conflictExpr string
	if isPostgres {
		conflictExpr = fmt.Sprintf("%s::date <= fo.end_date::date AND %s::date >= fo.start_date::date", r.getPlaceholder(2), r.getPlaceholder(3))
	} else {
		conflictExpr = fmt.Sprintf("DATE(%s) <= DATE(fo.end_date) AND DATE(%s) >= DATE(fo.start_date)", r.getPlaceholder(2), r.getPlaceholder(3))
	}

	fleetJoinExpr := "b.fleet_id = f.uuid"
	totalUnitExpr := "COALESCE((SELECT COUNT(*) FROM fleet_units fu WHERE fu.fleet_id = f.uuid AND fu.status = 1), 0)"
	if isPostgres {
		fleetJoinExpr = "b.fleet_id::text = f.uuid::text"
		totalUnitExpr = "COALESCE((SELECT COUNT(*) FROM fleet_units fu WHERE fu.fleet_id::text = f.uuid::text AND fu.status = 1), 0)"
	}

	greatestFn := "GREATEST"

	query := fmt.Sprintf(`
		SELECT
			f.uuid AS fleet_id,
			f.fleet_name,
			%s AS total_unit,
			%s((%s - COALESCE(b.booked_unit, 0)), 0) AS total_available
		FROM fleets f
		LEFT JOIN (
			SELECT
				sf.fleet_id,
				COUNT(DISTINCT sf.unit_id) AS booked_unit
			FROM schedule_fleets sf
			INNER JOIN schedules s ON s.schedule_id = sf.schedule_id
			INNER JOIN fleet_orders fo ON fo.order_id = s.order_id
			WHERE %s
			  AND sf.status = 1
			  AND s.order_type = 1
			  AND s.status = 1
			  AND %s
			GROUP BY sf.fleet_id
		) b ON %s
		WHERE f.status > 0
		  AND %s
		  %s
		ORDER BY f.fleet_name ASC
	`, totalUnitExpr, greatestFn, totalUnitExpr, bookOrgExpr, conflictExpr, fleetJoinExpr, orgExpr, fleetFilterExpr)

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]FleetAvailibilityItem, 0)
	for rows.Next() {
		var it FleetAvailibilityItem
		var totalUnit int64
		var totalAvail int64
		if err := rows.Scan(&it.FleetID, &it.FleetName, &totalUnit, &totalAvail); err != nil {
			return nil, err
		}
		it.TotalUnit = int(totalUnit)
		it.TotalAvailable = int(totalAvail)
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FleetRepository) getPlaceholder(pos int) string {
	if r.driver == "postgres" || r.driver == "pgx" {
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

func (r *FleetRepository) SetFleetActiveStatus(orgID, userID, fleetID string, active bool) error {
	query := setFleetActiveMySQL
	if r.driver == "postgres" || r.driver == "pgx" {
		query = setFleetActivePostgres
	}
	res, err := database.Exec(r.db, query, active, time.Now(), userID, fleetID, orgID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
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

func (r *FleetRepository) GetScheduleByOrderID(orderID string) (*model.ModuleScheduleInfo, error) {
	query := fmt.Sprintf("SELECT schedule_id, order_id, departure_time, arrival_time, status, created_at, created_by FROM schedules WHERE order_type = 1 AND order_id = %s AND status = 1 LIMIT 1", r.getPlaceholder(1))
	var schedule model.ModuleScheduleInfo
	var DepartureTime sql.NullTime
	var ArrivalTime sql.NullTime
	if err := database.QueryRow(r.db, query, orderID).Scan(&schedule.ScheduleID, &schedule.OrderID, &DepartureTime, &ArrivalTime, &schedule.Status, &schedule.CreatedAt, &schedule.CreatedBy); err != nil {
		return nil, err
	}
	if DepartureTime.Valid {
		schedule.DepartureTime = DepartureTime.Time
	}
	if ArrivalTime.Valid {
		schedule.ArrivalTime = ArrivalTime.Time
	}
	return &schedule, nil
}

type OrderAvailabilityRepoResult struct {
	Available    bool
	ServiceTypes []int
	MinimalDay   int
	Prices       []model.OrderAvailabilityPriceItem
}

func (r *FleetRepository) GetOrderAvailability(orgID, fleetID string, cityID int, startDate time.Time, endDate *time.Time, daysCount int, serviceType *int) (*OrderAvailabilityRepoResult, error) {
	result := &OrderAvailabilityRepoResult{
		Available: true,
	}

	// 1. Get service types (use provided service_type if available, otherwise get from preference_city_types)
	if serviceType != nil {
		result.ServiceTypes = []int{*serviceType}
	} else {
		serviceTypesQuery := fmt.Sprintf(`
			SELECT service_type
			FROM preference_city_types
			WHERE city_id = %s AND organization_id = %s
		`, r.getPlaceholder(1), r.getPlaceholder(2))
		serviceTypesArgs := []interface{}{cityID, orgID}
		serviceTypesRows, err := database.Query(r.db, serviceTypesQuery, serviceTypesArgs...)
		if err != nil {
			return nil, err
		}
		defer serviceTypesRows.Close()
		for serviceTypesRows.Next() {
			var t int
			if err := serviceTypesRows.Scan(&t); err == nil {
				result.ServiceTypes = append(result.ServiceTypes, t)
			}
		}
	}

	// 2. Get minimal_day from preference_cities
	minDayQuery := fmt.Sprintf(`
		SELECT minimal_day
		FROM preference_cities
		WHERE city_id = %s AND organization_id = %s
		LIMIT 1
	`, r.getPlaceholder(1), r.getPlaceholder(2))
	minDayArgs := []interface{}{cityID, orgID}
	var minDay sql.NullInt64
	_ = database.QueryRow(r.db, minDayQuery, minDayArgs...).Scan(&minDay)
	if minDay.Valid {
		result.MinimalDay = int(minDay.Int64)
	}

	// 3. Check availability and get next schedule if endDate not provided
	if endDate == nil {
		// Find next departure date
		nextScheduleQuery := fmt.Sprintf(`
			SELECT s.departure_time
			FROM schedules s
			INNER JOIN schedule_fleets sf ON s.schedule_id = sf.schedule_id
			WHERE sf.fleet_id::text = %s
			  AND s.organization_id::text = %s
			  AND s.departure_time > %s
			  AND s.status = 1
			ORDER BY s.departure_time ASC
			LIMIT 1
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
		nextScheduleArgs := []interface{}{fleetID, orgID, startDate}
		var nextDeparture sql.NullTime
		_ = database.QueryRow(r.db, nextScheduleQuery, nextScheduleArgs...).Scan(&nextDeparture)
		if nextDeparture.Valid {
			// Availability is until day before next departure
			availableDays := int(nextDeparture.Time.Sub(startDate).Hours() / 24)
			if availableDays < daysCount {
				result.Available = false
			}
		}
	} else {
		// Use existing availability check
		availItems, err := r.GetFleetAvailibility(orgID, startDate, *endDate, fleetID)
		if err != nil {
			return nil, err
		}
		result.Available = false
		for _, item := range availItems {
			if item.TotalAvailable > 0 {
				result.Available = true
				break
			}
		}
	}

	// 4. Get prices
	if len(result.ServiceTypes) > 0 {
		// Build IN clause for service types
		placeholders := make([]string, len(result.ServiceTypes))
		args := []interface{}{fleetID}
		for i, st := range result.ServiceTypes {
			placeholders[i] = r.getPlaceholder(i + 2)
			args = append(args, st)
		}

		priceQuery := fmt.Sprintf(`
			SELECT uuid as price_id, duration, price, rent_type
			FROM fleet_prices
			WHERE fleet_id::text = %s
			  AND rent_type IN (%s)
		`, r.getPlaceholder(1), strings.Join(placeholders, ","))

		rows, err := database.Query(r.db, priceQuery, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var item model.OrderAvailabilityPriceItem
			if err := rows.Scan(&item.PriceID, &item.Duration, &item.Price, &item.RentType); err == nil {
				// Apply duration logic: duration >= daysCount, for service_type 3: duration >= daysCount - 1
				include := false
				if item.RentType == 3 {
					if item.Duration >= daysCount-1 {
						include = true
					}
				} else {
					if item.Duration >= daysCount {
						include = true
					}
				}
				if include {
					result.Prices = append(result.Prices, item)
				}
			}
		}
	}

	return result, nil
}

func (r *FleetRepository) ProcessFleetOrder(orgID, userID, orderID string, processTypeId int) error {
	query := "UPDATE fleet_orders SET status = $1, updated_at = $2, updated_by = $3 WHERE order_id = $4 AND organization_id = $5"
	res, err := database.Exec(r.db, query, processTypeId, time.Now(), userID, orderID, orgID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *FleetRepository) GetCustomerOrderMeta(orderID, organizationID string) (string, int, error) {
	query := fmt.Sprintf(`
		SELECT customer_id, order_type
		FROM customer_orders
		WHERE order_id = %s AND organization_id = %s
		LIMIT 1
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	var customerID string
	var orderType int
	if err := database.QueryRow(r.db, query, orderID, organizationID).Scan(&customerID, &orderType); err != nil {
		return "", 0, err
	}
	return customerID, orderType, nil
}

func (r *FleetRepository) InsertOrderReview(reviewID, orderID string, star int, review string, organizationID string, customerID string, orderType int, createdAt time.Time) error {
	query := fmt.Sprintf(`
		INSERT INTO order_reviews (
			review_id, star, review, organization_id, customer_id, order_type, order_id, created_at
		) VALUES (
			%s, %s, %s, %s, %s, %s, %s, %s
		)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8))

	_, err := database.Exec(r.db, query, reviewID, star, review, organizationID, customerID, orderType, orderID, createdAt)
	return err
}

func (r *FleetRepository) GetOrderReviews(orderID, organizationID string) ([]model.OrderReviewItem, error) {
	query := fmt.Sprintf(`
		SELECT r.star, r.review, c.customer_name, r.created_at
		FROM order_reviews r
		INNER JOIN fleet_orders fo ON r.order_id = fo.order_id
		INNER JOIN customers c ON c.customer_id = r.customer_id
		WHERE fo.order_id = %s AND r.organization_id = %s
		ORDER BY r.created_at DESC
		LIMIT 20
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	rows, err := database.Query(r.db, query, orderID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.OrderReviewItem, 0)
	for rows.Next() {
		var it model.OrderReviewItem
		var createdAt time.Time
		if err := rows.Scan(&it.Star, &it.Review, &it.CustomerName, &createdAt); err != nil {
			return nil, err
		}
		it.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FleetRepository) GetOrderRatingSummary(orderID, organizationID string) (*model.OrderRatingSummary, error) {
	query := fmt.Sprintf(`
		SELECT ROUND(AVG(r.star),1) AS rating, COUNT(r.review_id) AS total_ulasan
		FROM order_reviews r
		INNER JOIN fleet_orders fo ON r.order_id = fo.order_id
		WHERE fo.order_id = %s AND r.organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	var ratingAny interface{}
	var total int64
	if err := database.QueryRow(r.db, query, orderID, organizationID).Scan(&ratingAny, &total); err != nil {
		return nil, err
	}

	rating := 0.0
	switch v := ratingAny.(type) {
	case nil:
		rating = 0
	case float64:
		rating = v
	case int64:
		rating = float64(v)
	case []byte:
		if f, err := strconv.ParseFloat(string(v), 64); err == nil {
			rating = f
		}
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			rating = f
		}
	}

	return &model.OrderRatingSummary{
		Rating:      rating,
		TotalUlasan: total,
	}, nil
}

func (r *FleetRepository) GetFleetRatings(organizationID string, fleetIDs []string) (map[string]model.FleetRatingSummary, error) {
	if len(fleetIDs) == 0 {
		return map[string]model.FleetRatingSummary{}, nil
	}

	placeholders := make([]string, 0, len(fleetIDs))
	args := make([]interface{}, 0, len(fleetIDs)+1)
	for i, id := range fleetIDs {
		placeholders = append(placeholders, r.getPlaceholder(i+1))
		args = append(args, id)
	}
	args = append(args, organizationID)

	query := fmt.Sprintf(`
		SELECT
			COALESCE(ROUND(AVG(r.star), 1), 0) AS rating,
			COUNT(r.review_id) AS total_ulasan,
			fo.fleet_id
		FROM order_reviews r
		INNER JOIN fleet_orders fo ON r.order_id = fo.order_id
		WHERE fo.fleet_id IN (%s) AND fo.organization_id = %s
		GROUP BY fo.fleet_id
	`, strings.Join(placeholders, ","), r.getPlaceholder(len(fleetIDs)+1))

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]model.FleetRatingSummary, 0)
	for rows.Next() {
		var ratingAny interface{}
		var total int64
		var fleetID string
		if err := rows.Scan(&ratingAny, &total, &fleetID); err != nil {
			return nil, err
		}

		rating := 0.0
		switch v := ratingAny.(type) {
		case nil:
			rating = 0
		case float64:
			rating = v
		case int64:
			rating = float64(v)
		case []byte:
			if f, err := strconv.ParseFloat(string(v), 64); err == nil {
				rating = f
			}
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				rating = f
			}
		}

		out[fleetID] = model.FleetRatingSummary{
			FleetID:     fleetID,
			Rating:      rating,
			TotalUlasan: total,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FleetRepository) GetFleetReviews(fleetID, organizationID string) ([]model.OrderReviewItem, error) {
	query := fmt.Sprintf(`
		SELECT r.star, r.review, r.order_id, c.customer_name, r.created_at
		FROM order_reviews r
		INNER JOIN fleet_orders fo ON r.order_id = fo.order_id
		INNER JOIN customers c ON c.customer_id = r.customer_id
		WHERE fo.fleet_id = %s AND r.organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	rows, err := database.Query(r.db, query, fleetID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.OrderReviewItem, 0)
	for rows.Next() {
		var it model.OrderReviewItem
		var createdAt time.Time
		if err := rows.Scan(&it.Star, &it.Review, &it.OrderID, &it.CustomerName, &createdAt); err != nil {
			return nil, err
		}
		it.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FleetRepository) GetFleetRevenue(orgID, fleetID, startDate, endDate string) (*model.FleetRevenue, error) {
	startAt, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDate), time.Local)
	if err != nil {
		return nil, err
	}
	endAt, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDate), time.Local)
	if err != nil {
		return nil, err
	}
	endExclusive := endAt.AddDate(0, 0, 1)

	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(po.payment_amount), 0) AS revenue,
			COALESCE(COUNT(DISTINCT fo.order_id), 0) AS total_booking
		FROM fleet_orders fo
		INNER JOIN payment_orders po ON po.order_id = fo.order_id
		WHERE fo.fleet_id = %s
		  AND fo.organization_id = %s
		  AND fo.status = 1
		  AND fo.payment_status NOT IN (0,2)
		  AND po.created_at >= %s AND po.created_at < %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))
	var revenueAny interface{}
	var totalBooking int64
	if err := database.QueryRow(r.db, query, fleetID, orgID, startAt, endExclusive).Scan(&revenueAny, &totalBooking); err != nil {
		if err == sql.ErrNoRows {
			return &model.FleetRevenue{
				StartDate:    startDate,
				EndDate:      endDate,
				TotalRevenue: 0,
				TotalBooking: 0,
			}, nil
		}
		return nil, err
	}
	switch v := revenueAny.(type) {
	case nil:
		return &model.FleetRevenue{
			StartDate:    startDate,
			EndDate:      endDate,
			TotalRevenue: 0,
			TotalBooking: totalBooking,
		}, nil
	case float64:
		return &model.FleetRevenue{
			StartDate:    startDate,
			EndDate:      endDate,
			TotalRevenue: v,
			TotalBooking: totalBooking,
		}, nil
	case int64:
		return &model.FleetRevenue{
			StartDate:    startDate,
			EndDate:      endDate,
			TotalRevenue: float64(v),
			TotalBooking: totalBooking,
		}, nil
	case []byte:
		if f, err := strconv.ParseFloat(string(v), 64); err == nil {
			return &model.FleetRevenue{
				StartDate:    startDate,
				EndDate:      endDate,
				TotalRevenue: f,
				TotalBooking: totalBooking,
			}, nil
		}
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return &model.FleetRevenue{
				StartDate:    startDate,
				EndDate:      endDate,
				TotalRevenue: f,
				TotalBooking: totalBooking,
			}, nil
		}
	}
	return &model.FleetRevenue{
		StartDate:    startDate,
		EndDate:      endDate,
		TotalRevenue: 0,
		TotalBooking: totalBooking,
	}, nil
}

func (r *FleetRepository) GetPaidAmount(orderID, orgID string) (float64, error) {
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(t.amount), 0) AS paid_amount
		FROM transactions t
		INNER JOIN fleet_orders fo ON fo.order_id = t.reference_id
		WHERE fo.organization_id = %s
		  AND fo.status = 1
		  AND t.reference_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))
	var paidAmount float64
	if err := database.QueryRow(r.db, query, orgID, orderID).Scan(&paidAmount); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return paidAmount, nil
}

func (r *FleetRepository) FleetOrderCancelation(userID, orderID, orgID string) error {
	query := fmt.Sprintf(`
		UPDATE fleet_orders
		SET status = 0, updated_at = now(), updated_by = %s
		WHERE order_id = %s
		  AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	if _, err := database.Exec(r.db, query, userID, orderID, orgID); err != nil {
		return err
	}
	return nil
}

func (r *FleetRepository) RefundOrderTransactions(orderID string, refundAmount float64, reason string, paymentMethod string, bankCode string, bankAccount string, bankAccountName string, orgID string, userID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	transactionID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	invoiceNumber, err := utils.GenerateInvoiceNumberTx(tx, r.driver, orgID, 1, time.Now())
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`INSERT INTO transactions 
	(
		transaction_id,
		transaction_type,
		order_type,
		transaction_category,
		transaction_item,
		invoice_number,
		description,
		transaction_date,
		payment_type,
		amount,
		organization_id,
		payment_method,
		created_at,
		created_by,
		reference_id,
		status
	)
	VALUES (%s, 2, 1, 'TRX01', 'TRX-I14', %s, %s, %s, 1004, %s, %s, %s, %s, %s, %s, 1)`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10))
	_, err = database.TxExec(tx, query,
		transactionID.String(),
		invoiceNumber,
		"Refund - Order ID "+orderID,
		time.Now(),
		refundAmount,
		orgID,
		paymentMethod,
		time.Now(),
		userID,
		orderID,
	)
	if err != nil {
		fmt.Println("INSERT INTO transactions err:", err)
		return err
	}

	refundID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	refundQuery := fmt.Sprintf(`INSERT INTO transaction_refund
	(
		refund_id,
		transaction_id,
		reference_id,
		description,
		amount,
		bank_code,
		bank_account,
		bank_account_name,
		organization_id,
		created_at,
		created_by
	)
	VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11))
	_, err = database.TxExec(tx, refundQuery,
		refundID.String(),
		transactionID.String(),
		orderID,
		reason,
		refundAmount,
		bankCode,
		bankAccount,
		bankAccountName,
		orgID,
		time.Now(),
		userID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *FleetRepository) CancelSchedulesAndRelated(userID, orderID, orgID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get all schedule IDs first
	var scheduleIDs []string
	orgExpr := "organization_id::text = " + r.getPlaceholder(2)
	getScheduleQuery := fmt.Sprintf(`
		SELECT schedule_id
		FROM schedules
		WHERE order_id = %s AND %s
	`, r.getPlaceholder(1), orgExpr)

	rows, err := database.TxQuery(tx, getScheduleQuery, orderID, orgID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		scheduleIDs = append(scheduleIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Update schedules table
	updateSchedulesQuery := fmt.Sprintf(`
		UPDATE schedules
		SET status = 0, updated_at = now(), updated_by = %s
		WHERE order_id = %s AND %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), orgExpr)
	_, err = database.TxExec(tx, updateSchedulesQuery, userID, orderID)
	if err != nil {
		return err
	}

	// Update schedule_fleets table
	updateScheduleFleetsQuery := fmt.Sprintf(`
		UPDATE schedule_fleets
		SET status = 0, updated_at = now(), updated_by = %s
		WHERE order_id = %s AND %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), orgExpr)
	_, err = database.TxExec(tx, updateScheduleFleetsQuery, userID, orderID)
	if err != nil {
		return err
	}

	// Update schedule_fleet_teams table for each schedule_id
	if len(scheduleIDs) > 0 {
		for _, sid := range scheduleIDs {
			updateScheduleTeamsQuery := fmt.Sprintf(`
				UPDATE schedule_fleet_teams
				SET status = 0, updated_at = now(), updated_by = %s
				WHERE schedule_id = %s
			`, r.getPlaceholder(1), r.getPlaceholder(2))
			_, err = database.TxExec(tx, updateScheduleTeamsQuery, userID, sid)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *FleetRepository) GetRefundOrderDetail(orderID string, orgID string) (*model.FleetOrderCancelRequest, error) {
	query := fmt.Sprintf(`
		WHERE order_id = %s AND %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))
	rows, err := database.TxQuery(r.db, query, orderID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var req model.FleetOrderCancelRequest
	if err := rows.Scan(&req); err != nil {
		return nil, err
	}
	return nil, nil
}
