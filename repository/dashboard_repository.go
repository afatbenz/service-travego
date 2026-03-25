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

func (r *DashboardRepository) GetDashboard(orgID string) (*model.DashboardResponse, error) {
	resp := &model.DashboardResponse{}

	tx, err := r.getTransactionMetrics(orgID)
	if err != nil {
		return nil, err
	}
	resp.Transaction = *tx

	cust, err := r.getCustomerMetrics(orgID)
	if err != nil {
		return nil, err
	}
	resp.Customers = *cust

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

func (r *DashboardRepository) getTransactionMetrics(orgID string) (*model.DashboardTransaction, error) {
	now := time.Now()
	from := now.AddDate(-1, 0, 0)

	var total int
	qTotal := fmt.Sprintf(`
		SELECT COUNT(order_id)
		FROM customer_orders
		WHERE organization_id = %s AND created_at >= %s AND created_at <= %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	if err := r.db.QueryRow(qTotal, orgID, from, now).Scan(&total); err != nil {
		return nil, err
	}

	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	startOfLastMonth := startOfMonth.AddDate(0, -1, 0)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	var currentMonthCount int
	var lastMonthCount int

	qMonth := fmt.Sprintf(`
		SELECT COUNT(order_id)
		FROM customer_orders
		WHERE organization_id = %s AND created_at >= %s AND created_at < %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	if err := r.db.QueryRow(qMonth, orgID, startOfMonth, startOfNextMonth).Scan(&currentMonthCount); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(qMonth, orgID, startOfLastMonth, startOfMonth).Scan(&lastMonthCount); err != nil {
		return nil, err
	}

	var percentage float64
	if lastMonthCount > 0 {
		percentage = (float64(currentMonthCount-lastMonthCount) / float64(lastMonthCount)) * 100
	} else if currentMonthCount > 0 {
		percentage = 100
	} else {
		percentage = 0
	}

	return &model.DashboardTransaction{
		TotalOrder:      total,
		OrderPercentage: math.Round(percentage*100) / 100,
	}, nil
}

func (r *DashboardRepository) getCustomerMetrics(orgID string) (*model.DashboardCustomers, error) {
	now := time.Now()
	from := now.AddDate(-1, 0, 0)

	var total int
	qTotal := fmt.Sprintf(`
		SELECT COUNT(customer_id)
		FROM customers
		WHERE organization_id = %s AND created_at >= %s AND created_at <= %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	if err := r.db.QueryRow(qTotal, orgID, from, now).Scan(&total); err != nil {
		return nil, err
	}

	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	startOfLastMonth := startOfMonth.AddDate(0, -1, 0)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	var currentMonthCount int
	var lastMonthCount int

	qMonth := fmt.Sprintf(`
		SELECT COUNT(customer_id)
		FROM customers
		WHERE organization_id = %s AND created_at >= %s AND created_at < %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	if err := r.db.QueryRow(qMonth, orgID, startOfMonth, startOfNextMonth).Scan(&currentMonthCount); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(qMonth, orgID, startOfLastMonth, startOfMonth).Scan(&lastMonthCount); err != nil {
		return nil, err
	}

	var percentage float64
	if lastMonthCount > 0 {
		percentage = (float64(currentMonthCount-lastMonthCount) / float64(lastMonthCount)) * 100
	} else if currentMonthCount > 0 {
		percentage = 100
	} else {
		percentage = 0
	}

	return &model.DashboardCustomers{
		TotalCustomers:     total,
		CustomerPercentage: math.Round(percentage*100) / 100,
	}, nil
}
