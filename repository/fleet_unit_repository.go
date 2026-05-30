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
	COALESCE(fu.status, 0) AS status
FROM fleet_units fu
LEFT JOIN fleets f ON f.uuid::text = fu.fleet_id::text
LEFT JOIN schedule_fleets sf ON sf.fleet_id::text = fu.fleet_id::text
AND sf.order_id::text = $3
WHERE fu.organization_id::text = $1 AND (fu.fleet_id::text = $2 OR $2 = '')
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
	COALESCE(fu.status, 0) AS status
FROM fleet_units fu
LEFT JOIN fleets f ON f.uuid::text = fu.fleet_id::text
INNER JOIN schedule_fleets sf ON sf.fleet_id::text = fu.fleet_id::text
WHERE fu.organization_id::text = $1 AND (fu.fleet_id::text = $2 OR $2 = '') AND sf.order_id::text = $3
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

const unitOrderHistoryMySQL = `
SELECT
	fuo.unit_order_id,
	fuo.order_id,
	fuo.unit_id,
	COALESCE(fuo.driver_id, '') AS driver_id,
	COALESCE(d.fullname, '') AS driver_name,
	fo.start_date,
	fo.end_date,
	COALESCE(CAST(fo.pickup_city_id AS CHAR), '') AS pickup_city_id
FROM fleet_unit_orders fuo
INNER JOIN fleet_orders fo ON fuo.order_id = fo.order_id
LEFT JOIN users d ON d.user_id = fuo.driver_id
WHERE fo.organization_id = ? AND fuo.unit_id = ? AND fo.start_date >= ? AND fo.end_date <= ?
ORDER BY fo.start_date DESC
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

const unitTotalSchedulesPostgres = `
SELECT COUNT(*) AS total_schedules
FROM schedule_fleets
WHERE unit_id::text = $1 AND organization_id::text = $2
`

const unitTotalSchedulesMySQL = `
SELECT COUNT(*) AS total_schedules
FROM schedule_fleets
WHERE unit_id = ? AND organization_id = ?
`

const unitLatestSchedulePostgres = `
SELECT fo.start_date, fo.end_date
FROM schedule_fleets sf
INNER JOIN fleet_orders fo ON sf.order_id = fo.order_id
WHERE sf.unit_id::text = $1 AND sf.organization_id::text = $2 AND fo.end_date <= $3
ORDER BY fo.end_date DESC
LIMIT 1
`

const unitLatestScheduleMySQL = `
SELECT fo.start_date, fo.end_date
FROM schedule_fleets sf
INNER JOIN fleet_orders fo ON sf.order_id = fo.order_id
WHERE sf.unit_id = ? AND sf.organization_id = ? AND fo.end_date <= ?
ORDER BY fo.end_date DESC
LIMIT 1
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

func (r *FleetUnitRepository) List(orgID, fleetId, orderID string) ([]model.FleetUnitListItem, error) {

	query := listFleetUnitsPostgres
	if strings.TrimSpace(orderID) != "" {
		query = listFleetUnitsPostgresByOrderID
	}

	args := make([]interface{}, 0, 3)
	args = append(args, orgID)
	args = append(args, fleetId)
	args = append(args, orderID)
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
	WHERE fo.organization_id::text = $1 AND fuo.unit_id::text = $2 AND fo.start_date >= $3 AND fo.end_date <= $4
	GROUP BY fuo.order_id, fuo.unit_id, sft.driver_id, d.fullname, fo.start_date, fo.end_date, fo.status, fo.pickup_city_id, fi.city_id
	ORDER BY fo.start_date DESC
	`

	rows, err := database.Query(r.db, query, orgID, unitID, startDate, endDate)
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
			&it.Destinations,
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

func (r *FleetUnitRepository) UnitLatestSchedule(orgID, unitID, today string) (*model.FleetUnitScheduleRange, error) {
	query := `SELECT fo.start_date, fo.end_date
			FROM schedule_fleets sf
			INNER JOIN fleet_orders fo ON sf.order_id = fo.order_id
			WHERE sf.unit_id::text = $1 AND sf.organization_id::text = $2 AND fo.end_date <= $3
			ORDER BY fo.end_date DESC
			LIMIT 1
			`
	var startDate sql.NullTime
	var endDate sql.NullTime
	if err := database.QueryRow(r.db, query, unitID, orgID, today).Scan(&startDate, &endDate); err != nil {
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

func (r *FleetUnitRepository) UnitUpcomingSchedule(orgID, unitID, today string) (*model.FleetUnitScheduleRange, error) {
	query := unitUpcomingScheduleMySQL
	if r.driver == "postgres" || r.driver == "pgx" {
		query = unitUpcomingSchedulePostgres
	}
	var startDate sql.NullTime
	var endDate sql.NullTime
	if err := database.QueryRow(r.db, query, unitID, orgID, today).Scan(&startDate, &endDate); err != nil {
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
	query := fmt.Sprintf(`
		SELECT SUM(po.payment_amount) AS revenue,
		COUNT(fo.order_id) as total_booking
		FROM fleet_orders fo
		INNER JOIN payment_orders po ON po.order_id = fo.order_id
		INNER JOIN schedule_fleets sf ON sf.order_id = po.order_id
		WHERE sf.unit_id = %s AND fo.organization_id = %s AND fo.status = 1 AND fo.payment_status NOT IN (0,2) AND po.created_at BETWEEN %s AND %s 
		GROUP BY sf.unit_id
	`, r.placeholder(1), r.placeholder(2), r.placeholder(3), r.placeholder(4))
	var revenueAny interface{}
	var totalBooking int64
	if err := database.QueryRow(r.db, query, unitID, orgID, startDate, endDate).Scan(&revenueAny, &totalBooking); err != nil {
		return nil, err
	}
	switch v := revenueAny.(type) {
	case nil:
		return &model.FleetUnitRevenue{TotalRevenue: 0, TotalBooking: 0}, nil
	case float64:
		return &model.FleetUnitRevenue{TotalRevenue: v, TotalBooking: totalBooking}, nil
	case int64:
		return &model.FleetUnitRevenue{TotalRevenue: float64(v), TotalBooking: totalBooking}, nil
	case []byte:
		if f, err := strconv.ParseFloat(string(v), 64); err == nil {
			return &model.FleetUnitRevenue{TotalRevenue: f, TotalBooking: totalBooking}, nil
		}
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return &model.FleetUnitRevenue{TotalRevenue: f, TotalBooking: totalBooking}, nil
		}
	}
	return &model.FleetUnitRevenue{TotalRevenue: 0, TotalBooking: 0}, nil
}
