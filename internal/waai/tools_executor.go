package waai

import (
	"context"
	"database/sql"
	"strconv"
)

// ToolExecutor handles tool execution with database queries
type ToolExecutor struct {
	db       *sql.DB
	dbDriver string
}

// NewToolExecutor creates a new tool executor
func NewToolExecutor(db *sql.DB, dbDriver string) *ToolExecutor {
	return &ToolExecutor{
		db:       db,
		dbDriver: dbDriver,
	}
}

func (te *ToolExecutor) getPlaceholder(pos int) string {
	if te.dbDriver == "mysql" {
		return "?"
	}
	return "$" + strconv.Itoa(pos)
}

func (te *ToolExecutor) textCompareExpr(column string, pos int) string {
	if te.dbDriver == "mysql" {
		return column + " = " + te.getPlaceholder(pos)
	}
	return column + "::text = " + te.getPlaceholder(pos)
}

func (te *ToolExecutor) dateParamExpr(pos int) string {
	if te.dbDriver == "mysql" {
		return te.getPlaceholder(pos)
	}
	return te.getPlaceholder(pos) + "::date"
}

// ExecuteGetBusinessSnapshot returns current business metrics
func (te *ToolExecutor) ExecuteGetBusinessSnapshot(ctx context.Context, orgID string) map[string]interface{} {
	result := map[string]interface{}{
		"fleet_count":     0,
		"available_units": 0,
		"today_bookings":  0,
		"today_revenue":   0,
	}

	// Query fleet count
	fleetQuery := `SELECT COUNT(*) FROM fleets WHERE ` + te.textCompareExpr("organization_id", 1)
	var fleetCount int
	_ = te.db.QueryRowContext(ctx, fleetQuery, orgID).Scan(&fleetCount)
	result["fleet_count"] = fleetCount

	// Query available units (simplified)
	unitQuery := `
		SELECT COUNT(*) FROM fleet_units
		WHERE fleet_id IN (SELECT id FROM fleets WHERE ` + te.textCompareExpr("organization_id", 1) + `)
		AND is_active = true
	`
	var unitCount int
	_ = te.db.QueryRowContext(ctx, unitQuery, orgID).Scan(&unitCount)
	result["available_units"] = unitCount

	// Query today's bookings
	bookingQuery := `
		SELECT COUNT(*) FROM bookings
		WHERE ` + te.textCompareExpr("organization_id", 1) + `
		AND DATE(created_at) = CURRENT_DATE
	`
	var bookingCount int
	_ = te.db.QueryRowContext(ctx, bookingQuery, orgID).Scan(&bookingCount)
	result["today_bookings"] = bookingCount

	// Query today's revenue (simplified, assumes there's a revenue tracking)
	revenueQuery := `
		SELECT COALESCE(SUM(total_price), 0) FROM bookings
		WHERE ` + te.textCompareExpr("organization_id", 1) + `
		AND DATE(created_at) = CURRENT_DATE
		AND status = 'completed'
	`
	var revenue float64
	_ = te.db.QueryRowContext(ctx, revenueQuery, orgID).Scan(&revenue)
	result["today_revenue"] = revenue

	return result
}

// ExecuteGetFleetAvailability returns available fleet units for a date range
func (te *ToolExecutor) ExecuteGetFleetAvailability(ctx context.Context, orgID string, dateStart, dateEnd string) map[string]interface{} {
	result := map[string]interface{}{
		"available_units": 0,
		"date_range":      dateStart + " to " + dateEnd,
		"details":         []map[string]interface{}{},
	}

	// Query available units for date range
	query := `
		SELECT DISTINCT fu.id, fu.name, ft.name as fleet_type
		FROM fleet_units fu
		JOIN fleets f ON fu.fleet_id = f.id
		JOIN fleet_types ft ON f.fleet_type_id = ft.id
		WHERE ` + te.textCompareExpr("f.organization_id", 1) + `
		AND fu.is_active = true
		AND fu.id NOT IN (
			SELECT DISTINCT fu2.id
			FROM bookings b
			JOIN booking_units bu ON b.id = bu.booking_id
			JOIN fleet_units fu2 ON bu.fleet_unit_id = fu2.id
			WHERE b.start_date <= ` + te.dateParamExpr(3) + `
			AND b.end_date >= ` + te.dateParamExpr(2) + `
			AND b.status NOT IN ('cancelled')
		)
		LIMIT 20
	`

	rows, err := te.db.QueryContext(ctx, query, orgID, dateStart, dateEnd)
	if err != nil {
		return result
	}
	defer rows.Close()

	count := 0
	details := []map[string]interface{}{}

	for rows.Next() {
		var id int64
		var unitName, fleetType string
		if err := rows.Scan(&id, &unitName, &fleetType); err != nil {
			continue
		}
		count++
		details = append(details, map[string]interface{}{
			"id":         id,
			"name":       unitName,
			"fleet_type": fleetType,
		})
	}

	result["available_units"] = count
	result["details"] = details

	return result
}

// ExecuteGetBookingList returns list of bookings with optional status filter
func (te *ToolExecutor) ExecuteGetBookingList(ctx context.Context, orgID string, status string, limit int) []map[string]interface{} {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT id, booking_code, start_date, end_date, status, total_price
		FROM bookings
		WHERE ` + te.textCompareExpr("organization_id", 1) + `
	`
	args := []interface{}{orgID}

	if status != "" {
		query += ` AND status = ` + te.getPlaceholder(2)
		args = append(args, status)
	}

	query += ` ORDER BY created_at DESC LIMIT ` + te.getPlaceholder(len(args)+1)
	args = append(args, limit)

	rows, err := te.db.QueryContext(ctx, query, args...)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()

	bookings := []map[string]interface{}{}

	for rows.Next() {
		var id int64
		var code, status, startDate, endDate string
		var price float64

		if err := rows.Scan(&id, &code, &startDate, &endDate, &status, &price); err != nil {
			continue
		}

		bookings = append(bookings, map[string]interface{}{
			"id":          id,
			"code":        code,
			"start_date":  startDate,
			"end_date":    endDate,
			"status":      status,
			"total_price": price,
		})
	}

	return bookings
}

// ExecuteGetRevenueSummary returns revenue data for a period
func (te *ToolExecutor) ExecuteGetRevenueSummary(ctx context.Context, orgID string, period string) map[string]interface{} {
	result := map[string]interface{}{
		"period":              period,
		"total_revenue":       0,
		"transaction_count":   0,
		"average_transaction": 0,
	}

	// Determine date range based on period
	var dateFilter string
	switch period {
	case "daily":
		dateFilter = `DATE(created_at) = CURRENT_DATE`
	case "weekly":
		dateFilter = `created_at >= CURRENT_DATE - INTERVAL '7 days'`
	case "monthly":
		dateFilter = `created_at >= CURRENT_DATE - INTERVAL '30 days'`
	default:
		dateFilter = `created_at >= CURRENT_DATE - INTERVAL '1 day'`
	}

	query := `
		SELECT
			COALESCE(SUM(total_price), 0) as total_revenue,
			COUNT(*) as transaction_count,
			COALESCE(AVG(total_price), 0) as average_transaction
		FROM bookings
		WHERE ` + te.textCompareExpr("organization_id", 1) + `
		AND status = 'completed'
		AND ` + dateFilter

	var totalRevenue float64
	var transactionCount int
	var avgTransaction float64

	err := te.db.QueryRowContext(ctx, query, orgID).Scan(
		&totalRevenue,
		&transactionCount,
		&avgTransaction,
	)

	if err != nil && err != sql.ErrNoRows {
		// Log error but return result anyway
	}

	result["total_revenue"] = totalRevenue
	result["transaction_count"] = transactionCount
	result["average_transaction"] = avgTransaction

	return result
}

// MockToolExecutor untuk testing tanpa database
type MockToolExecutor struct{}

// ExecuteGetBusinessSnapshot mock
func (mte *MockToolExecutor) ExecuteGetBusinessSnapshot(ctx context.Context, orgID string) map[string]interface{} {
	return map[string]interface{}{
		"fleet_count":     5,
		"available_units": 12,
		"today_bookings":  3,
		"today_revenue":   2500000,
	}
}

// ExecuteGetFleetAvailability mock
func (mte *MockToolExecutor) ExecuteGetFleetAvailability(ctx context.Context, orgID string, dateStart, dateEnd string) map[string]interface{} {
	return map[string]interface{}{
		"available_units": 8,
		"date_range":      dateStart + " to " + dateEnd,
		"details": []map[string]interface{}{
			{"id": 1, "name": "Bus 01", "fleet_type": "Coach"},
			{"id": 2, "name": "Bus 02", "fleet_type": "Coach"},
		},
	}
}

// ExecuteGetBookingList mock
func (mte *MockToolExecutor) ExecuteGetBookingList(ctx context.Context, orgID string, status string, limit int) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id":          1001,
			"code":        "BK-001",
			"start_date":  "2026-06-15",
			"end_date":    "2026-06-17",
			"status":      "confirmed",
			"total_price": 1500000,
		},
	}
}

// ExecuteGetRevenueSummary mock
func (mte *MockToolExecutor) ExecuteGetRevenueSummary(ctx context.Context, orgID string, period string) map[string]interface{} {
	return map[string]interface{}{
		"period":              period,
		"total_revenue":       2500000,
		"transaction_count":   5,
		"average_transaction": 500000,
	}
}
