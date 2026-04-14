package repository

import (
	"database/sql"
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
WHERE fu.organization_id::text = $1
ORDER BY fu.created_at DESC
`

const listFleetUnitsMySQL = `
SELECT
	fu.unit_id,
	COALESCE(fu.vehicle_id, '') AS vehicle_id,
	fu.plate_number,
	fu.fleet_id,
	COALESCE(f.fleet_name, '') AS fleet_name,
	COALESCE(fu.engine, '') AS engine,
	COALESCE(fu.transmission, '') AS transmission,
	fu.capacity,
	fu.production_year,
	COALESCE(fu.created_by, '') AS created_by,
	fu.created_at,
	COALESCE(fu.status, 0) AS status
FROM fleet_units fu
LEFT JOIN fleets f ON fu.fleet_id = f.uuid
WHERE fu.organization_id = ?
ORDER BY fu.created_at DESC
`

const createFleetUnitPostgres = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_at)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
`

const createFleetUnitPostgresAndStatus = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_at)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
`

const createFleetUnitPostgresWithUUID = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_at)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
`

const createFleetUnitPostgresWithUUIDAndStatus = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_at)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

const createFleetUnitMySQL = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_at)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitMySQLAndStatus = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_at)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitMySQLWithUUIDAndStatus = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_at)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitMySQLWithUUID = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_at)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitPostgresCreatedDate = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_date)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
`

const createFleetUnitPostgresCreatedDateAndStatus = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_date)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
`

const createFleetUnitPostgresWithUUIDCreatedDateAndStatus = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_date)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

const createFleetUnitPostgresWithUUIDCreatedDate = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_date)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
`

const createFleetUnitMySQLCreatedDate = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_date)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitMySQLCreatedDateAndStatus = `
INSERT INTO fleet_units
	(unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_date)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitMySQLWithUUIDCreatedDateAndStatus = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, status, created_by, organization_id, created_date)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const createFleetUnitMySQLWithUUIDCreatedDate = `
INSERT INTO fleet_units
	(uuid, unit_id, vehicle_id, plate_number, fleet_id, engine, transmission, capacity, production_year, created_by, organization_id, created_date)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
	updated_at = $9
WHERE unit_id = $10 AND organization_id = $11
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
	updated_at = ?
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
	fu.updated_at
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
	fu.updated_at
FROM fleet_units fu
LEFT JOIN fleets f ON fu.fleet_id = f.uuid
LEFT JOIN fleet_types ft ON f.fleet_type = ft.id
LEFT JOIN users uc ON fu.created_by = uc.user_id
LEFT JOIN users uu ON fu.updated_by = uu.user_id
WHERE fu.unit_id = ? AND fu.organization_id = ?
`

const unitOrderHistoryPostgres = `
SELECT
	COALESCE(fuo.unit_order_id::text, '') AS unit_order_id,
	COALESCE(fuo.order_id::text, '') AS order_id,
	COALESCE(fuo.unit_id::text, '') AS unit_id,
	COALESCE(fuo.driver_id::text, '') AS driver_id,
	COALESCE(d.fullname, '') AS driver_name,
	fo.start_date,
	fo.end_date,
	COALESCE(fo.pickup_city_id::text, '') AS pickup_city_id
FROM fleet_unit_orders fuo
INNER JOIN fleet_orders fo ON fuo.order_id::text = fo.order_id::text
LEFT JOIN users d ON d.user_id::text = fuo.driver_id::text
WHERE fo.organization_id::text = $1 AND fuo.unit_id::text = $2 AND fo.start_date >= $3 AND fo.end_date <= $4
ORDER BY fo.start_date DESC
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

func (r *FleetUnitRepository) List(orgID string) ([]model.FleetUnitListItem, error) {
	query := listFleetUnitsMySQL
	if r.driver == "postgres" || r.driver == "pgx" {
		query = listFleetUnitsPostgres
	}
	rows, err := database.Query(r.db, query, orgID)
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
		err = tryExec(createFleetUnitPostgresWithUUIDAndStatus, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate)
		if err != nil {
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "column") && strings.Contains(errMsg, "status") && strings.Contains(errMsg, "does not exist") {
				err = tryExec(createFleetUnitPostgresWithUUID, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate)
				errMsg = strings.ToLower(err.Error())
			}
			if strings.Contains(errMsg, "column") && strings.Contains(errMsg, "uuid") && strings.Contains(errMsg, "does not exist") {
				err = tryExec(createFleetUnitPostgresAndStatus, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate)
				if err != nil {
					errMsg2 := strings.ToLower(err.Error())
					if strings.Contains(errMsg2, "column") && strings.Contains(errMsg2, "status") && strings.Contains(errMsg2, "does not exist") {
						err = tryExec(createFleetUnitPostgres, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate)
					}
				}
			}
		}
		if err != nil {
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "created_at") && strings.Contains(errMsg, "does not exist") {
				err = tryExec(createFleetUnitPostgresWithUUIDCreatedDateAndStatus, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate)
				if err != nil {
					errMsg2 := strings.ToLower(err.Error())
					if strings.Contains(errMsg2, "column") && strings.Contains(errMsg2, "status") && strings.Contains(errMsg2, "does not exist") {
						err = tryExec(createFleetUnitPostgresWithUUIDCreatedDate, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate)
						errMsg2 = strings.ToLower(err.Error())
					}
					if strings.Contains(errMsg2, "column") && strings.Contains(errMsg2, "uuid") && strings.Contains(errMsg2, "does not exist") {
						err = tryExec(createFleetUnitPostgresCreatedDateAndStatus, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate)
						if err != nil {
							errMsg3 := strings.ToLower(err.Error())
							if strings.Contains(errMsg3, "column") && strings.Contains(errMsg3, "status") && strings.Contains(errMsg3, "does not exist") {
								err = tryExec(createFleetUnitPostgresCreatedDate, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate)
							}
						}
					}
				}
			}
		}
	} else {
		err = tryExec(createFleetUnitMySQLWithUUIDAndStatus, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate)
		if err != nil {
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "unknown column") && strings.Contains(errMsg, "status") {
				err = tryExec(createFleetUnitMySQLWithUUID, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate)
				errMsg = strings.ToLower(err.Error())
			}
			if strings.Contains(errMsg, "unknown column") && strings.Contains(errMsg, "uuid") {
				err = tryExec(createFleetUnitMySQLAndStatus, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate)
				if err != nil {
					errMsg2 := strings.ToLower(err.Error())
					if strings.Contains(errMsg2, "unknown column") && strings.Contains(errMsg2, "status") {
						err = tryExec(createFleetUnitMySQL, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate)
					}
				}
			}
		}
		if err != nil {
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "unknown column") && strings.Contains(errMsg, "created_at") {
				err = tryExec(createFleetUnitMySQLWithUUIDCreatedDateAndStatus, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate)
				if err != nil {
					errMsg2 := strings.ToLower(err.Error())
					if strings.Contains(errMsg2, "unknown column") && strings.Contains(errMsg2, "status") {
						err = tryExec(createFleetUnitMySQLWithUUIDCreatedDate, id, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate)
						errMsg2 = strings.ToLower(err.Error())
					}
					if strings.Contains(errMsg2, "unknown column") && strings.Contains(errMsg2, "uuid") {
						err = tryExec(createFleetUnitMySQLCreatedDateAndStatus, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, 1, req.CreatedBy, req.OrganizationID, req.CreatedDate)
						if err != nil {
							errMsg3 := strings.ToLower(err.Error())
							if strings.Contains(errMsg3, "unknown column") && strings.Contains(errMsg3, "status") {
								err = tryExec(createFleetUnitMySQLCreatedDate, req.UnitID, req.VehicleID, req.PlateNumber, req.FleetID, req.Engine, req.Transmission, req.Capacity, req.ProductionYear, req.CreatedBy, req.OrganizationID, req.CreatedDate)
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
	)
	if err != nil {
		return nil, err
	}
	res.CreatedDate = createdAt.Format("2006-01-02 15:04:05")
	if updatedAt.Valid {
		res.UpdatedDate = updatedAt.Time.Format("2006-01-02 15:04:05")
	}
	return &res, nil
}

func (r *FleetUnitRepository) UnitOrderHistory(orgID, unitID, startDate, endDate string) ([]model.FleetUnitOrderHistoryItem, error) {
	query := unitOrderHistoryMySQL
	if r.driver == "postgres" || r.driver == "pgx" {
		query = unitOrderHistoryPostgres
	}

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
			&it.UnitOrderID,
			&it.OrderID,
			&it.UnitID,
			&it.DriverID,
			&it.DriverName,
			&startDate,
			&endDate,
			&it.PickupCityID,
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

	query := "SELECT COALESCE(order_id, '') AS order_id, COALESCE(CAST(city_id AS CHAR), '') AS city_id FROM fleet_order_destinations WHERE order_id IN (" + strings.Join(in, ",") + ")"
	if r.driver == "postgres" || r.driver == "pgx" {
		query = "SELECT COALESCE(order_id::text, '') AS order_id, COALESCE(city_id::text, '') AS city_id FROM fleet_order_destinations WHERE order_id::text IN (" + strings.Join(in, ",") + ")"
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
