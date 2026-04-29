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

type ScheduleRepository struct {
	db     *sql.DB
	driver string
}

type inClauseInput struct {
	ColumnName string
	Values     []string
	Position   int
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

func (r *ScheduleRepository) UpdateSchedule(input model.ScheduleUpdateRepositoryInput) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	scheduleExpr := "schedule_id = " + r.placeholder(1)
	orgExpr := "organization_id = " + r.placeholder(2)
	orderExpr := "order_id = " + r.placeholder(3)
	if r.driver == "postgres" || r.driver == "pgx" {
		scheduleExpr = "schedule_id::text = " + r.placeholder(1)
		orgExpr = "organization_id::text = " + r.placeholder(2)
		orderExpr = "order_id::text = " + r.placeholder(3)
	}

	if input.ArrivalTime == nil {
		updateSchedule := `
			UPDATE schedules
			SET departure_time = ` + r.placeholder(4) + `, updated_at = ` + r.placeholder(5) + `, updated_by = ` + r.placeholder(6) + `
			WHERE ` + scheduleExpr + ` AND ` + orgExpr + ` AND ` + orderExpr + `
		`
		res, execErr := database.TxExec(tx, updateSchedule, input.ScheduleID, input.OrganizationID, input.OrderID, input.DepartureTime, input.UpdatedAt, input.UserID)
		if execErr != nil {
			return execErr
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			return sql.ErrNoRows
		}
	} else {
		updateSchedule := `
			UPDATE schedules
			SET departure_time = ` + r.placeholder(4) + `, arrival_time = ` + r.placeholder(5) + `, updated_at = ` + r.placeholder(6) + `, updated_by = ` + r.placeholder(7) + `
			WHERE ` + scheduleExpr + ` AND ` + orgExpr + ` AND ` + orderExpr + `
		`
		res, execErr := database.TxExec(tx, updateSchedule, input.ScheduleID, input.OrganizationID, input.OrderID, input.DepartureTime, *input.ArrivalTime, input.UpdatedAt, input.UserID)
		if execErr != nil {
			return execErr
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			return sql.ErrNoRows
		}
	}

	scheduleFleetExpr := "schedule_id = " + r.placeholder(1)
	scheduleFleetOrgExpr := "organization_id = " + r.placeholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		scheduleFleetExpr = "schedule_id::text = " + r.placeholder(1)
		scheduleFleetOrgExpr = "organization_id::text = " + r.placeholder(2)
	}

	selectExisting := `
		SELECT
			COALESCE(CAST(uuid AS CHAR), '') AS uuid,
			COALESCE(CAST(unit_id AS CHAR), '') AS unit_id
		FROM schedule_fleets
		WHERE ` + scheduleFleetExpr + ` AND ` + scheduleFleetOrgExpr + ` AND status = 1
	`
	if r.driver == "postgres" || r.driver == "pgx" {
		selectExisting = `
			SELECT
				COALESCE(uuid::text, '') AS uuid,
				COALESCE(unit_id::text, '') AS unit_id
			FROM schedule_fleets
			WHERE ` + scheduleFleetExpr + ` AND ` + scheduleFleetOrgExpr + ` AND status = 1
		`
	}

	rows, qErr := database.TxQuery(tx, selectExisting, input.ScheduleID, input.OrganizationID)
	if qErr != nil {
		return qErr
	}
	defer rows.Close()

	existingByUnit := map[string]string{}
	for rows.Next() {
		var uuidText string
		var unitID string
		if err := rows.Scan(&uuidText, &unitID); err != nil {
			return err
		}
		unitID = strings.TrimSpace(unitID)
		uuidText = strings.TrimSpace(uuidText)
		if unitID != "" && uuidText != "" {
			existingByUnit[unitID] = uuidText
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, fleet := range input.Fleets {
		unitID := strings.TrimSpace(fleet.UnitID)
		if unitID == "" {
			continue
		}

		scheduleFleetID := existingByUnit[unitID]
		if scheduleFleetID == "" {
			scheduleFleetID = uuid.New().String()
			insertScheduleFleet := `
				INSERT INTO schedule_fleets (uuid, schedule_id, order_id, fleet_d, unit_id, departure_time, created_at, created_by, status, organization_id)
				VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `, ` + r.placeholder(4) + `, ` + r.placeholder(5) + `, ` + r.placeholder(6) + `, ` + r.placeholder(7) + `, ` + r.placeholder(8) + `, 1, ` + r.placeholder(9) + `)
			`
			if _, err = database.TxExec(tx, insertScheduleFleet, scheduleFleetID, input.ScheduleID, input.OrderID, fleet.FleetID, unitID, input.DepartureTime, input.UpdatedAt, input.UserID, input.OrganizationID); err != nil {
				return err
			}
		} else {
			updateScheduleFleet := `
				UPDATE schedule_fleets
				SET fleet_d = ` + r.placeholder(3) + `, departure_time = ` + r.placeholder(4) + `
				WHERE uuid = ` + r.placeholder(1) + ` AND organization_id = ` + r.placeholder(2) + `
			`
			if r.driver == "postgres" || r.driver == "pgx" {
				updateScheduleFleet = `
					UPDATE schedule_fleets
					SET fleet_d = ` + r.placeholder(3) + `, departure_time = ` + r.placeholder(4) + `
					WHERE uuid::text = ` + r.placeholder(1) + ` AND organization_id::text = ` + r.placeholder(2) + `
				`
			}
			if _, err = database.TxExec(tx, updateScheduleFleet, scheduleFleetID, input.OrganizationID, fleet.FleetID, input.DepartureTime); err != nil {
				return err
			}
		}

		deleteTeams := `
			DELETE FROM schedule_fleet_teams
			WHERE schedule_fleet_id = ` + r.placeholder(1) + ` AND organization_id = ` + r.placeholder(2) + `
		`
		if r.driver == "postgres" || r.driver == "pgx" {
			deleteTeams = `
				DELETE FROM schedule_fleet_teams
				WHERE schedule_fleet_id::text = ` + r.placeholder(1) + ` AND organization_id::text = ` + r.placeholder(2) + `
			`
		}
		if _, err = database.TxExec(tx, deleteTeams, scheduleFleetID, input.OrganizationID); err != nil {
			return err
		}

		insertTeam := `
			INSERT INTO schedule_fleet_teams (uuid, schedule_id, unit_id, schedule_fleet_id, employee_id, created_by, created_at, organization_id, status)
			VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `, ` + r.placeholder(4) + `, ` + r.placeholder(5) + `, ` + r.placeholder(6) + `, ` + r.placeholder(7) + `, ` + r.placeholder(8) + `, 1)
		`

		for _, employeeID := range fleet.DriverID {
			driverID := strings.TrimSpace(employeeID)
			if driverID == "" {
				continue
			}
			if _, err = database.TxExec(tx, insertTeam, uuid.New().String(), input.ScheduleID, unitID, scheduleFleetID, driverID, input.UserID, input.UpdatedAt, input.OrganizationID); err != nil {
				return err
			}
		}
		for _, employeeID := range fleet.CrewID {
			crewID := strings.TrimSpace(employeeID)
			if crewID == "" {
				continue
			}
			if _, err = database.TxExec(tx, insertTeam, uuid.New().String(), input.ScheduleID, unitID, scheduleFleetID, crewID, input.UserID, input.UpdatedAt, input.OrganizationID); err != nil {
				return err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *ScheduleRepository) ListScheduleFleetOrders(input model.ScheduleFleetListQuery, organizationID string, monthStart, monthEnd time.Time) ([]model.ScheduleFleetOrderRow, error) {
	orgExpr := "s.organization_id = " + r.placeholder(1)
	departureExpr := "COALESCE(CAST(s.departure_time AS CHAR), '')"
	arrivalExpr := "COALESCE(CAST(s.arrival_time AS CHAR), '')"
	orderIDExpr := "COALESCE(CAST(fo.order_id AS CHAR), '')"
	pickupCityExpr := "COALESCE(CAST(fo.pickup_city_id AS CHAR), '')"
	createdByExpr := "COALESCE(CAST(s.created_by AS CHAR), '')"
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "s.organization_id::text = " + r.placeholder(1)
		departureExpr = "COALESCE(s.departure_time::text, '')"
		arrivalExpr = "COALESCE(s.arrival_time::text, '')"
		orderIDExpr = "COALESCE(fo.order_id::text, '')"
		pickupCityExpr = "COALESCE(fo.pickup_city_id::text, '')"
		createdByExpr = "COALESCE(s.created_by::text, '')"
	}

	query := `
		SELECT
			s.schedule_id,
			` + orderIDExpr + ` AS order_id,
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

	query += `
		AND (
			(fo.start_date >= ` + r.placeholder(position) + ` AND fo.start_date <= ` + r.placeholder(position+1) + `)
			OR
			(fo.end_date >= ` + r.placeholder(position) + ` AND fo.end_date <= ` + r.placeholder(position+1) + `)
		)
	`
	args = append(args, monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02"))
	position += 2

	fleetFilters := make([]string, 0, 6)
	capacityExpr := "CAST(u.capacity AS CHAR)"
	productionYearExpr := "CAST(u.production_year AS CHAR)"
	scheduleFleetOrderIDExpr := "CAST(sf.order_id AS CHAR)"
	scheduleFleetIDExpr := "CAST(sf.fleet_id AS CHAR)"
	scheduleFleetUnitIDExpr := "CAST(sf.unit_id AS CHAR)"
	if r.driver == "postgres" || r.driver == "pgx" {
		capacityExpr = "u.capacity::text"
		productionYearExpr = "u.production_year::text"
		scheduleFleetOrderIDExpr = "sf.order_id::text"
		scheduleFleetIDExpr = "sf.fleet_id::text"
		scheduleFleetUnitIDExpr = "sf.unit_id::text"
	}
	if clause, values := r.buildExactFilterClause(scheduleFleetOrderIDExpr, input.OrderID, position); clause != "" {
		fleetFilters = append(fleetFilters, clause)
		args = append(args, values...)
		position += len(values)
	}
	if clause, values := r.buildExactFilterClause(scheduleFleetIDExpr, input.FleetID, position); clause != "" {
		fleetFilters = append(fleetFilters, clause)
		args = append(args, values...)
		position += len(values)
	}
	if clause, values := r.buildExactFilterClause(scheduleFleetUnitIDExpr, input.UnitID, position); clause != "" {
		fleetFilters = append(fleetFilters, clause)
		args = append(args, values...)
		position += len(values)
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
			&item.OrderID,
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

func (r *ScheduleRepository) GetFleetAvailability(filter model.ScheduleFleetAvailabilityFilter, organizationID string) ([]model.ScheduleFleetAvailabilityRow, error) {
	orgExpr := "s.organization_id = " + r.placeholder(1)
	scheduleIDExpr := "COALESCE(CAST(s.schedule_id AS CHAR), '')"
	departureTimeExpr := "COALESCE(CAST(s.departure_time AS CHAR), '')"
	arrivalTimeExpr := "COALESCE(CAST(s.arrival_time AS CHAR), '')"
	capacityExpr := "CAST(fu.capacity AS CHAR)"
	productionYearExpr := "CAST(fu.production_year AS CHAR)"
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "s.organization_id::text = " + r.placeholder(1)
		scheduleIDExpr = "COALESCE(s.schedule_id::text, '')"
		departureTimeExpr = "COALESCE(s.departure_time::text, '')"
		arrivalTimeExpr = "COALESCE(s.arrival_time::text, '')"
		capacityExpr = "fu.capacity::text"
		productionYearExpr = "fu.production_year::text"
	}

	query := `
		SELECT DISTINCT
			` + scheduleIDExpr + ` AS schedule_id,
			COALESCE(ft.label, '') AS fleet_type,
			COALESCE(f.fleet_name, '') AS fleet_name,
			` + departureTimeExpr + ` AS departure_time,
			` + arrivalTimeExpr + ` AS arrival_time,
			fo.start_date,
			fo.end_date,
			COALESCE(fu.vehicle_id, '') AS vehicle_id,
			COALESCE(fu.plate_number, '') AS plate_number,
			COALESCE(fu.engine, '') AS engine,
			COALESCE(fu.capacity, 0) AS capacity,
			COALESCE(fu.production_year, 0) AS production_year,
			COALESCE(fu.transmission, '') AS transmission
		FROM schedules s
		INNER JOIN fleet_orders fo ON fo.order_id = s.order_id
		INNER JOIN schedule_fleets sf ON sf.schedule_id = s.schedule_id
		INNER JOIN fleets f ON f.uuid = fo.fleet_id
		INNER JOIN fleet_order_items foi ON foi.order_id = fo.order_id
		INNER JOIN fleet_units fu ON sf.unit_id = fu.unit_id
		INNER JOIN fleet_types ft ON ft.id = f.fleet_type
		WHERE s.order_type = 1
		  AND s.status = 1
		  AND ` + orgExpr + `
		  AND fo.start_date <= ` + r.placeholder(2) + `
		  AND fo.end_date >= ` + r.placeholder(3) + `
	`
	args := []interface{}{organizationID, filter.EndDate, filter.StartDate}
	position := 4

	if clause, values := r.buildInClause(inClauseInput{ColumnName: "fu.vehicle_id", Values: filter.VehicleID, Position: position}); clause != "" {
		query += " AND " + clause
		args = append(args, values...)
		position += len(values)
	}
	if clause, values := r.buildInClause(inClauseInput{ColumnName: "f.fleet_name", Values: filter.FleetName, Position: position}); clause != "" {
		query += " AND " + clause
		args = append(args, values...)
		position += len(values)
	}
	if clause, values := r.buildInClause(inClauseInput{ColumnName: "fu.plate_number", Values: filter.PlateNumber, Position: position}); clause != "" {
		query += " AND " + clause
		args = append(args, values...)
		position += len(values)
	}
	if clause, values := r.buildInClause(inClauseInput{ColumnName: "ft.label", Values: filter.FleetType, Position: position}); clause != "" {
		query += " AND " + clause
		args = append(args, values...)
		position += len(values)
	}
	if clause, values := r.buildInClause(inClauseInput{ColumnName: "fu.engine", Values: filter.Engine, Position: position}); clause != "" {
		query += " AND " + clause
		args = append(args, values...)
		position += len(values)
	}
	if clause, values := r.buildInClause(inClauseInput{ColumnName: capacityExpr, Values: filter.Capacity, Position: position}); clause != "" {
		query += " AND " + clause
		args = append(args, values...)
		position += len(values)
	}
	if clause, values := r.buildInClause(inClauseInput{ColumnName: productionYearExpr, Values: filter.ProductionYear, Position: position}); clause != "" {
		query += " AND " + clause
		args = append(args, values...)
	}

	query += " ORDER BY fo.start_date ASC, s.departure_time ASC"

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]model.ScheduleFleetAvailabilityRow, 0)
	for rows.Next() {
		var item model.ScheduleFleetAvailabilityRow
		if err := rows.Scan(
			&item.ScheduleID,
			&item.FleetType,
			&item.FleetName,
			&item.DepartureTime,
			&item.ArrivalTime,
			&item.StartDate,
			&item.EndDate,
			&item.VehicleID,
			&item.PlateNumber,
			&item.Engine,
			&item.Capacity,
			&item.ProductionYear,
			&item.Transmission,
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

func (r *ScheduleRepository) buildExactFilterClause(columnName, queryValue string, position int) (string, []interface{}) {
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

	return columnName + " = " + r.placeholder(position), []interface{}{value}
}

func (r *ScheduleRepository) buildInClause(input inClauseInput) (string, []interface{}) {
	if len(input.Values) == 0 {
		return "", nil
	}

	placeholders := make([]string, 0, len(input.Values))
	args := make([]interface{}, 0, len(input.Values))
	for _, value := range input.Values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		placeholders = append(placeholders, r.placeholder(input.Position+len(args)))
		args = append(args, trimmed)
	}
	if len(args) == 0 {
		return "", nil
	}
	return input.ColumnName + " IN (" + strings.Join(placeholders, ",") + ")", args
}

func (r *ScheduleRepository) ListScheduleFleets(scheduleID, organizationID string) ([]model.ScheduleFleetListItem, error) {
	scheduleExpr := "sf.schedule_id = " + r.placeholder(1)
	orgExpr := "sf.organization_id = " + r.placeholder(2)
	fleetIDExpr := "COALESCE(CAST(f.uuid AS CHAR), '')"
	if r.driver == "postgres" || r.driver == "pgx" {
		scheduleExpr = "sf.schedule_id::text = " + r.placeholder(1)
		orgExpr = "sf.organization_id::text = " + r.placeholder(2)
		fleetIDExpr = "COALESCE(f.uuid::text, '')"
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

	result := make([]model.ScheduleFleetListItem, 0)
	for rows.Next() {
		var item model.ScheduleFleetListItem
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

func (r *ScheduleRepository) LatestScheduleIDByOrderID(organizationID, orderID string) (string, bool, error) {
	orderExpr := "order_id = " + r.placeholder(1)
	orgExpr := "organization_id = " + r.placeholder(2)
	scheduleIDExpr := "schedule_id"
	if r.driver == "postgres" || r.driver == "pgx" {
		orderExpr = "order_id::text = " + r.placeholder(1)
		orgExpr = "organization_id::text = " + r.placeholder(2)
		scheduleIDExpr = "schedule_id::text"
	}

	query := `
		SELECT ` + scheduleIDExpr + `
		FROM schedules
		WHERE ` + orderExpr + ` AND ` + orgExpr + `
		ORDER BY created_at DESC
		LIMIT 1
	`

	var scheduleID string
	if err := database.QueryRow(r.db, query, orderID, organizationID).Scan(&scheduleID); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	return scheduleID, true, nil
}

func (r *ScheduleRepository) GetScheduleDetailRows(scheduleID, organizationID, orderID string) ([]model.ScheduleDetailRow, error) {
	scheduleExpr := "s.schedule_id = " + r.placeholder(1)
	orgExpr := "s.organization_id = " + r.placeholder(2)
	orderExpr := "s.order_id = " + r.placeholder(3)
	scheduleIDExpr := "COALESCE(CAST(s.schedule_id AS CHAR), '')"
	orderIDExpr := "COALESCE(CAST(s.order_id AS CHAR), '')"
	departureExpr := "COALESCE(CAST(s.departure_time AS CHAR), '')"
	arrivalExpr := "COALESCE(CAST(s.arrival_time AS CHAR), '')"
	fleetIDExpr := "COALESCE(CAST(sf.fleet_id AS CHAR), '')"
	unitIDExpr := "COALESCE(CAST(sf.unit_id AS CHAR), '')"
	driverIDExpr := "COALESCE(CAST(e.employee_id AS CHAR), '')"
	if r.driver == "postgres" || r.driver == "pgx" {
		scheduleExpr = "s.schedule_id::text = " + r.placeholder(1)
		orgExpr = "s.organization_id::text = " + r.placeholder(2)
		orderExpr = "s.order_id::text = " + r.placeholder(3)
		scheduleIDExpr = "COALESCE(s.schedule_id::text, '')"
		orderIDExpr = "COALESCE(s.order_id::text, '')"
		departureExpr = "COALESCE(s.departure_time::text, '')"
		arrivalExpr = "COALESCE(s.arrival_time::text, '')"
		fleetIDExpr = "COALESCE(sf.fleet_id::text, '')"
		unitIDExpr = "COALESCE(sf.unit_id::text, '')"
		driverIDExpr = "COALESCE(e.employee_id::text, '')"
	}

	query := `
		SELECT
			` + scheduleIDExpr + ` AS schedule_id,
			` + orderIDExpr + ` AS order_id,
			COALESCE(s.order_type, 0) AS order_type,
			` + departureExpr + ` AS departure_time,
			` + arrivalExpr + ` AS arrival_time,
			COALESCE(s.status, 0) AS status,
			` + fleetIDExpr + ` AS fleet_id,
			COALESCE(f.fleet_name, '') AS fleet_name,
			COALESCE(ft.label, '') AS fleet_type,
			` + unitIDExpr + ` AS unit_id,
			COALESCE(fu.vehicle_id, '') AS vehicle_id,
			COALESCE(fu.plate_number, '') AS plate_number,
			` + driverIDExpr + ` AS driver_id,
			COALESCE(e.fullname, '') AS fullname,
			COALESCE(orole.role_name, '') AS role_name
		FROM schedules s
		INNER JOIN schedule_fleets sf ON sf.schedule_id = s.schedule_id AND sf.organization_id = s.organization_id
		INNER JOIN fleets f ON f.uuid = sf.fleet_id
		INNER JOIN fleet_units fu ON fu.unit_id = sf.unit_id
		INNER JOIN fleet_types ft ON f.fleet_type = ft.id
		INNER JOIN schedule_fleet_teams sft ON sft.schedule_id = s.schedule_id AND sft.unit_id = sf.unit_id AND sft.organization_id = s.organization_id
		INNER JOIN employee e ON sft.employee_id = e.uuid
		INNER JOIN organization_roles orole ON orole.role_id = e.role_id
		WHERE ` + orgExpr + ` AND ` + orderExpr + ` AND ` + scheduleExpr + `
	`

	rows, err := database.Query(r.db, query, scheduleID, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]model.ScheduleDetailRow, 0)
	for rows.Next() {
		var item model.ScheduleDetailRow
		if err := rows.Scan(
			&item.ScheduleID,
			&item.OrderID,
			&item.OrderType,
			&item.DepartureTime,
			&item.ArrivalTime,
			&item.Status,
			&item.FleetID,
			&item.FleetName,
			&item.FleetType,
			&item.UnitID,
			&item.VehicleID,
			&item.PlateNumber,
			&item.DriverID,
			&item.Fullname,
			&item.RoleName,
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
