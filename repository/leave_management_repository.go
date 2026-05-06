package repository

import (
	"database/sql"
	"fmt"
	"service-travego/database"
	"service-travego/model"
	"time"
)

type LeaveManagementRepository struct {
	db     *sql.DB
	driver string
}

func NewLeaveManagementRepository(db *sql.DB, driver string) *LeaveManagementRepository {
	return &LeaveManagementRepository{db: db, driver: driver}
}

func (r *LeaveManagementRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

func (r *LeaveManagementRepository) ListLeaveTypes() ([]model.LeaveManagementTypeItem, error) {
	query := `
		SELECT id, label
		FROM employee_leave_type
		ORDER BY id ASC
	`
	rows, err := database.Query(r.db, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.LeaveManagementTypeItem, 0)
	for rows.Next() {
		var it model.LeaveManagementTypeItem
		if err := rows.Scan(&it.ID, &it.Label); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LeaveManagementRepository) ListEmployeeLeaves(organizationID string, start *time.Time, end *time.Time) ([]model.LeaveManagementListItem, error) {
	orgExpr := "e.organization_id = " + r.getPlaceholder(1)
	if r.driver != "mysql" {
		orgExpr = "e.organization_id::text = " + r.getPlaceholder(1)
	}

	dateFilter := ""
	args := []interface{}{organizationID}
	if start != nil && end != nil {
		dateFilter = fmt.Sprintf(`
			AND el.start_date <= %s
			AND COALESCE(el.end_date, el.start_date) >= %s
		`, r.getPlaceholder(2), r.getPlaceholder(3))
		args = append(args, *end, *start)
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(el.leave_id::text, ''),
			COALESCE(el.employee_id::text, ''),
			COALESCE(el.substituted_by::text, ''),
			el.start_date,
			el.end_date,
			COALESCE(el.leave_type, 0),
			COALESCE(lt.label, '')
		FROM employee_leaves el
		INNER JOIN employee_leave_type lt ON lt.id = el.leave_type
		INNER JOIN employee e ON e.uuid = el.employee_id
		INNER JOIN employee es ON es.uuid = el.substituted_by
		WHERE %s
		%s
		ORDER BY el.start_date DESC
	`, orgExpr, dateFilter)

	if r.driver == "mysql" {
		query = fmt.Sprintf(`
			SELECT
				COALESCE(el.leave_id, ''),
				COALESCE(el.employee_id, ''),
				COALESCE(el.substituted_by, ''),
				el.start_date,
				el.end_date,
				COALESCE(el.leave_type, 0),
				COALESCE(lt.label, '')
			FROM employee_leaves el
			INNER JOIN employee_leave_type lt ON lt.id = el.leave_type
			INNER JOIN employee e ON e.uuid = el.employee_id
			INNER JOIN employee es ON es.uuid = el.substituted_by
			WHERE %s
			%s
			ORDER BY el.start_date DESC
		`, orgExpr, dateFilter)
	}

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.LeaveManagementListItem, 0)
	for rows.Next() {
		var it model.LeaveManagementListItem
		var startDate sql.NullTime
		var endDate sql.NullTime
		var leaveType sql.NullInt64
		if err := rows.Scan(
			&it.LeaveID,
			&it.EmployeeID,
			&it.SubstitutedBy,
			&startDate,
			&endDate,
			&leaveType,
			&it.LeaveTypeLabel,
		); err != nil {
			return nil, err
		}

		if startDate.Valid {
			it.StartDate = startDate.Time.Format("2006-01-02")
		}
		if endDate.Valid {
			it.EndDate = endDate.Time.Format("2006-01-02")
		} else if startDate.Valid {
			it.EndDate = startDate.Time.Format("2006-01-02")
		}
		if leaveType.Valid {
			it.LeaveType = int(leaveType.Int64)
		}

		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
