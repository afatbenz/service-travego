package repository

import (
	"database/sql"
	"service-travego/database"
	"service-travego/model"
	"service-travego/utils"
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
	orderExpr := "order_id::text = " + r.placeholder(2)
	orgExpr := "organization_id::text = " + r.placeholder(1)
	fleetExpr := "fleet_id::text = " + r.placeholder(3)

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
		SELECT schedule_id::text
		FROM schedules
		WHERE order_id::text = ` + r.placeholder(1) + ` AND organization_id::text = ` + r.placeholder(2) + `
		ORDER BY created_at DESC
		LIMIT 1
	`

	if err = database.TxQueryRow(tx, selectLatestSchedule, input.OrderID, input.OrganizationID).Scan(&scheduleID); err != nil {
		return "", err
	}

	var orgCode string
	orgQuery := "SELECT organization_code FROM organizations WHERE organization_id = " + r.placeholder(1)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgQuery = "SELECT organization_code FROM organizations WHERE organization_id::text = " + r.placeholder(1)
	}
	if err = database.TxQueryRow(tx, orgQuery, input.OrganizationID).Scan(&orgCode); err != nil {
		return "", err
	}

	var count int
	countQuery := "SELECT COUNT(schedule_number) FROM schedule_fleets WHERE organization_id = " + r.placeholder(1)
	if r.driver == "postgres" || r.driver == "pgx" {
		countQuery = "SELECT COUNT(schedule_number) FROM schedule_fleets WHERE organization_id::text = " + r.placeholder(1)
	}
	if err = database.TxQueryRow(tx, countQuery, input.OrganizationID).Scan(&count); err != nil {
		return "", err
	}

	scheduleFleetIDByUnit := map[string]string{}
	for _, fleet := range input.Fleets {
		count++
		scheduleFleetID := uuid.New().String()
		tripID := utils.GenerateTripID(orgCode, count, input.CreatedAt)
		insertScheduleFleet := `
			INSERT INTO schedule_fleets (uuid, schedule_id, order_id, fleet_id, unit_id, departure_time, created_at, created_by, status, organization_id, schedule_number)
			VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `, ` + r.placeholder(4) + `, ` + r.placeholder(5) + `, ` + r.placeholder(6) + `, ` + r.placeholder(7) + `, ` + r.placeholder(8) + `, 1, ` + r.placeholder(9) + `, ` + r.placeholder(10) + `)
		`
		if _, err = database.TxExec(tx, insertScheduleFleet, scheduleFleetID, scheduleID, input.OrderID, fleet.FleetID, fleet.UnitID, input.DepartureTime, input.CreatedAt, input.UserID, input.OrganizationID, tripID); err != nil {
			return "", err
		}
		unitID := strings.TrimSpace(fleet.UnitID)
		if unitID != "" {
			scheduleFleetIDByUnit[unitID] = scheduleFleetID
		}
	}

	insertTeam := `
		INSERT INTO schedule_fleet_teams (uuid, schedule_id, unit_id, schedule_fleet_id, driver_id, crew_id, created_at, created_by, organization_id, status)
		VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `, ` + r.placeholder(4) + `, ` + r.placeholder(5) + `, ` + r.placeholder(6) + `, ` + r.placeholder(7) + `, ` + r.placeholder(8) + `, ` + r.placeholder(9) + `, 1)
	`
	for _, team := range input.Teams {
		unitID := strings.TrimSpace(team.UnitID)
		driverID := strings.TrimSpace(team.DriverID)
		crewID := strings.TrimSpace(team.CrewID)
		scheduleFleetID := scheduleFleetIDByUnit[unitID]
		if unitID == "" || scheduleFleetID == "" || driverID == "" {
			continue
		}
		var crewArg interface{}
		if crewID != "" {
			crewArg = crewID
		}
		if _, err = database.TxExec(tx, insertTeam, uuid.New().String(), scheduleID, unitID, scheduleFleetID, driverID, crewArg, input.CreatedAt, input.UserID, input.OrganizationID); err != nil {
			return "", err
		}
	}

	orderExpr := "order_id::text = " + r.placeholder(1)
	orgExpr := "organization_id::text = " + r.placeholder(2)

	selectEndDate := `
		SELECT end_date
		FROM fleet_orders
		WHERE ` + orderExpr + ` AND ` + orgExpr + `
		LIMIT 1
	`

	var endDate sql.NullTime
	if err = database.TxQueryRow(tx, selectEndDate, input.OrderID, input.OrganizationID).Scan(&endDate); err != nil {
		return "", err
	}
	scheduleEndDate := input.DepartureTime
	if endDate.Valid {
		scheduleEndDate = endDate.Time
	}

	insertScheduleTeams := `
		INSERT INTO schedule_teams (schedule_team_id, employee_id, order_id, order_type, start_date, end_date, created_at, created_by, organization_id, status)
		VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `, 1, ` + r.placeholder(4) + `, ` + r.placeholder(5) + `, ` + r.placeholder(6) + `, ` + r.placeholder(7) + `, ` + r.placeholder(8) + `, 1)
	`
	employees := map[string]struct{}{}
	for _, t := range input.Teams {
		driverID := strings.TrimSpace(t.DriverID)
		if driverID != "" {
			employees[driverID] = struct{}{}
		}
		crewID := strings.TrimSpace(t.CrewID)
		if crewID != "" {
			employees[crewID] = struct{}{}
		}
	}
	for employeeID := range employees {
		if _, err = database.TxExec(tx, insertScheduleTeams, uuid.New().String(), employeeID, input.OrderID, input.DepartureTime, scheduleEndDate, input.CreatedAt, input.UserID, input.OrganizationID); err != nil {
			return "", err
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

	var orgCode string
	orgQuery := "SELECT organization_code FROM organizations WHERE organization_id = " + r.placeholder(1)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgQuery = "SELECT organization_code FROM organizations WHERE organization_id::text = " + r.placeholder(1)
	}
	if err = database.TxQueryRow(tx, orgQuery, input.OrganizationID).Scan(&orgCode); err != nil {
		return err
	}

	var count int
	countQuery := "SELECT COUNT(schedule_number) FROM schedule_fleets WHERE organization_id = " + r.placeholder(1)
	if r.driver == "postgres" || r.driver == "pgx" {
		countQuery = "SELECT COUNT(schedule_number) FROM schedule_fleets WHERE organization_id::text = " + r.placeholder(1)
	}
	if err = database.TxQueryRow(tx, countQuery, input.OrganizationID).Scan(&count); err != nil {
		return err
	}

	scheduleFleetIDByUnit := map[string]string{}
	for _, fleet := range input.Fleets {
		unitID := strings.TrimSpace(fleet.UnitID)
		if unitID == "" {
			continue
		}

		scheduleFleetID := existingByUnit[unitID]
		if scheduleFleetID == "" {
			count++
			scheduleFleetID = uuid.New().String()
			tripID := utils.GenerateTripID(orgCode, count, input.UpdatedAt)
			insertScheduleFleet := `
				INSERT INTO schedule_fleets (uuid, schedule_id, order_id, fleet_id, unit_id, departure_time, created_at, created_by, status, organization_id, schedule_number)
				VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `, ` + r.placeholder(4) + `, ` + r.placeholder(5) + `, ` + r.placeholder(6) + `, ` + r.placeholder(7) + `, ` + r.placeholder(8) + `, 1, ` + r.placeholder(9) + `, ` + r.placeholder(10) + `)
			`
			if _, err = database.TxExec(tx, insertScheduleFleet, scheduleFleetID, input.ScheduleID, input.OrderID, fleet.FleetID, unitID, input.DepartureTime, input.UpdatedAt, input.UserID, input.OrganizationID, tripID); err != nil {
				return err
			}
		} else {
			updateScheduleFleet := `
				UPDATE schedule_fleets
				SET fleet_id = ` + r.placeholder(3) + `,
					departure_time = ` + r.placeholder(4) + `
				WHERE uuid = ` + r.placeholder(1) + ` AND organization_id = ` + r.placeholder(2) + `
			`
			if r.driver == "postgres" || r.driver == "pgx" {
				updateScheduleFleet = `
					UPDATE schedule_fleets
					SET fleet_id = ` + r.placeholder(3) + `,
						departure_time = ` + r.placeholder(4) + `
					WHERE uuid::text = ` + r.placeholder(1) + ` AND organization_id::text = ` + r.placeholder(2) + `
				`
			}
			if _, err = database.TxExec(tx, updateScheduleFleet, scheduleFleetID, input.OrganizationID, fleet.FleetID, input.DepartureTime); err != nil {
				return err
			}
		}
		scheduleFleetIDByUnit[unitID] = scheduleFleetID
	}

	insertTeam := `
		INSERT INTO schedule_fleet_teams (uuid, schedule_id, unit_id, schedule_fleet_id, driver_id, crew_id, created_at, created_by, organization_id, status)
		VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `, ` + r.placeholder(4) + `, ` + r.placeholder(5) + `, ` + r.placeholder(6) + `, ` + r.placeholder(7) + `, ` + r.placeholder(8) + `, ` + r.placeholder(9) + `, 1)
	`
	updateTeam := `
		UPDATE schedule_fleet_teams
		SET schedule_fleet_id = ` + r.placeholder(3) + `, unit_id = ` + r.placeholder(4) + `, driver_id = ` + r.placeholder(5) + `, crew_id = ` + r.placeholder(6) + `, updated_at = ` + r.placeholder(7) + `, updated_by = ` + r.placeholder(8) + `
		WHERE uuid = ` + r.placeholder(1) + ` AND organization_id = ` + r.placeholder(2) + `
	`
	if r.driver == "postgres" || r.driver == "pgx" {
		updateTeam = `
			UPDATE schedule_fleet_teams
			SET schedule_fleet_id = ` + r.placeholder(3) + `, unit_id = ` + r.placeholder(4) + `, driver_id = ` + r.placeholder(5) + `, crew_id = ` + r.placeholder(6) + `, updated_at = ` + r.placeholder(7) + `, updated_by = ` + r.placeholder(8) + `
			WHERE uuid::text = ` + r.placeholder(1) + ` AND organization_id::text = ` + r.placeholder(2) + `
		`
	}

	for _, team := range input.Teams {
		unitID := strings.TrimSpace(team.UnitID)
		driverID := strings.TrimSpace(team.DriverID)
		crewID := strings.TrimSpace(team.CrewID)
		uuidText := strings.TrimSpace(team.UUID)
		scheduleFleetID := scheduleFleetIDByUnit[unitID]

		if unitID == "" || scheduleFleetID == "" || driverID == "" {
			continue
		}

		var crewArg interface{}
		if crewID != "" {
			crewArg = crewID
		}

		if uuidText == "" {
			if _, err = database.TxExec(tx, insertTeam, uuid.New().String(), input.ScheduleID, unitID, scheduleFleetID, driverID, crewArg, input.UpdatedAt, input.UserID, input.OrganizationID); err != nil {
				return err
			}
			continue
		}

		if _, err = database.TxExec(tx, updateTeam, uuidText, input.OrganizationID, scheduleFleetID, unitID, driverID, crewArg, input.UpdatedAt, input.UserID); err != nil {
			return err
		}
	}

	orderExprTeams := "order_id = " + r.placeholder(1)
	orgExprTeams := "organization_id = " + r.placeholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		orderExprTeams = "order_id::text = " + r.placeholder(1)
		orgExprTeams = "organization_id::text = " + r.placeholder(2)
	}
	deleteScheduleTeams := `
		DELETE FROM schedule_teams
		WHERE ` + orderExprTeams + ` AND ` + orgExprTeams + ` AND order_type = 1
	`
	if _, err = database.TxExec(tx, deleteScheduleTeams, input.OrderID, input.OrganizationID); err != nil {
		return err
	}

	selectEndDate := `
		SELECT end_date
		FROM fleet_orders
		WHERE ` + orderExprTeams + ` AND ` + orgExprTeams + `
		LIMIT 1
	`
	var endDate sql.NullTime
	if err = database.TxQueryRow(tx, selectEndDate, input.OrderID, input.OrganizationID).Scan(&endDate); err != nil {
		return err
	}
	scheduleEndDate := input.DepartureTime
	if endDate.Valid {
		scheduleEndDate = endDate.Time
	}

	insertScheduleTeams := `
		INSERT INTO schedule_teams (schedule_team_id, employee_id, order_id, order_type, start_date, end_date, created_at, created_by, organization_id, status)
		VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `, 1, ` + r.placeholder(4) + `, ` + r.placeholder(5) + `, ` + r.placeholder(6) + `, ` + r.placeholder(7) + `, ` + r.placeholder(8) + `, 1)
	`
	employees := map[string]struct{}{}
	for _, t := range input.Teams {
		driverID := strings.TrimSpace(t.DriverID)
		if driverID != "" {
			employees[driverID] = struct{}{}
		}
		crewID := strings.TrimSpace(t.CrewID)
		if crewID != "" {
			employees[crewID] = struct{}{}
		}
	}
	for employeeID := range employees {
		if _, err = database.TxExec(tx, insertScheduleTeams, uuid.New().String(), employeeID, input.OrderID, input.DepartureTime, scheduleEndDate, input.UpdatedAt, input.UserID, input.OrganizationID); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *ScheduleRepository) ListScheduleFleetOrders(input model.ScheduleFleetListQuery, organizationID string, monthStart, monthEnd time.Time) ([]model.ScheduleFleetOrderRow, error) {
	orgExpr := "s.organization_id::text = " + r.placeholder(1)
	departureExpr := "COALESCE(s.departure_time::text, '')"
	arrivalExpr := "COALESCE(s.arrival_time::text, '')"
	orderIDExpr := "COALESCE(fo.order_id::text, '')"
	pickupCityExpr := "COALESCE(fo.pickup_city_id::text, '')"
	createdByExpr := "COALESCE(s.created_by::text, '')"

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
			STRING_AGG(DISTINCT foi.city_id::text, ', ') AS destination_ids,
			` + createdByExpr + ` AS created_by
		FROM schedules s
		INNER JOIN fleet_orders fo ON s.order_id = fo.order_id
		INNER JOIN fleet_order_itinerary foi ON s.order_id = foi.order_id
		WHERE ` + orgExpr + ` AND s.order_type = 1
	`

	args := []interface{}{organizationID}
	position := 2

	query += `
		AND (
			(fo.start_date::date >= ` + r.placeholder(position) + ` AND fo.start_date::date <= ` + r.placeholder(position+1) + `)
			OR
			(fo.end_date::date >= ` + r.placeholder(position) + ` AND fo.end_date::date <= ` + r.placeholder(position+1) + `)
		)
	`
	args = append(args, monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02"))
	position += 2

	fleetFilters := make([]string, 0, 6)
	capacityExpr := "u.capacity::text"
	productionYearExpr := "u.production_year::text"
	scheduleFleetOrderIDExpr := "sf.order_id::text"
	scheduleFleetIDExpr := "sf.fleet_id::text"
	scheduleFleetUnitIDExpr := "sf.unit_id::text"

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
	query += `
		GROUP BY
			s.schedule_id,
			fo.order_id,
			fo.start_date,
			fo.end_date,
			s.departure_time,
			s.arrival_time,
			s.status,
			fo.unit_qty,
			fo.pickup_city_id,
			fo.additional_request,
			fo.payment_status,
			s.created_at,
			s.created_by
		ORDER BY fo.start_date ASC, s.created_at DESC
	`

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
		var destinationIDs sql.NullString
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
			&destinationIDs,
			&createdBy,
		); err != nil {
			return nil, err
		}

		item.DepartureTime = departureTime.String
		item.ArrivalTime = arrivalTime.String
		item.DestinationIDs = destinationIDs.String
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

func (r *ScheduleRepository) ListScheduleDetailsByDate(selectedDate time.Time, organizationID string) ([]model.ScheduleDetailByDateRow, error) {
	orgExpr := "s.organization_id::text = " + r.placeholder(1)
	scheduleIDExpr := "COALESCE(s.schedule_id::text, '')"
	orderIDExpr := "COALESCE(s.order_id::text, '')"
	cityAggExpr := "COALESCE(ARRAY_AGG(DISTINCT foi.city_id)::text, '{}')"

	query := `
		SELECT
			` + scheduleIDExpr + ` AS schedule_id,
			` + orderIDExpr + ` AS order_id,
			COALESCE(f.fleet_name, '') AS fleet_name,
			COALESCE(fu.vehicle_id, '') AS vehicle_id,
			COALESCE(fu.plate_number, '') AS plate_number,
			COALESCE(e1.fullname, '') AS driver_name,
			fo.start_date,
			fo.end_date,
			` + cityAggExpr + ` AS city_ids
		FROM schedules s
		INNER JOIN schedule_fleets sf ON s.schedule_id = sf.schedule_id AND sf.organization_id = s.organization_id
		INNER JOIN fleets f ON sf.fleet_id = f.uuid
		INNER JOIN schedule_fleet_teams sft ON sft.schedule_fleet_id = sf.uuid AND sft.unit_id = sf.unit_id AND sft.organization_id = s.organization_id
		INNER JOIN fleet_units fu ON fu.unit_id = sft.unit_id
		INNER JOIN fleet_orders fo ON fo.order_id = s.order_id
		INNER JOIN fleet_order_itinerary foi ON fo.order_id = foi.order_id
		INNER JOIN employee e1 ON e1.uuid = sft.driver_id
		WHERE ` + orgExpr + `
		  AND fo.start_date::date <= ` + r.placeholder(2) + `
		  AND fo.end_date::date >= ` + r.placeholder(3) + `
		GROUP BY s.schedule_id, s.order_id, f.fleet_name, fu.vehicle_id, fu.plate_number, fo.start_date, fo.end_date, e1.fullname
		ORDER BY fo.start_date ASC, s.schedule_id ASC
	`

	rows, err := database.Query(r.db, query, organizationID, selectedDate, selectedDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]model.ScheduleDetailByDateRow, 0)
	for rows.Next() {
		var item model.ScheduleDetailByDateRow
		if err := rows.Scan(
			&item.ScheduleID,
			&item.OrderID,
			&item.FleetName,
			&item.VehicleID,
			&item.PlateNumber,
			&item.DriverName,
			&item.StartDate,
			&item.EndDate,
			&item.CityIDsRaw,
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

func (r *ScheduleRepository) ListScheduleOperationAvailabilityEmployees(organizationID string, startDate, endDate time.Time, employeeID string) ([]model.ScheduleOperationAvailabilityRow, error) {
	uuidExpr := "COALESCE(CAST(e.uuid AS CHAR), '')"
	employeeIDExpr := "COALESCE(CAST(e.employee_id AS CHAR), '')"
	scheduleIDExpr := "''"
	orgEmployeeExpr := "e.organization_id = " + r.placeholder(1)
	orgScheduleExpr := "s.organization_id = " + r.placeholder(1)
	employeeExpr := "e.uuid = " + r.placeholder(4)
	if r.driver == "postgres" || r.driver == "pgx" {
		uuidExpr = "COALESCE(e.uuid::text, '')"
		employeeIDExpr = "COALESCE(e.employee_id::text, '')"
		scheduleIDExpr = "''"
		orgEmployeeExpr = "e.organization_id::text = " + r.placeholder(1)
		orgScheduleExpr = "s.organization_id::text = " + r.placeholder(1)
		employeeExpr = "e.uuid::text = " + r.placeholder(4)
	}

	employeeFilter := ""
	args := []interface{}{organizationID, endDate, startDate}
	if strings.TrimSpace(employeeID) != "" {
		employeeFilter = " AND " + employeeExpr
		args = append(args, strings.TrimSpace(employeeID))
	}

	query := `
		SELECT
			` + uuidExpr + ` AS uuid,
			` + employeeIDExpr + ` AS employee_id,
			COALESCE(e.fullname, '') AS fullname,
			COALESCE(e.phone, '') AS phone,
			` + scheduleIDExpr + ` AS schedule_id
		FROM employee e
		WHERE ` + orgEmployeeExpr + `
		  AND COALESCE(e.status, 0) > 0
		  ` + employeeFilter + `
		  AND NOT EXISTS (
			SELECT 1
			FROM schedule_fleet_teams st
			INNER JOIN schedules s ON s.schedule_id = st.schedule_id
			INNER JOIN fleet_orders fo ON fo.order_id = s.order_id
			WHERE ` + orgScheduleExpr + `
			  AND st.organization_id = s.organization_id
			  AND (st.driver_id = e.uuid OR st.crew_id = e.uuid)
			  AND fo.start_date <= ` + r.placeholder(2) + `
			  AND fo.end_date >= ` + r.placeholder(3) + `
		  )
		ORDER BY e.fullname ASC
	`

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]model.ScheduleOperationAvailabilityRow, 0)
	for rows.Next() {
		var item model.ScheduleOperationAvailabilityRow
		if err := rows.Scan(
			&item.UUID,
			&item.EmployeeID,
			&item.Fullname,
			&item.Phone,
			&item.ScheduleID,
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

func (r *ScheduleRepository) ListAvailableScheduleFleetUnits(organizationID string, startDate, endDate time.Time, fleetID string) ([]model.ScheduleFleetUnitAvailabilityRow, error) {
	orgFleetExpr := "fu.organization_id = " + r.placeholder(1)
	orgFleetJoinExpr := "f.organization_id = " + r.placeholder(1)
	orgScheduleExpr := "sf.organization_id = " + r.placeholder(1)
	orderJoinExpr := "fo.order_id = sf.order_id AND fo.organization_id = sf.organization_id"
	unitJoinExpr := "sf.unit_id = fu.unit_id"
	fleetJoinExpr := "f.uuid = fu.fleet_id"
	unitIDExpr := "COALESCE(CAST(fu.unit_id AS CHAR), '')"
	fleetIDExpr := "COALESCE(CAST(f.uuid AS CHAR), '')"
	vehicleIDExpr := "COALESCE(CAST(fu.vehicle_id AS CHAR), '')"
	fleetFilterExpr := "fu.fleet_id = " + r.placeholder(4)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgFleetExpr = "fu.organization_id::text = " + r.placeholder(1)
		orgFleetJoinExpr = "f.organization_id::text = " + r.placeholder(1)
		orgScheduleExpr = "sf.organization_id::text = " + r.placeholder(1)
		orderJoinExpr = "fo.order_id::text = sf.order_id::text AND fo.organization_id::text = sf.organization_id::text"
		unitJoinExpr = "sf.unit_id::text = fu.unit_id::text"
		fleetJoinExpr = "f.uuid::text = fu.fleet_id::text"
		unitIDExpr = "COALESCE(fu.unit_id::text, '')"
		fleetIDExpr = "COALESCE(f.uuid::text, '')"
		vehicleIDExpr = "COALESCE(fu.vehicle_id::text, '')"
		fleetFilterExpr = "fu.fleet_id::text = " + r.placeholder(4)
	}

	query := `
		SELECT
			` + unitIDExpr + ` AS unit_id,
			` + fleetIDExpr + ` AS fleet_id,
			COALESCE(f.fleet_name, '') AS fleet_name,
			` + vehicleIDExpr + ` AS vehicle_id,
			COALESCE(fu.plate_number, '') AS plate_number
		FROM fleet_units fu
		INNER JOIN fleets f ON ` + fleetJoinExpr + `
		WHERE ` + orgFleetExpr + `
		  AND ` + orgFleetJoinExpr + `
		  AND ` + fleetFilterExpr + `
		  AND NOT EXISTS (
			SELECT 1
			FROM schedule_fleets sf
			INNER JOIN fleet_orders fo ON ` + orderJoinExpr + `
			WHERE ` + orgScheduleExpr + `
			  AND ` + unitJoinExpr + `
			  AND COALESCE(sf.status, 0) = 1
			  AND fo.start_date <= ` + r.placeholder(2) + `
			  AND fo.end_date >= ` + r.placeholder(3) + `
		  )
		ORDER BY f.fleet_name ASC, fu.created_at ASC
	`

	rows, err := database.Query(r.db, query, organizationID, endDate, startDate, strings.TrimSpace(fleetID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]model.ScheduleFleetUnitAvailabilityRow, 0)
	for rows.Next() {
		var item model.ScheduleFleetUnitAvailabilityRow
		if err := rows.Scan(
			&item.UnitID,
			&item.FleetID,
			&item.FleetName,
			&item.VehicleID,
			&item.PlateNumber,
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

func (r *ScheduleRepository) GetFleetWithUnitsForDailyAvailability(organizationID, fleetID string) (string, []model.DailyAvailabilityFleetUnitRow, bool, error) {
	orgExpr := "f.organization_id = " + r.placeholder(1)
	fleetExpr := "f.uuid = " + r.placeholder(2)
	unitOrgJoinExpr := "fu.organization_id = f.organization_id"
	unitFleetJoinExpr := "fu.fleet_id = f.uuid"
	unitIDExpr := "COALESCE(CAST(fu.unit_id AS CHAR), '')"
	vehicleIDExpr := "COALESCE(CAST(fu.vehicle_id AS CHAR), '')"
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "f.organization_id::text = " + r.placeholder(1)
		fleetExpr = "f.uuid::text = " + r.placeholder(2)
		unitOrgJoinExpr = "fu.organization_id::text = f.organization_id::text"
		unitFleetJoinExpr = "fu.fleet_id::text = f.uuid::text"
		unitIDExpr = "COALESCE(fu.unit_id::text, '')"
		vehicleIDExpr = "COALESCE(fu.vehicle_id::text, '')"
	}

	query := `
		SELECT
			COALESCE(f.fleet_name, '') AS fleet_name,
			` + unitIDExpr + ` AS unit_id,
			` + vehicleIDExpr + ` AS vehicle_id,
			COALESCE(fu.plate_number, '') AS plate_number
		FROM fleets f
		LEFT JOIN fleet_units fu ON ` + unitFleetJoinExpr + ` AND ` + unitOrgJoinExpr + `
		WHERE ` + orgExpr + `
		  AND ` + fleetExpr + `
		ORDER BY fu.created_at ASC
	`

	rows, err := database.Query(r.db, query, organizationID, strings.TrimSpace(fleetID))
	if err != nil {
		return "", nil, false, err
	}
	defer rows.Close()

	var fleetName string
	units := make([]model.DailyAvailabilityFleetUnitRow, 0)

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return "", nil, false, err
		}
		return "", nil, false, nil
	}

	for {
		var unitID, vehicleID, plateNumber string
		if err := rows.Scan(&fleetName, &unitID, &vehicleID, &plateNumber); err != nil {
			return "", nil, false, err
		}
		if strings.TrimSpace(unitID) != "" {
			units = append(units, model.DailyAvailabilityFleetUnitRow{
				UnitID:      unitID,
				VehicleID:   vehicleID,
				PlateNumber: plateNumber,
			})
		}
		if !rows.Next() {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return "", nil, false, err
	}

	return fleetName, units, true, nil
}

func (r *ScheduleRepository) ListScheduledFleetUnitDaysForDailyAvailability(organizationID string, startDate, endDate time.Time, fleetID string) ([]model.DailyAvailabilityFleetScheduledUnitDayRow, error) {
	if r.driver == "postgres" || r.driver == "pgx" {
		query := `
			SELECT DISTINCT
				gs.day::date AS day,
				COALESCE(sf.unit_id::text, '') AS unit_id
			FROM schedule_fleets sf
			INNER JOIN fleet_orders fo ON fo.order_id::text = sf.order_id::text AND fo.organization_id::text = sf.organization_id::text
			INNER JOIN fleet_units fu ON fu.unit_id::text = sf.unit_id::text AND fu.organization_id::text = sf.organization_id::text
			CROSS JOIN LATERAL generate_series(date_trunc('day', fo.start_date), date_trunc('day', fo.end_date), interval '1 day') gs(day)
			WHERE sf.organization_id::text = ` + r.placeholder(1) + `
			  AND fu.fleet_id::text = ` + r.placeholder(4) + `
			  AND COALESCE(sf.status, 0) = 1
			  AND gs.day::date >= ` + r.placeholder(2) + `::date
			  AND gs.day::date <= ` + r.placeholder(3) + `::date
		`

		rows, err := database.Query(r.db, query, organizationID, startDate, endDate, strings.TrimSpace(fleetID))
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		result := make([]model.DailyAvailabilityFleetScheduledUnitDayRow, 0)
		for rows.Next() {
			var item model.DailyAvailabilityFleetScheduledUnitDayRow
			if err := rows.Scan(&item.Day, &item.UnitID); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return result, nil
	}

	orgExpr := "sf.organization_id = " + r.placeholder(1)
	fleetFilterExpr := "fu.fleet_id = " + r.placeholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "sf.organization_id::text = " + r.placeholder(1)
		fleetFilterExpr = "fu.fleet_id::text = " + r.placeholder(2)
	}

	query := `
		SELECT DISTINCT
			COALESCE(CAST(sf.unit_id AS CHAR), '') AS unit_id
		FROM schedule_fleets sf
		INNER JOIN fleet_orders fo ON fo.order_id = sf.order_id AND fo.organization_id = sf.organization_id
		INNER JOIN fleet_units fu ON fu.unit_id = sf.unit_id AND fu.organization_id = sf.organization_id
		WHERE ` + orgExpr + `
		  AND ` + fleetFilterExpr + `
		  AND COALESCE(sf.status, 0) = 1
		  AND fo.start_date <= ` + r.placeholder(3) + `
		  AND fo.end_date >= ` + r.placeholder(4) + `
	`

	days := make(map[string]map[string]struct{})
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		rows, err := database.Query(r.db, query, organizationID, strings.TrimSpace(fleetID), d, d)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var unitID string
			if err := rows.Scan(&unitID); err != nil {
				rows.Close()
				return nil, err
			}
			dateKey := d.Format("2006-01-02")
			if _, ok := days[dateKey]; !ok {
				days[dateKey] = make(map[string]struct{})
			}
			if strings.TrimSpace(unitID) != "" {
				days[dateKey][unitID] = struct{}{}
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}

	result := make([]model.DailyAvailabilityFleetScheduledUnitDayRow, 0)
	for dateKey, unitSet := range days {
		day, _ := time.Parse("2006-01-02", dateKey)
		for unitID := range unitSet {
			result = append(result, model.DailyAvailabilityFleetScheduledUnitDayRow{
				Day:    day,
				UnitID: unitID,
			})
		}
	}

	return result, nil
}

func (r *ScheduleRepository) GetFleetUnitForDailyAvailability(organizationID, unitID string) (model.DailyAvailabilityFleetUnitRow, bool, error) {
	orgExpr := "organization_id::text = " + r.placeholder(1)
	unitExpr := "unit_id::text = " + r.placeholder(2)
	unitIDExpr := "COALESCE(unit_id::text, '')"
	vehicleIDExpr := "COALESCE(vehicle_id::text, '')"

	query := `
		SELECT
			` + unitIDExpr + ` AS unit_id,
			` + vehicleIDExpr + ` AS vehicle_id,
			COALESCE(plate_number, '') AS plate_number
		FROM fleet_units
		WHERE ` + orgExpr + ` AND ` + unitExpr + `
		LIMIT 1
	`

	var row model.DailyAvailabilityFleetUnitRow
	if err := database.QueryRow(r.db, query, organizationID, strings.TrimSpace(unitID)).Scan(&row.UnitID, &row.VehicleID, &row.PlateNumber); err != nil {
		if err == sql.ErrNoRows {
			return model.DailyAvailabilityFleetUnitRow{}, false, nil
		}
		return model.DailyAvailabilityFleetUnitRow{}, false, err
	}
	return row, true, nil
}

func (r *ScheduleRepository) ListScheduledUnitDaysForDailyAvailability(organizationID string, startDate, endDate time.Time, unitID string) ([]model.DailyAvailabilityFleetUnitScheduledDayRow, error) {
	query := `
			SELECT
				gs.day::date AS day,
				COALESCE(sf.order_id::text, '-') AS order_id,
				COALESCE(STRING_AGG(DISTINCT foi.city_id::text, ','), '') AS destination_ids
			FROM schedule_fleets sf
			INNER JOIN fleet_orders fo ON fo.order_id::text = sf.order_id::text AND fo.organization_id::text = sf.organization_id::text
			LEFT JOIN fleet_order_itinerary foi ON foi.order_id::text = sf.order_id::text
			CROSS JOIN LATERAL generate_series(date_trunc('day', fo.start_date), date_trunc('day', fo.end_date), interval '1 day') gs(day)
			WHERE sf.organization_id::text = ` + r.placeholder(1) + `
			  AND sf.unit_id::text = ` + r.placeholder(4) + `
			  AND COALESCE(sf.status, 0) = 1
			  AND gs.day::date >= ` + r.placeholder(2) + `::date
			  AND gs.day::date <= ` + r.placeholder(3) + `::date
			GROUP BY gs.day::date, sf.order_id::text
		`

	rows, err := database.Query(r.db, query, organizationID, startDate, endDate, strings.TrimSpace(unitID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]model.DailyAvailabilityFleetUnitScheduledDayRow, 0)
	for rows.Next() {
		var item model.DailyAvailabilityFleetUnitScheduledDayRow
		if err := rows.Scan(&item.Day, &item.OrderID, &item.DestinationIDs); err != nil {
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

	scheduleExpr := "sf.schedule_id::text = " + r.placeholder(1)
	orgExpr := "sf.organization_id::text = " + r.placeholder(2)
	fleetIDExpr := "COALESCE(f.uuid::text, '')"

	query := `
		SELECT
			` + fleetIDExpr + ` AS fleet_id,
			COALESCE(f.fleet_name, '') AS fleet_name,
			COALESCE(u.vehicle_id, '') AS vehicle_id,
			COALESCE(u.plate_number, '') AS plate_number,
			COALESCE(u.engine, '') AS engine,
			COALESCE(u.capacity, 0) AS capacity,
			COALESCE(e.fullname, '') AS driver_name,
			COALESCE(e2.fullname, '') AS crew_name,
			COALESCE(sf.schedule_number, '') AS schedule_number,
			STRING_AGG(DISTINCT foi.city_id::text, ', ') AS destination_ids
		FROM schedule_fleets sf
		INNER JOIN schedules s ON s.schedule_id = sf.schedule_id
		INNER JOIN fleet_units u ON sf.unit_id = u.unit_id
		INNER JOIN fleets f ON u.fleet_id = f.uuid
		INNER JOIN schedule_fleet_teams sft ON sft.schedule_fleet_id = sf.uuid
		INNER JOIN employee e ON sft.driver_id = e.uuid
		INNER JOIN fleet_order_itinerary foi ON foi.order_id = sf.order_id
		LEFT JOIN employee e2 ON sft.crew_id = e2.uuid
		WHERE ` + scheduleExpr + ` AND ` + orgExpr + ` AND s.status = 1
		GROUP BY f.uuid, f.fleet_name, u.vehicle_id, u.plate_number, u.engine, u.capacity, e.fullname, e2.fullname, sf.schedule_number
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
			&item.DriverName,
			&item.CrewName,
			&item.ScheduleNumber,
			&item.DestinationIDs,
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
	orderExpr := "order_id::text = " + r.placeholder(1)
	orgExpr := "organization_id::text = " + r.placeholder(2)
	scheduleIDExpr := "schedule_id::text"

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
	crewIDExpr := "COALESCE(CAST(ecrew.employee_id AS CHAR), '')"
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
		crewIDExpr = "COALESCE(ecrew.employee_id::text, '')"
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
			` + crewIDExpr + ` AS crew_id,
			COALESCE(ecrew.fullname, '') AS crew_name,
			COALESCE(orole.role_name, '') AS role_name
		FROM schedules s
		INNER JOIN schedule_fleets sf ON sf.schedule_id = s.schedule_id AND sf.organization_id = s.organization_id
		INNER JOIN fleets f ON f.uuid = sf.fleet_id
		INNER JOIN fleet_units fu ON fu.unit_id = sf.unit_id
		INNER JOIN fleet_types ft ON f.fleet_type = ft.id
		INNER JOIN schedule_fleet_teams sft ON sft.schedule_fleet_id = sf.uuid AND sft.unit_id = sf.unit_id AND sft.organization_id = s.organization_id
		LEFT JOIN employee e ON sft.driver_id = e.uuid
		LEFT JOIN employee ecrew ON sft.crew_id = ecrew.uuid
		LEFT JOIN organization_roles orole ON orole.role_id = e.role_id
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
			&item.CrewID,
			&item.CrewName,
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

func (r *ScheduleRepository) GetFleetTripDetail(input model.ScheduleFleetTripDetailServiceInput) (*model.ScheduleFleetTripDetailResponse, bool, error) {
	scheduleNumber := strings.TrimSpace(input.ScheduleNumber)
	orgID := strings.TrimSpace(input.OrganizationID)

	scheduleNumberExpr := "sf.schedule_number = " + r.placeholder(1)
	orgExpr := "sf.organization_id::text = " + r.placeholder(2)

	query := `
			SELECT
				COALESCE(sf.uuid::text, '') AS schedule_fleet_id,
				COALESCE(s.schedule_id::text, '') AS schedule_id,
				COALESCE(s.order_id::text, '') AS order_id,
				COALESCE(s.departure_time::text, '') AS departure_time,
				COALESCE(s.arrival_time::text, '') AS arrival_time,
				COALESCE(f.fleet_name, '') AS fleet_name,
				COALESCE(f.thumbnail, '') AS fleet_photo,
				COALESCE(sf.unit_id::text, '') AS unit_id,
				COALESCE(fu.vehicle_id, '') AS vehicle_id,
				COALESCE(fu.plate_number, '') AS plate_number,
				COALESCE(fo.start_date::text, '') AS start_date,
				COALESCE(fo.end_date::text, '') AS end_date,
				COALESCE(fo.payment_status, 0) AS payment_status,
				COALESCE(e.fullname, '') AS driver_name,
				COALESCE(e.avatar, '') AS driver_avatar,
				COALESCE(e2.fullname, '') AS crew_name,
				COALESCE(e2.avatar, '') AS crew_avatar
			FROM schedule_fleets sf
			INNER JOIN schedules s ON s.schedule_id::text = sf.schedule_id::text AND s.organization_id::text = sf.organization_id::text
			INNER JOIN fleets f ON f.uuid::text = sf.fleet_id::text
			INNER JOIN fleet_units fu ON fu.unit_id::text = sf.unit_id::text
			INNER JOIN fleet_orders fo ON fo.order_id::text = s.order_id::text AND fo.organization_id::text = s.organization_id::text
			INNER JOIN schedule_fleet_teams sft ON sft.schedule_fleet_id::text = sf.uuid::text AND sft.organization_id::text = sf.organization_id::text
			INNER JOIN employee e ON sft.driver_id::text = e.uuid::text
			INNER JOIN employee e2 ON sft.crew_id::text = e2.uuid::text
			WHERE ` + scheduleNumberExpr + ` AND ` + orgExpr + `
		`
	var res model.ScheduleFleetTripDetailResponse
	if err := database.QueryRow(
		r.db,
		query,
		scheduleNumber,
		orgID,
	).Scan(
		&res.ScheduleFleetID,
		&res.ScheduleID,
		&res.OrderID,
		&res.DepartureTime,
		&res.ArrivalTime,
		&res.FleetName,
		&res.FleetPhoto,
		&res.UnitID,
		&res.VehicleID,
		&res.PlateNumber,
		&res.StartDate,
		&res.EndDate,
		&res.PaymentStatus,
		&res.DriverName,
		&res.DriverAvatar,
		&res.CrewName,
		&res.CrewAvatar,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &res, true, nil
}
