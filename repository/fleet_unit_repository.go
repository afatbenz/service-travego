package repository

import (
	"database/sql"
	"fmt"
	"service-travego/database"
	"service-travego/model"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FleetUnitRepository struct {
	db     *sql.DB
	driver string
}

func NewFleetUnitRepository(db *sql.DB, driver string) *FleetUnitRepository {
	return &FleetUnitRepository{db: db, driver: driver}
}

func (r *FleetUnitRepository) placeholder(pos int) string {
	if r.driver == "postgres" || r.driver == "pgx" {
		return "$" + strconv.Itoa(pos)
	}
	return "?"
}

func (r *FleetUnitRepository) FindExistingVehicleIDs(orgID string, vehicleIDs []string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	if len(vehicleIDs) == 0 {
		return out, nil
	}

	in := make([]string, 0, len(vehicleIDs))
	args := make([]interface{}, 0, 1+len(vehicleIDs))
	args = append(args, orgID)
	for i, v := range vehicleIDs {
		in = append(in, r.placeholder(i+2))
		args = append(args, strings.ToUpper(strings.TrimSpace(v)))
	}

	orgExpr := "organization_id = " + r.placeholder(1)
	vehicleExpr := "UPPER(COALESCE(vehicle_id, ''))"
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.placeholder(1)
		vehicleExpr = "UPPER(COALESCE(vehicle_id::text, ''))"
	}

	query := "SELECT DISTINCT " + vehicleExpr + " AS vehicle_id FROM fleet_units WHERE " + orgExpr + " AND " + vehicleExpr + " IN (" + strings.Join(in, ",") + ")"
	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var v sql.NullString
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		if v.Valid && v.String != "" {
			out[v.String] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FleetUnitRepository) FindExistingPlateNumbers(orgID string, plateNumbers []string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	if len(plateNumbers) == 0 {
		return out, nil
	}

	in := make([]string, 0, len(plateNumbers))
	args := make([]interface{}, 0, 1+len(plateNumbers))
	args = append(args, orgID)
	for i, v := range plateNumbers {
		in = append(in, r.placeholder(i+2))
		args = append(args, strings.ToUpper(strings.TrimSpace(v)))
	}

	orgExpr := "organization_id = " + r.placeholder(1)
	plateExpr := "UPPER(COALESCE(plate_number, ''))"
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.placeholder(1)
	}

	query := "SELECT DISTINCT " + plateExpr + " AS plate_number FROM fleet_units WHERE " + orgExpr + " AND " + plateExpr + " IN (" + strings.Join(in, ",") + ")"
	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var v sql.NullString
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		if v.Valid && v.String != "" {
			out[v.String] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

const listFleetUnitsPostgres = `
SELECT
	fu.unit_id,
	COALESCE(fu.vehicle_id::text, '') AS vehicle_id,
	fu.plate_number,
	COALESCE(fu.fleet_id::text, '') AS fleet_id,
	COALESCE(f.fleet_name, '') AS fleet_name,
	COALESCE(fu.engine, '') AS engine,
	COALESCE(fu.transmission, '') AS transmission,
	fu.capacity,
	fu.production_year,
	COALESCE(fu.created_by::text, '') AS created_by,
	fu.created_at,
	COALESCE(fu.status, 0) AS status,
	COALESCE(fu.ownership_type, 0) AS ownership_type
FROM fleet_units fu
LEFT JOIN fleets f ON f.uuid::text = fu.fleet_id::text
LEFT JOIN schedule_fleets sf ON sf.fleet_id::text = fu.fleet_id::text
AND sf.order_id::text = $3
WHERE fu.organization_id::text = $1 AND (fu.fleet_id::text = $2 OR $2 = '')
AND (
	$4 = '' OR
	COALESCE(fu.vehicle_id::text, '') ILIKE $4 OR
	COALESCE(fu.plate_number, '') ILIKE $4 OR
	COALESCE(f.fleet_name, '') ILIKE $4 OR
	COALESCE(fu.engine, '') ILIKE $4
)
ORDER BY fu.created_at DESC
`

const listFleetUnitsPostgresByOrderID = `
SELECT
	fu.unit_id,
	COALESCE(fu.vehicle_id::text, '') AS vehicle_id,
	fu.plate_number,
	COALESCE(fu.fleet_id::text, '') AS fleet_id,
	COALESCE(f.fleet_name, '') AS fleet_name,
	COALESCE(fu.engine, '') AS engine,
	COALESCE(fu.transmission, '') AS transmission,
	fu.capacity,
	fu.production_year,
	COALESCE(fu.created_by::text, '') AS created_by,
	fu.created_at,
	COALESCE(fu.status, 0) AS status,
	COALESCE(fu.ownership_type, 0) AS ownership_type
FROM fleet_units fu
LEFT JOIN fleets f ON f.uuid::text = fu.fleet_id::text
INNER JOIN schedule_fleets sf ON sf.fleet_id::text = fu.fleet_id::text
WHERE fu.organization_id::text = $1 AND (fu.fleet_id::text = $2 OR $2 = '') AND sf.order_id::text = $3
AND (
	$4 = '' OR
	COALESCE(fu.vehicle_id::text, '') ILIKE $4 OR
	COALESCE(fu.plate_number, '') ILIKE $4 OR
	COALESCE(f.fleet_name, '') ILIKE $4 OR
	COALESCE(fu.engine, '') ILIKE $4
)
ORDER BY fu.created_at DESC
`

const createFleetUnitPostgres = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_at, ownership_type)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
`

const createFleetUnitPostgresAndStatus = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_at, ownership_type)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

const createFleetUnitPostgresWithUUID = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_at, ownership_type)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

const createFleetUnitPostgresWithUUIDAndStatus = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_at, ownership_type)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
`

const createFleetUnitMySQL = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_at, ownership_type)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitMySQLAndStatus = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_at, ownership_type)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitMySQLWithUUIDAndStatus = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_at, ownership_type)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitMySQLWithUUID = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_at, ownership_type)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitPostgresCreatedDate = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_date, ownership_type)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
`

const createFleetUnitPostgresCreatedDateAndStatus = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_date, ownership_type)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

const createFleetUnitPostgresWithUUIDCreatedDateAndStatus = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_date, ownership_type)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
`

const createFleetUnitPostgresWithUUIDCreatedDate = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_date, ownership_type)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

const createFleetUnitMySQLCreatedDate = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_date, ownership_type)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitMySQLCreatedDateAndStatus = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_date, ownership_type)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitMySQLWithUUIDCreatedDateAndStatus = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_date, ownership_type)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitMySQLWithUUIDCreatedDate = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_date, ownership_type)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const updateFleetUnitPostgres = `
UPDATE fleet_units
SET vehicle_id = $1,
	plate_number = $2,
	fleet_id = $3,
	engine = $4,
	transmission = $5,
	capacity = $6,
	production_year = $7,
	updated_by = $8,
	updated_at = $9,
	ownership_type = $10
WHERE unit_id = $11 AND organization_id = $12
`

const updateFleetUnitMySQL = `
UPDATE fleet_units
SET vehicle_id = ?,
	plate_number = ?,
	fleet_id = ?,
	engine = ?,
	transmission = ?,
	capacity = ?,
	production_year = ?,
	updated_by = ?,
	updated_at = ?,
	ownership_type = ?
WHERE unit_id = ? AND organization_id = ?
`

const detailFleetUnitPostgres = `
SELECT
	fu.unit_id,
	COALESCE(fu.vehicle_id::text, '') AS vehicle_id,
	fu.plate_number,
	COALESCE(fu.fleet_id::text, '') AS fleet_id,
	COALESCE(f.fleet_name, '') AS fleet_name,
	COALESCE(ft.label, '') AS fleet_type,
	COALESCE(fu.engine, '') AS engine,
	COALESCE(fu.transmission, '') AS transmission,
	fu.capacity,
	fu.production_year,
	COALESCE(fu.status, 0) AS status,
	COALESCE(f.description, '') AS description,
	COALESCE(f.thumbnail, '') AS thumbnail,
	COALESCE(uc.fullname, uc.username, '') AS created_by,
	fu.created_at,
	COALESCE(uu.fullname, uu.username, '') AS updated_by,
	fu.updated_at,
	fu.ownership_type
FROM fleet_units fu
LEFT JOIN fleets f ON f.uuid::text = fu.fleet_id::text
LEFT JOIN fleet_types ft ON f.fleet_type = ft.id
LEFT JOIN users uc ON fu.created_by = uc.user_id
LEFT JOIN users uu ON fu.updated_by = uu.user_id
WHERE fu.unit_id = $1 AND fu.organization_id::text = $2
`

const detailFleetUnitMySQL = `
SELECT
	fu.unit_id,
	COALESCE(fu.vehicle_id, '') AS vehicle_id,
	fu.plate_number,
	fu.fleet_id,
	COALESCE(f.fleet_name, '') AS fleet_name,
	COALESCE(ft.label, '') AS fleet_type,
	COALESCE(fu.engine, '') AS engine,
	COALESCE(fu.transmission, '') AS transmission,
	fu.capacity,
	fu.production_year,
	COALESCE(fu.status, 0) AS status,
	COALESCE(f.description, '') AS description,
	COALESCE(f.thumbnail, '') AS thumbnail,
	COALESCE(uc.fullname, uc.username, '') AS created_by,
	fu.created_at,
	COALESCE(uu.fullname, uu.username, '') AS updated_by,
	fu.updated_at,
	fu.ownership_type
FROM fleet_units fu
LEFT JOIN fleets f ON fu.fleet_id = f.uuid
LEFT JOIN fleet_types ft ON f.fleet_type = ft.id
LEFT JOIN users uc ON fu.created_by = uc.user_id
LEFT JOIN users uu ON fu.updated_by = uu.user_id
WHERE fu.unit_id = ? AND fu.organization_id = ?
`

const unitRatingPostgres = `
SELECT
	COALESCE(ROUND(AVG(r.star), 1), 0)::float8 AS rating
FROM order_reviews r
INNER JOIN fleet_orders fo ON r.order_id = fo.order_id
INNER JOIN schedule_fleets sf ON sf.order_id = r.order_id
WHERE sf.unit_id::text = $1 AND sf.organization_id::text = $2
`

const unitRatingMySQL = `
SELECT
	COALESCE(ROUND(AVG(r.star), 1), 0) AS rating
FROM order_reviews r
INNER JOIN fleet_orders fo ON r.order_id = fo.order_id
INNER JOIN schedule_fleets sf ON sf.order_id = r.order_id
WHERE sf.unit_id = ? AND sf.organization_id = ?
`

const unitReviewsPostgres = `
SELECT r.star, r.review, c.customer_name, r.created_at
FROM order_reviews r
INNER JOIN fleet_orders fo ON r.order_id = fo.order_id
INNER JOIN schedule_fleets sf ON sf.order_id = r.order_id
INNER JOIN customers c ON c.customer_id = r.customer_id
WHERE sf.unit_id::text = $1 AND sf.organization_id::text = $2
ORDER BY r.created_at DESC
LIMIT 10
`

const unitReviewsMySQL = `
SELECT r.star, r.review, c.customer_name, r.created_at
FROM order_reviews r
INNER JOIN fleet_orders fo ON r.order_id = fo.order_id
INNER JOIN schedule_fleets sf ON sf.order_id = r.order_id
INNER JOIN customers c ON c.customer_id = r.customer_id
WHERE sf.unit_id = ? AND sf.organization_id = ?
ORDER BY r.created_at DESC
LIMIT 10
`

const unitUpcomingSchedulePostgres = `
SELECT fo.start_date, fo.end_date
FROM schedule_fleets sf
INNER JOIN fleet_orders fo ON sf.order_id = fo.order_id
WHERE sf.unit_id::text = $1 AND sf.organization_id::text = $2 AND fo.start_date >= $3
ORDER BY fo.start_date ASC
LIMIT 1
`

const unitUpcomingScheduleMySQL = `
SELECT fo.start_date, fo.end_date
FROM schedule_fleets sf
INNER JOIN fleet_orders fo ON sf.order_id = fo.order_id
WHERE sf.unit_id = ? AND sf.organization_id = ? AND fo.start_date >= ?
ORDER BY fo.start_date ASC
LIMIT 1
`

func (r *FleetUnitRepository) GetFleetPickupCityIDs(orgID, fleetID string) ([]int, error) {
	orgExpr := "organization_id = " + r.placeholder(1)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.placeholder(1)
	}
	query := "SELECT city_id FROM fleet_pickup WHERE fleet_id = " + r.placeholder(2) + " AND " + orgExpr

	rows, err := database.Query(r.db, query, orgID, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]int, 0)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetUnitRepository) List(orgID, fleetId, orderID, search string) ([]model.FleetUnitListItem, error) {
	search = strings.TrimSpace(search)
	searchPattern := ""
	if search != "" {
		searchPattern = "%" + search + "%"
	}

	query := listFleetUnitsPostgres
	if strings.TrimSpace(orderID) != "" {
		query = listFleetUnitsPostgresByOrderID
	}

	args := make([]interface{}, 0, 4)
	args = append(args, orgID)
	args = append(args, fleetId)
	args = append(args, orderID)
	args = append(args, searchPattern)
	rows, err := database.Query(r.db, query, args...)
	fmt.Println(query)
	fmt.Println(args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.FleetUnitListItem, 0)
	for rows.Next() {
		var it model.FleetUnitListItem
		var createdAt time.Time
		if err := rows.Scan(
			&it.UnitID,
			&it.VehicleID,
			&it.PlateNumber,
			&it.FleetID,
			&it.FleetName,
			&it.Engine,
			&it.Transmission,
			&it.Capacity,
			&it.ProductionYear,
			&it.CreatedBy,
			&createdAt,
			&it.Status,
			&it.OwnershipType,
		); err != nil {
			return nil, err
		}
		it.CreatedDate = createdAt.Format("2006-01-02 15:04:05")
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetUnitRepository) Create(req *model.FleetUnitCreateRequest) (string, error) {
	id := uuid.New().String()
	now := time.Now()
	req.CreatedDate = now
	req.UnitID = id

	tryExec := func(query string, args ...interface{}) error {
		_, err := database.Exec(r.db, query, args...)
		return err
	}

	var err error
	if r.driver == "postgres" || r.driver == "pgx" {
		err = tryExec(createFleetUnitPostgresWithUUIDAndStatus, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
		if err != nil {
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "column") && strings.Contains(errMsg, "status") && strings.Contains(errMsg, "does not exist") {
				err = tryExec(createFleetUnitPostgresWithUUID, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
				errMsg = strings.ToLower(err.Error())
			}
			if strings.Contains(errMsg, "column") && strings.Contains(errMsg, "uuid") && strings.Contains(errMsg, "does not exist") {
				err = tryExec(createFleetUnitPostgresAndStatus, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
				if err != nil {
					errMsg2 := strings.ToLower(err.Error())
					if strings.Contains(errMsg2, "column") && strings.Contains(errMsg2, "status") && strings.Contains(errMsg2, "does not exist") {
						err = tryExec(createFleetUnitPostgres, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
					}
				}
			}
		}
		if err != nil {
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "created_at") && strings.Contains(errMsg, "does not exist") {
				err = tryExec(createFleetUnitPostgresWithUUIDCreatedDateAndStatus, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
				if err != nil {
					errMsg2 := strings.ToLower(err.Error())
					if strings.Contains(errMsg2, "column") && strings.Contains(errMsg2, "status") && strings.Contains(errMsg2, "does not exist") {
						err = tryExec(createFleetUnitPostgresWithUUIDCreatedDate, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
						errMsg2 = strings.ToLower(err.Error())
					}
					if strings.Contains(errMsg2, "column") && strings.Contains(errMsg2, "uuid") && strings.Contains(errMsg2, "does not exist") {
						err = tryExec(createFleetUnitPostgresCreatedDateAndStatus, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
						if err != nil {
							errMsg3 := strings.ToLower(err.Error())
							if strings.Contains(errMsg3, "column") && strings.Contains(errMsg3, "status") && strings.Contains(errMsg3, "does not exist") {
								err = tryExec(createFleetUnitPostgresCreatedDate, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
							}
						}
					}
				}
			}
		}
	} else {
		err = tryExec(createFleetUnitMySQLWithUUIDAndStatus, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
		if err != nil {
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "unknown column") && strings.Contains(errMsg, "status") {
				err = tryExec(createFleetUnitMySQLWithUUID, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
				errMsg = strings.ToLower(err.Error())
			}
			if strings.Contains(errMsg, "unknown column") && strings.Contains(errMsg, "uuid") {
				err = tryExec(createFleetUnitMySQLAndStatus, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
				if err != nil {
					errMsg2 := strings.ToLower(err.Error())
					if strings.Contains(errMsg2, "unknown column") && strings.Contains(errMsg2, "status") {
						err = tryExec(createFleetUnitMySQL, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
					}
				}
			}
		}
		if err != nil {
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "unknown column") && strings.Contains(errMsg, "created_at") {
				err = tryExec(createFleetUnitMySQLWithUUIDCreatedDateAndStatus, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
				if err != nil {
					errMsg2 := strings.ToLower(err.Error())
					if strings.Contains(errMsg2, "unknown column") && strings.Contains(errMsg2, "status") {
						err = tryExec(createFleetUnitMySQLWithUUIDCreatedDate, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
						errMsg2 = strings.ToLower(err.Error())
					}
					if strings.Contains(errMsg2, "unknown column") && strings.Contains(errMsg2, "uuid") {
						err = tryExec(createFleetUnitMySQLCreatedDateAndStatus, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
						if err != nil {
							errMsg3 := strings.ToLower(err.Error())
							if strings.Contains(errMsg3, "unknown column") && strings.Contains(errMsg3, "status") {
								err = tryExec(createFleetUnitMySQLCreatedDate, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate, req.OwnershipType)
							}
						}
					}
				}
			}
		}
	}

	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *FleetUnitRepository) Update(req *model.FleetUnitUpdateRequest) error {
	now := time.Now()
	req.UpdatedDate = now

	query := updateFleetUnitMySQL
	if r.driver == "postgres" || r.driver == "pgx" {
		query = updateFleetUnitPostgres
	}
	res, err := database.Exec(
		r.db,
		query,
		req.VehicleID,
		req.PlateNumber,
		req.FleetID,
		req.Engine,
		req.Transmission,
		req.Capacity,
		req.ProductionYear,
		req.UpdatedBy,
		req.UpdatedDate,
		req.OwnershipType,
		req.UnitID,
		req.OrganizationID,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *FleetUnitRepository) Detail(orgID, id string) (*model.FleetUnitDetailResponse, error) {
	query := detailFleetUnitMySQL
	if r.driver == "postgres" || r.driver == "pgx" {
		query = detailFleetUnitPostgres
	}
	var res model.FleetUnitDetailResponse
	var createdAt time.Time
	var updatedAt sql.NullTime
	var ownershipType sql.NullInt32
	err := database.QueryRow(r.db, query, id, orgID).Scan(
		&res.UnitID,
		&res.VehicleID,
		&res.PlateNumber,
		&res.FleetID,
		&res.FleetName,
		&res.FleetType,
		&res.Engine,
		&res.Transmission,
		&res.Capacity,
		&res.ProductionYear,
		&res.Status,
		&res.Description,
		&res.Thumbnail,
		&res.CreatedBy,
		&createdAt,
		&res.UpdatedBy,
		&updatedAt,
		&ownershipType,
	)
	if err != nil {
		return nil, err
	}
	res.CreatedDate = createdAt.Format("2006-01-02 15:04:05")
	if updatedAt.Valid {
		res.UpdatedDate = updatedAt.Time.Format("2006-01-02 15:04:05")
	}
	if ownershipType.Valid {
		ot := int(ownershipType.Int32)
		res.OwnershipType = &ot
	}
	return &res, nil
}

func (r *FleetUnitRepository) GetOwnershipInformation(orgID, unitID string) (*model.FleetUnitOwnershipInformation, error) {
	query := `
		SELECT op.partner_id, op.partner_name, op.partner_phone, op.partner_email
		FROM operation_partner op
		INNER JOIN fleet_unit_ownership fuo ON fuo.partner_id = op.partner_id
		INNER JOIN fleet_units fu ON fu.unit_id = fuo.unit_id
		WHERE fuo.organization_id = $1 AND fu.unit_id = $2
	`
	if r.driver == "mysql" {
		query = strings.ReplaceAll(query, "$1", "?")
		query = strings.ReplaceAll(query, "$2", "?")
	}

	var info model.FleetUnitOwnershipInformation
	var partnerEmail sql.NullString
	err := r.db.QueryRow(query, orgID, unitID).Scan(
		&info.PartnerID,
		&info.PartnerName,
		&info.PartnerPhone,
		&partnerEmail,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if partnerEmail.Valid {
		info.PartnerEmail = &partnerEmail.String
	}
	return &info, nil
}

func (r *FleetUnitRepository) SetUnitOwnership(unitID, partnerID, orgID, userID string) error {
	now := time.Now()
	fleetOwnershipID := uuid.New().String()

	query := `
		INSERT INTO fleet_unit_ownership
			(fleet_ownership_id, unit_id, partner_id, created_at, created_by, updated_at, updated_by, organization_id)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8)
	`
	if r.driver == "mysql" {
		query = strings.ReplaceAll(query, "$1", "?")
		query = strings.ReplaceAll(query, "$2", "?")
		query = strings.ReplaceAll(query, "$3", "?")
		query = strings.ReplaceAll(query, "$4", "?")
		query = strings.ReplaceAll(query, "$5", "?")
		query = strings.ReplaceAll(query, "$6", "?")
		query = strings.ReplaceAll(query, "$7", "?")
		query = strings.ReplaceAll(query, "$8", "?")
	}

	_, err := r.db.Exec(query, fleetOwnershipID, unitID, partnerID, now, userID, now, userID, orgID)
	if err != nil {
		return err
	}

	deleteQuery := `
		DELETE FROM fleet_unit_ownership
		WHERE unit_id = $1 AND partner_id != $2 AND organization_id = $3
	`
	if r.driver == "mysql" {
		deleteQuery = strings.ReplaceAll(deleteQuery, "$1", "?")
		deleteQuery = strings.ReplaceAll(deleteQuery, "$2", "?")
		deleteQuery = strings.ReplaceAll(deleteQuery, "$3", "?")
	}

	_, err = r.db.Exec(deleteQuery, unitID, partnerID, orgID)
	return err
}

func (r *FleetUnitRepository) UnitOrderHistory(orgID, unitID, startDate, endDate string) ([]model.FleetUnitOrderHistoryItem, error) {
	startAt, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDate), time.Local)
	if err != nil {
		return nil, err
	}
	endAt, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDate), time.Local)
	if err != nil {
		return nil, err
	}
	endExclusive := endAt.AddDate(0, 0, 1)

	query := `SELECT
		COALESCE(fuo.order_id::text, '') AS order_id,
		COALESCE(fuo.unit_id::text, '') AS unit_id,
		COALESCE(sft.driver_id::text, '') AS driver_id,
		COALESCE(d.fullname, '') AS driver_name,
		fo.start_date,
		fo.end_date,
		fo.status,
		COALESCE(fo.pickup_city_id::text, '') AS pickup_city_id,
		STRING_AGG(DISTINCT fi.city_id::text, ', ') AS destination_ids
	FROM schedule_fleets fuo
	INNER JOIN fleet_orders fo ON fuo.order_id::text = fo.order_id::text
	INNER JOIN schedule_fleet_teams sft ON sft.schedule_fleet_id = fuo.uuid
	INNER JOIN fleet_order_itinerary fi ON fi.order_id = fo.order_id
	LEFT JOIN employee d ON d.uuid::text = sft.driver_id::text
	WHERE fo.organization_id::text = $1 AND fuo.unit_id::text = $2 AND fo.start_date >= $3 AND fo.end_date < $4
	GROUP BY fuo.order_id, fuo.unit_id, sft.driver_id, d.fullname, fo.start_date, fo.end_date, fo.status, fo.pickup_city_id, fi.city_id
	ORDER BY fo.start_date DESC
	`
	fmt.Println(query, orgID, unitID, startAt, endExclusive)

	rows, err := database.Query(r.db, query, orgID, unitID, startAt, endExclusive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.FleetUnitOrderHistoryItem, 0)
	for rows.Next() {
		var it model.FleetUnitOrderHistoryItem
		var startDate sql.NullTime
		var endDate sql.NullTime

		if err := rows.Scan(
			&it.OrderID,
			&it.UnitID,
			&it.DriverID,
			&it.DriverName,
			&startDate,
			&endDate,
			&it.Status,
			&it.PickupCityID,
			&it.DestinationIDs,
		); err != nil {
			return nil, err
		}
		if startDate.Valid {
			it.StartDate = startDate.Time.Format("2006-01-02")
		}
		if endDate.Valid {
			it.EndDate = endDate.Time.Format("2006-01-02")
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FleetUnitRepository) UnitRating(orgID, unitID string) (float64, error) {
	query := unitRatingMySQL
	if r.driver == "postgres" || r.driver == "pgx" {
		query = unitRatingPostgres
	}
	var rating sql.NullFloat64
	if err := database.QueryRow(r.db, query, unitID, orgID).Scan(&rating); err != nil {
		return 0, err
	}
	if rating.Valid {
		return rating.Float64, nil
	}
	return 0, nil
}

func (r *FleetUnitRepository) UnitReviews(orgID, unitID string) ([]model.OrderReviewItem, error) {
	query := unitReviewsMySQL
	if r.driver == "postgres" || r.driver == "pgx" {
		query = unitReviewsPostgres
	}

	rows, err := database.Query(r.db, query, unitID, orgID)
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

func (r *FleetUnitRepository) UnitTotalSchedules(orgID, unitID string) (int64, error) {
	query := `
			SELECT COUNT(*) AS total_schedules
			FROM schedule_fleets
			WHERE unit_id::text = $1 AND organization_id::text = $2
			`
	var total int64
	if err := database.QueryRow(r.db, query, unitID, orgID).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *FleetUnitRepository) UnitLatestSchedule(orgID, unitID string, now time.Time) (*model.FleetUnitScheduleRange, error) {
	query := `SELECT fo.start_date, fo.end_date
			FROM schedule_fleets sf
			INNER JOIN fleet_orders fo ON sf.order_id = fo.order_id
			WHERE sf.unit_id::text = $1 AND sf.organization_id::text = $2 AND fo.end_date <= $3
			ORDER BY fo.end_date DESC
			LIMIT 1
			`
	var startDate sql.NullTime
	var endDate sql.NullTime
	if err := database.QueryRow(r.db, query, unitID, orgID, now).Scan(&startDate, &endDate); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	out := &model.FleetUnitScheduleRange{}
	if startDate.Valid {
		out.StartDate = startDate.Time.Format("2006-01-02")
	}
	if endDate.Valid {
		out.EndDate = endDate.Time.Format("2006-01-02")
	}
	return out, nil
}

func (r *FleetUnitRepository) UnitUpcomingSchedule(orgID, unitID string, now time.Time) (*model.FleetUnitScheduleRange, error) {
	query := unitUpcomingScheduleMySQL
	if r.driver == "postgres" || r.driver == "pgx" {
		query = unitUpcomingSchedulePostgres
	}
	var startDate sql.NullTime
	var endDate sql.NullTime
	if err := database.QueryRow(r.db, query, unitID, orgID, now).Scan(&startDate, &endDate); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	out := &model.FleetUnitScheduleRange{}
	if startDate.Valid {
		out.StartDate = startDate.Time.Format("2006-01-02")
	}
	if endDate.Valid {
		out.EndDate = endDate.Time.Format("2006-01-02")
	}
	return out, nil
}

func (r *FleetUnitRepository) GetOrderDestinationCityIDs(orderIDs []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(orderIDs) == 0 {
		return out, nil
	}

	in := make([]string, 0, len(orderIDs))
	args := make([]interface{}, 0, len(orderIDs))
	for i, id := range orderIDs {
		in = append(in, r.placeholder(i+1))
		args = append(args, strings.TrimSpace(id))
	}

	query := "SELECT COALESCE(order_id, '') AS order_id, COALESCE(CAST(city_id AS CHAR), '') AS city_id FROM fleet_order_itinerary WHERE order_id IN (" + strings.Join(in, ",") + ")"
	if r.driver == "postgres" || r.driver == "pgx" {
		query = "SELECT COALESCE(order_id::text, '') AS order_id, COALESCE(city_id::text, '') AS city_id FROM fleet_order_itinerary WHERE order_id::text IN (" + strings.Join(in, ",") + ")"
	}

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[string]map[string]struct{}{}
	for rows.Next() {
		var orderID string
		var cityID string
		if err := rows.Scan(&orderID, &cityID); err != nil {
			return nil, err
		}
		orderID = strings.TrimSpace(orderID)
		cityID = strings.TrimSpace(cityID)
		if orderID == "" || cityID == "" {
			continue
		}
		if _, ok := seen[orderID]; !ok {
			seen[orderID] = map[string]struct{}{}
		}
		if _, ok := seen[orderID][cityID]; ok {
			continue
		}
		seen[orderID][cityID] = struct{}{}
		out[orderID] = append(out[orderID], cityID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FleetUnitRepository) GetUnitRevenue(orgID, unitID, startDate, endDate string) (*model.FleetUnitRevenue, error) {
	startAt, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDate), time.Local)
	if err != nil {
		return nil, err
	}
	endAt, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDate), time.Local)
	if err != nil {
		return nil, err
	}
	endExclusive := endAt.AddDate(0, 0, 1)

	parseFloat64 := func(v interface{}) (float64, bool) {
		switch vv := v.(type) {
		case nil:
			return 0, true
		case float64:
			return vv, true
		case float32:
			return float64(vv), true
		case int64:
			return float64(vv), true
		case int32:
			return float64(vv), true
		case int:
			return float64(vv), true
		case []byte:
			f, err := strconv.ParseFloat(string(vv), 64)
			return f, err == nil
		case string:
			f, err := strconv.ParseFloat(vv, 64)
			return f, err == nil
		default:
			return 0, false
		}
	}

	parseInt64 := func(v interface{}) (int64, bool) {
		switch vv := v.(type) {
		case nil:
			return 0, true
		case int64:
			return vv, true
		case int32:
			return int64(vv), true
		case int:
			return int64(vv), true
		case float64:
			return int64(vv), true
		case float32:
			return int64(vv), true
		case []byte:
			i, err := strconv.ParseInt(string(vv), 10, 64)
			return i, err == nil
		case string:
			i, err := strconv.ParseInt(vv, 10, 64)
			return i, err == nil
		default:
			return 0, false
		}
	}

	sfOrderExpr := "order_id::text"
	trxRefExpr := "reference_id::text"
	foiOrderExpr := "order_id::text"
	unitExpr := "unit_id::text = " + r.placeholder(1)
	orgExpr := "organization_id::text = " + r.placeholder(2)
	totalBookingUnitExpr := "sf.unit_id::text = " + r.placeholder(5)
	totalBookingOrgExpr := "sf.organization_id::text = " + r.placeholder(6)
	totalBookingFoOrgExpr := "fo2.organization_id::text = " + r.placeholder(7)

	query := fmt.Sprintf(`
		SELECT
			(
				SELECT COALESCE(SUM(COALESCE(t.total_amount / NULLIF(q.total_qty, 0), 0)), 0) AS revenue
				FROM (
					SELECT DISTINCT %s AS order_id
					FROM schedule_fleets
					WHERE %s AND %s
				) sf
				INNER JOIN (
					SELECT %s AS order_id, SUM(amount) AS total_amount
					FROM transactions
					WHERE transaction_date IS NOT NULL
						AND transaction_date >= %s AND transaction_date < %s
					GROUP BY %s
				) t ON t.order_id = sf.order_id
				INNER JOIN (
					SELECT %s AS order_id, SUM(quantity) AS total_qty
					FROM fleet_order_items
					GROUP BY %s
				) q ON q.order_id = sf.order_id
			) AS revenue,
			(
				SELECT COALESCE(COUNT(DISTINCT sf.schedule_number), 0) AS total_booking
				FROM schedule_fleets sf
				INNER JOIN fleet_orders fo2 ON fo2.order_id = sf.order_id
				WHERE %s
					AND %s
					AND %s
					AND fo2.status = 1
					AND fo2.start_date >= %s
					AND fo2.end_date < %s
			) AS total_booking
	`,
		sfOrderExpr,
		unitExpr,
		orgExpr,
		trxRefExpr,
		r.placeholder(3),
		r.placeholder(4),
		trxRefExpr,
		foiOrderExpr,
		foiOrderExpr,
		totalBookingUnitExpr,
		totalBookingOrgExpr,
		totalBookingFoOrgExpr,
		r.placeholder(8),
		r.placeholder(9),
	)
	fmt.Println(query)

	var revenueAny interface{}
	var totalBookingAny interface{}
	if err := database.QueryRow(
		r.db,
		query,
		unitID,
		orgID,
		startAt,
		endExclusive,
		unitID,
		orgID,
		orgID,
		startAt,
		endExclusive,
	).Scan(&revenueAny, &totalBookingAny); err != nil {
		return nil, err
	}

	revenue, ok := parseFloat64(revenueAny)
	if !ok {
		revenue = 0
	}
	totalBooking, ok := parseInt64(totalBookingAny)
	if !ok {
		totalBooking = 0
	}

	return &model.FleetUnitRevenue{TotalRevenue: revenue, TotalBooking: totalBooking}, nil
}

func (r *FleetUnitRepository) ListUnitRevenueHistory(orgID, unitID, startDate, endDate string) ([]model.FleetUnitRevenueHistoryItem, error) {
	if r.driver != "postgres" && r.driver != "pgx" {
		return nil, fmt.Errorf("unsupported driver")
	}

	startAt, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDate), time.Local)
	if err != nil {
		return nil, err
	}
	endAt, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDate), time.Local)
	if err != nil {
		return nil, err
	}
	endExclusive := endAt.AddDate(0, 0, 1)

	parseFloat64 := func(v interface{}) (float64, bool) {
		switch vv := v.(type) {
		case nil:
			return 0, true
		case float64:
			return vv, true
		case float32:
			return float64(vv), true
		case int64:
			return float64(vv), true
		case int32:
			return float64(vv), true
		case int:
			return float64(vv), true
		case []byte:
			f, e := strconv.ParseFloat(string(vv), 64)
			return f, e == nil
		case string:
			f, e := strconv.ParseFloat(vv, 64)
			return f, e == nil
		default:
			return 0, false
		}
	}

	parseInt := func(v interface{}) (int, bool) {
		switch vv := v.(type) {
		case nil:
			return 0, true
		case int:
			return vv, true
		case int32:
			return int(vv), true
		case int64:
			return int(vv), true
		case float64:
			return int(vv), true
		case float32:
			return int(vv), true
		case []byte:
			i, e := strconv.ParseInt(string(vv), 10, 64)
			return int(i), e == nil
		case string:
			i, e := strconv.ParseInt(vv, 10, 64)
			return int(i), e == nil
		default:
			return 0, false
		}
	}

	query := `
		SELECT
			t.transaction_date,
			t.reference_id::text AS order_id,
			COALESCE(t.payment_type, 0) AS payment_type,
			COALESCE(t.invoice_number, '') AS invoice_number,
			COALESCE(t.payment_method, 0) AS payment_method,
			COALESCE(SUM(t.amount) / NULLIF(SUM(foi.quantity), 0), 0) AS amount
		FROM schedule_fleets sf
		INNER JOIN transactions t ON t.reference_id::text = sf.order_id::text
		INNER JOIN fleet_order_items foi ON foi.order_id::text = sf.order_id::text
		WHERE sf.unit_id::text = $1
			AND sf.organization_id::text = $2
			AND t.transaction_date IS NOT NULL
			AND t.transaction_date >= $3 AND t.transaction_date < $4
			AND t.status = 1 AND t.transaction_type = 1
		GROUP BY
			t.amount,
			foi.quantity,
			t.transaction_date,
			t.reference_id,
			t.payment_type,
			t.payment_method,
			t.invoice_number,
			t.created_at
		ORDER BY t.created_at DESC
	`

	rows, err := database.Query(r.db, query, unitID, orgID, startAt, endExclusive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.FleetUnitRevenueHistoryItem, 0)
	for rows.Next() {
		var it model.FleetUnitRevenueHistoryItem
		var trxDate sql.NullTime
		var orderID sql.NullString
		var paymentTypeAny interface{}
		var invoiceNumber sql.NullString
		var paymentMethodAny interface{}
		var revenueAny interface{}

		if err := rows.Scan(
			&trxDate,
			&orderID,
			&paymentTypeAny,
			&invoiceNumber,
			&paymentMethodAny,
			&revenueAny,
		); err != nil {
			return nil, err
		}

		if trxDate.Valid {
			it.TransactionDate = trxDate.Time.Format("2006-01-02")
		}
		if orderID.Valid {
			it.OrderID = strings.TrimSpace(orderID.String)
		}
		if v, ok := parseInt(paymentTypeAny); ok {
			it.PaymentType = v
		}
		if invoiceNumber.Valid {
			it.InvoiceNumber = strings.TrimSpace(invoiceNumber.String)
		}
		if v, ok := parseInt(paymentMethodAny); ok {
			it.PaymentMethod = v
		}
		if v, ok := parseFloat64(revenueAny); ok {
			it.Amount = v
		}

		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FleetUnitRepository) ListUnitExpenses(orgID, unitID string, startDate, endDate time.Time) ([]model.FleetUnitExpenseItem, error) {
	parseFloat64 := func(v interface{}) (float64, bool) {
		switch vv := v.(type) {
		case nil:
			return 0, true
		case float64:
			return vv, true
		case float32:
			return float64(vv), true
		case int64:
			return float64(vv), true
		case int32:
			return float64(vv), true
		case int:
			return float64(vv), true
		case []byte:
			f, err := strconv.ParseFloat(string(vv), 64)
			return f, err == nil
		case string:
			f, err := strconv.ParseFloat(vv, 64)
			return f, err == nil
		default:
			return 0, false
		}
	}

	parseInt := func(v interface{}) (int, bool) {
		switch vv := v.(type) {
		case nil:
			return 0, true
		case int:
			return vv, true
		case int32:
			return int(vv), true
		case int64:
			return int(vv), true
		case float64:
			return int(vv), true
		case float32:
			return int(vv), true
		case []byte:
			i, err := strconv.ParseInt(string(vv), 10, 64)
			return int(i), err == nil
		case string:
			i, err := strconv.ParseInt(vv, 10, 64)
			return int(i), err == nil
		default:
			return 0, false
		}
	}

	joinExpr := "tf.transaction_id = t.transaction_id"
	unitExpr := "tf.fleet_unit_id = " + r.placeholder(1)
	orgExpr := "t.organization_id = " + r.placeholder(2)

	selectExpr := `
		COALESCE(tf.transaction_fleet_id, '') AS transaction_fleet_id,
		COALESCE(t.transaction_category, '') AS transaction_category,
		COALESCE(t.transaction_item, '') AS transaction_item,
		COALESCE(t.description, '') AS description,
		t.transaction_date,
		COALESCE(t.payment_type, 0) AS payment_type,
		COALESCE(t.amount, 0) AS amount
	`
	if r.driver == "postgres" || r.driver == "pgx" {
		joinExpr = "tf.transaction_id::text = t.transaction_id::text"
		unitExpr = "tf.fleet_unit_id::text = " + r.placeholder(1)
		orgExpr = "t.organization_id::text = " + r.placeholder(2)
		selectExpr = `
			COALESCE(tf.transaction_fleet_id::text, '') AS transaction_fleet_id,
			COALESCE(t.transaction_category, '') AS transaction_category,
			COALESCE(t.transaction_item, '') AS transaction_item,
			COALESCE(t.description, '') AS description,
			t.transaction_date,
			COALESCE(t.payment_type, 0) AS payment_type,
			COALESCE(t.amount, 0) AS amount
		`
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM transaction_fleets tf
		INNER JOIN transactions t ON %s
		WHERE %s
			AND %s
			AND COALESCE(t.status, 0) = 1
			AND COALESCE(t.transaction_type, 0) = 2
			AND t.transaction_date >= %s
			AND t.transaction_date < %s
		ORDER BY t.transaction_date DESC
	`, selectExpr, joinExpr, unitExpr, orgExpr, r.placeholder(3), r.placeholder(4))

	rows, err := database.Query(r.db, query, unitID, orgID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.FleetUnitExpenseItem, 0)
	for rows.Next() {
		var it model.FleetUnitExpenseItem
		var transactionDate sql.NullTime
		var paymentTypeAny interface{}
		var amountAny interface{}

		if err := rows.Scan(
			&it.TransactionFleetID,
			&it.TransactionCategory,
			&it.TransactionItem,
			&it.Description,
			&transactionDate,
			&paymentTypeAny,
			&amountAny,
		); err != nil {
			return nil, err
		}

		if transactionDate.Valid {
			it.TransactionDate = transactionDate.Time.Format("2006-01-02")
		}
		if v, ok := parseInt(paymentTypeAny); ok {
			it.PaymentType = v
		}
		if v, ok := parseFloat64(amountAny); ok {
			it.Amount = v
		}

		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
