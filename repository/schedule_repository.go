package repository

import (
	"database/sql"
	"service-travego/database"
	"service-travego/model"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type ScheduleRepository struct {
	db     *sql.DB
	driver string
}

func NewScheduleRepository(db *sql.DB, driver string) *ScheduleRepository {
	return &ScheduleRepository{db: db, driver: driver}
}

func (r *ScheduleRepository) placeholder(position int) string {
	if r.driver == "postgres" || r.driver == "pgx" {
		return "$" + strconv.Itoa(position)
	}
	return "?"
}

func (r *ScheduleRepository) OrderPaymentStatus(input model.ScheduleOrderValidationInput) (int, bool, error) {
	orgExpr := "organization_id = " + r.placeholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.placeholder(2)
	}
	query := "SELECT payment_status FROM fleet_orders WHERE order_id = " + r.placeholder(1) + " AND " + orgExpr + " LIMIT 1"

	var paymentStatus sql.NullInt64
	if err := database.QueryRow(r.db, query, input.OrderID, input.OrganizationID).Scan(&paymentStatus); err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	if !paymentStatus.Valid {
		return 0, true, nil
	}
	return int(paymentStatus.Int64), true, nil
}

func (r *ScheduleRepository) OrderItemExists(input model.ScheduleOrderItemValidationInput) (bool, error) {
	orderExpr := "order_id = " + r.placeholder(2)
	orgExpr := "organization_id = " + r.placeholder(1)
	fleetExpr := "fleet_id = " + r.placeholder(3)

	if r.driver == "postgres" || r.driver == "pgx" {
		orderExpr = "order_id::text = " + r.placeholder(2)
		orgExpr = "organization_id::text = " + r.placeholder(1)
		fleetExpr = "fleet_id::text = " + r.placeholder(3)
	}

	query := "SELECT COUNT(1) FROM fleet_order_items WHERE " + orgExpr + " AND " + orderExpr + " AND " + fleetExpr
	var count int
	if err := database.QueryRow(r.db, query, input.OrganizationID, input.OrderID, input.FleetID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ScheduleRepository) CreateSchedule(input model.ScheduleCreateRepositoryInput) (string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	scheduleID := uuid.New().String()
	insertSchedule := `
		INSERT INTO schedules (schedule_id, order_id, organization_id, departure_time, status, created_at, created_by, order_type)
		VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `, ` + r.placeholder(4) + `, 1, ` + r.placeholder(5) + `, ` + r.placeholder(6) + `, 1)
	`
	if _, err = database.TxExec(tx, insertSchedule, scheduleID, input.OrderID, input.OrganizationID, input.DepartureTime, input.CreatedAt, input.UserID); err != nil {
		return "", err
	}

	selectLatestSchedule := `
		SELECT schedule_id
		FROM schedules
		WHERE order_id = ` + r.placeholder(1) + ` AND organization_id = ` + r.placeholder(2) + `
		ORDER BY created_at DESC
		LIMIT 1
	`
	if r.driver == "postgres" || r.driver == "pgx" {
		selectLatestSchedule = `
			SELECT schedule_id::text
			FROM schedules
			WHERE order_id::text = ` + r.placeholder(1) + ` AND organization_id::text = ` + r.placeholder(2) + `
			ORDER BY created_at DESC
			LIMIT 1
		`
	}

	if err = database.TxQueryRow(tx, selectLatestSchedule, input.OrderID, input.OrganizationID).Scan(&scheduleID); err != nil {
		return "", err
	}

	for _, fleet := range input.Fleets {
		scheduleFleetID := uuid.New().String()
		insertScheduleFleet := `
			INSERT INTO schedule_fleets (uuid, schedule_id, order_id, fleet_d, unit_id, departure_time, created_at, created_by, status, organization_id)
			VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `, ` + r.placeholder(4) + `, ` + r.placeholder(5) + `, ` + r.placeholder(6) + `, ` + r.placeholder(7) + `, ` + r.placeholder(8) + `, 1, ` + r.placeholder(9) + `)
		`
		if _, err = database.TxExec(tx, insertScheduleFleet, scheduleFleetID, scheduleID, input.OrderID, fleet.FleetID, fleet.UnitID, input.DepartureTime, input.CreatedAt, input.UserID, input.OrganizationID); err != nil {
			return "", err
		}

		for _, employeeID := range fleet.DriverID {
			driverID := strings.TrimSpace(employeeID)
			if driverID == "" {
				continue
			}
			insertTeam := `
				INSERT INTO schedule_fleet_teams (uuid, schedule_id, unit_id, schedule_fleet_id, employee_id, created_by, created_at, organization_id, status)
				VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `, ` + r.placeholder(4) + `, ` + r.placeholder(5) + `, ` + r.placeholder(6) + `, ` + r.placeholder(7) + `, ` + r.placeholder(8) + `, 1)
			`
			if _, err = database.TxExec(tx, insertTeam, uuid.New().String(), scheduleID, fleet.UnitID, scheduleFleetID, driverID, input.UserID, input.CreatedAt, input.OrganizationID); err != nil {
				return "", err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return "", err
	}
	return scheduleID, nil
}

func (r *ScheduleRepository) ListScheduleFleetOrders(input model.ScheduleFleetListQuery, organizationID string) ([]model.ScheduleFleetOrderRow, error) {
	orgExpr := "s.organization_id = " + r.placeholder(1)
	departureExpr := "COALESCE(CAST(s.departure_time AS CHAR), '')"
	arrivalExpr := "COALESCE(CAST(s.departure_end AS CHAR), '')"
	pickupCityExpr := "COALESCE(CAST(fo.pickup_city_id AS CHAR), '')"
	createdByExpr := "COALESCE(CAST(s.created_by AS CHAR), '')"
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "s.organization_id::text = " + r.placeholder(1)
		departureExpr = "COALESCE(s.departure_time::text, '')"
		arrivalExpr = "COALESCE(s.departure_end::text, '')"
		pickupCityExpr = "COALESCE(fo.pickup_city_id::text, '')"
		createdByExpr = "COALESCE(s.created_by::text, '')"
	}

	query := `
		SELECT
			s.schedule_id,
			fo.start_date,
			fo.end_date,
			` + departureExpr + ` AS departure_time,
			` + arrivalExpr + ` AS arrival_time,
			COALESCE(s.status, 0) AS schedule_status,
			COALESCE(fo.unit_qty, 0) AS unit_qty,
			` + pickupCityExpr + ` AS pickup_city_id,
			COALESCE(fo.additional_request, '') AS additional_request,
			COALESCE(fo.payment_status, 0) AS payment_status,
			COALESCE(s.created_at, CURRENT_TIMESTAMP) AS created_at,
			` + createdByExpr + ` AS created_by
		FROM schedules s
		INNER JOIN fleet_orders fo ON s.order_id = fo.order_id
		WHERE ` + orgExpr + ` AND s.order_type = 1
	`

	args := []interface{}{organizationID}
	position := 2
	if strings.TrimSpace(input.StartDate) != "" {
		query += " AND fo.start_date = " + r.placeholder(position)
		args = append(args, input.StartDate)
		position++
	}
	if strings.TrimSpace(input.EndDate) != "" {
		query += " AND fo.end_date = " + r.placeholder(position)
		args = append(args, input.EndDate)
		position++
	}

	fleetFilters := make([]string, 0, 6)
	capacityExpr := "CAST(u.capacity AS CHAR)"
	productionYearExpr := "CAST(u.production_year AS CHAR)"
	if r.driver == "postgres" || r.driver == "pgx" {
		capacityExpr = "u.capacity::text"
		productionYearExpr = "u.production_year::text"
	}
	if clause, values := r.buildFleetFilterClause("f.fleet_name", input.FleetName, position); clause != "" {
		fleetFilters = append(fleetFilters, clause)
		args = append(args, values...)
		position += len(values)
	}
	if clause, values := r.buildFleetFilterClause("u.plate_number", input.PlateNumber, position); clause != "" {
		fleetFilters = append(fleetFilters, clause)
		args = append(args, values...)
		position += len(values)
	}
	if clause, values := r.buildFleetFilterClause("u.vehicle_id", input.VehicleID, position); clause != "" {
		fleetFilters = append(fleetFilters, clause)
		args = append(args, values...)
		position += len(values)
	}
	if clause, values := r.buildFleetFilterClause("u.engine", input.Engine, position); clause != "" {
		fleetFilters = append(fleetFilters, clause)
		args = append(args, values...)
		position += len(values)
	}
	if clause, values := r.buildFleetFilterClause(capacityExpr, input.Capacity, position); clause != "" {
		fleetFilters = append(fleetFilters, clause)
		args = append(args, values...)
		position += len(values)
	}
	if clause, values := r.buildFleetFilterClause(productionYearExpr, input.ProductionYear, position); clause != "" {
		fleetFilters = append(fleetFilters, clause)
		args = append(args, values...)
		position += len(values)
	}

	if len(fleetFilters) > 0 {
		query += `
			AND EXISTS (
				SELECT 1
				FROM schedule_fleets sf
				INNER JOIN fleet_units u ON sf.unit_id = u.unit_id
				INNER JOIN fleets f ON u.fleet_id = f.uuid
				WHERE sf.schedule_id = s.schedule_id
				  AND sf.organization_id = s.organization_id
				  AND ` + strings.Join(fleetFilters, " AND ") + `
			)
		`
	}
	query += " ORDER BY fo.start_date ASC, s.created_at DESC"

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]model.ScheduleFleetOrderRow, 0)
	for rows.Next() {
		var item model.ScheduleFleetOrderRow
		var createdBy sql.NullString
		var pickupCityID sql.NullString
		var additionalRequest sql.NullString
		var departureTime sql.NullString
		var arrivalTime sql.NullString

		if err := rows.Scan(
			&item.ScheduleID,
			&item.StartDate,
			&item.EndDate,
			&departureTime,
			&arrivalTime,
			&item.ScheduleStatus,
			&item.UnitQty,
			&pickupCityID,
			&additionalRequest,
			&item.PaymentStatus,
			&item.CreatedAt,
			&createdBy,
		); err != nil {
			return nil, err
		}

		item.DepartureTime = departureTime.String
		item.ArrivalTime = arrivalTime.String
		item.PickupCityID = pickupCityID.String
		item.AdditionalRequest = additionalRequest.String
		item.CreatedBy = createdBy.String

		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *ScheduleRepository) buildFleetFilterClause(columnName, queryValue string, position int) (string, []interface{}) {
	value := strings.TrimSpace(queryValue)
	if value == "" {
		return "", nil
	}

	if strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		filtered := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				filtered = append(filtered, trimmed)
			}
		}
		if len(filtered) == 0 {
			return "", nil
		}

		placeholders := make([]string, 0, len(filtered))
		args := make([]interface{}, 0, len(filtered))
		for i, item := range filtered {
			placeholders = append(placeholders, r.placeholder(position+i))
			args = append(args, item)
		}
		return columnName + " IN (" + strings.Join(placeholders, ",") + ")", args
	}

	return columnName + " LIKE " + r.placeholder(position), []interface{}{"%" + value + "%"}
}

func (r *ScheduleRepository) ListScheduleFleets(scheduleID, organizationID string) ([]model.ScheduleFleetListUnit, error) {
	scheduleExpr := "sf.schedule_id = " + r.placeholder(1)
	orgExpr := "sf.organization_id = " + r.placeholder(2)
	fleetIDExpr := "COALESCE(CAST(sf.uuid AS CHAR), '')"
	if r.driver == "postgres" || r.driver == "pgx" {
		scheduleExpr = "sf.schedule_id::text = " + r.placeholder(1)
		orgExpr = "sf.organization_id::text = " + r.placeholder(2)
		fleetIDExpr = "COALESCE(sf.uuid::text, '')"
	}

	query := `
		SELECT
			` + fleetIDExpr + ` AS fleet_id,
			COALESCE(f.fleet_name, '') AS fleet_name,
			COALESCE(u.vehicle_id, '') AS vehicle_id,
			COALESCE(u.plate_number, '') AS plate_number,
			COALESCE(u.engine, '') AS engine,
			COALESCE(u.capacity, 0) AS capacity
		FROM schedule_fleets sf
		INNER JOIN fleet_units u ON sf.unit_id = u.unit_id
		INNER JOIN fleets f ON u.fleet_id = f.uuid
		WHERE ` + scheduleExpr + ` AND ` + orgExpr + `
		ORDER BY f.fleet_name ASC
	`

	rows, err := database.Query(r.db, query, scheduleID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]model.ScheduleFleetListUnit, 0)
	for rows.Next() {
		var item model.ScheduleFleetListUnit
		if err := rows.Scan(
			&item.FleetID,
			&item.FleetName,
			&item.VehicleID,
			&item.PlateNumber,
			&item.Engine,
			&item.Capacity,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
