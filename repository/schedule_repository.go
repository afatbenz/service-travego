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
		INSERT INTO schedules (schedule_id, order_id, organization_id, departure_start, status, created_at, created_by, order_type)
		VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `, ` + r.placeholder(4) + `, 1, ` + r.placeholder(5) + `, ` + r.placeholder(6) + `, 1)
	`
	if _, err = database.TxExec(tx, insertSchedule, scheduleID, input.OrderID, input.OrganizationID, input.DepartureStart, input.CreatedAt, input.UserID); err != nil {
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
		if _, err = database.TxExec(tx, insertScheduleFleet, scheduleFleetID, scheduleID, input.OrderID, fleet.FleetID, fleet.UnitID, input.DepartureStart, input.CreatedAt, input.UserID, input.OrganizationID); err != nil {
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
