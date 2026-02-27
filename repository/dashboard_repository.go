package repository

import (
	"database/sql"
	"fmt"
	"math"
	"service-travego/model"
	"time"
)

type DashboardRepository struct {
	db     *sql.DB
	driver string
}

func NewDashboardRepository(db *sql.DB, driver string) *DashboardRepository {
	return &DashboardRepository{
		db:     db,
		driver: driver,
	}
}

func (r *DashboardRepository) getPlaceholder(pos int) string {
	if r.driver == "postgres" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}

func (r *DashboardRepository) GetPartnerSummary(orgID string) (*model.DashboardPartnerSummaryResponse, error) {
	resp := &model.DashboardPartnerSummaryResponse{}

	// 1. Orders Summary
	ordersSummary, err := r.getOrdersSummary(orgID)
	if err != nil {
		return nil, err
	}
	resp.Orders = *ordersSummary

	// 2. Members (Placeholder)
	resp.Members = model.DashboardSummaryItem{Period: "this month"}
	// 3. Messages (Placeholder)
	resp.Messages = model.DashboardSummaryItem{Period: "this month"}
	// 4. Revenue (Placeholder)
	resp.Revenue = model.DashboardSummaryItem{Period: "this month"}

	return resp, nil
}

func (r *DashboardRepository) getOrdersSummary(orgID string) (*model.DashboardOrdersSummary, error) {
	// Total orders (all time)
	var totalOrders int
	queryTotal := fmt.Sprintf(`
		SELECT COUNT(order_id)
		FROM fleet_orders
		WHERE organization_id = %s
	`, r.getPlaceholder(1))

	err := r.db.QueryRow(queryTotal, orgID).Scan(&totalOrders)
	if err != nil {
		return nil, err
	}

	// Calculate percentage (This Month vs Last Month)
	now := time.Now()
	// Start of this month
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	// Start of last month
	startOfLastMonth := startOfMonth.AddDate(0, -1, 0)

	// Start of next month (for upper bound of this month)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	var currentMonthCount int
	var lastMonthCount int

	queryMonth := fmt.Sprintf(`
		SELECT COUNT(order_id)
		FROM fleet_orders
		WHERE organization_id = %s AND created_at >= %s AND created_at < %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	// Current Month
	err = r.db.QueryRow(queryMonth, orgID, startOfMonth, startOfNextMonth).Scan(&currentMonthCount)
	if err != nil {
		return nil, err
	}

	// Last Month
	err = r.db.QueryRow(queryMonth, orgID, startOfLastMonth, startOfMonth).Scan(&lastMonthCount)
	if err != nil {
		return nil, err
	}

	var percentage float64
	direction := "flat"

	if lastMonthCount > 0 {
		percentage = (float64(currentMonthCount-lastMonthCount) / float64(lastMonthCount)) * 100
	} else if currentMonthCount > 0 {
		// No orders last month, but orders this month -> 100% increase (or treat as max)
		percentage = 100
	} else {
		percentage = 0
	}

	if percentage > 0 {
		direction = "up"
	} else if percentage < 0 {
		direction = "down"
	}

	return &model.DashboardOrdersSummary{
		TotalOrders: totalOrders,
		Percentage:  math.Round(percentage*100) / 100, // Round to 2 decimal places
		Direction:   direction,
		Period:      "this month",
	}, nil
}
