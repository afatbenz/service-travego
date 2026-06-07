package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"service-travego/configs"
	"service-travego/database"
	"service-travego/model"
	"strings"
	"sync"
	"time"
)

var (
	dashboardCitiesOnce sync.Once
	dashboardCitiesMap  map[string]string
)

func ensureDashboardCitiesLoaded() {
	dashboardCitiesOnce.Do(func() {
		dashboardCitiesMap = map[string]string{}
		f, err := os.Open("config/location.json")
		if err != nil {
			return
		}
		defer f.Close()

		var loc model.Location
		if err := json.NewDecoder(f).Decode(&loc); err != nil {
			return
		}
		for _, c := range loc.Cities {
			id := strings.TrimSpace(c.ID)
			if id == "" {
				continue
			}
			dashboardCitiesMap[id] = c.Name
		}
	})
}

func getDashboardCitiesMap() map[string]string {
	ensureDashboardCitiesLoaded()
	return dashboardCitiesMap
}

type DashboardRepository struct {
	db     *sql.DB
	driver string
}

type DashboardFinanceRow struct {
	Period   time.Time
	Revenue  float64
	Expenses float64
}

func NewDashboardRepository(db *sql.DB, driver string) *DashboardRepository {
	ensureDashboardCitiesLoaded()
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

func (r *DashboardRepository) GetFinance(orgID string, groupBy string, startDate time.Time, endDate time.Time) ([]DashboardFinanceRow, error) {
	if r.driver != "postgres" {
		return nil, fmt.Errorf("unsupported driver")
	}

	var query string
	switch groupBy {
	case "day":
		query = fmt.Sprintf(`
			SELECT
				DATE(created_at) AS period,
				SUM(CASE WHEN transaction_type=1 THEN amount ELSE 0 END) AS revenue,
				SUM(CASE WHEN transaction_type=2 THEN amount ELSE 0 END) AS expenses
			FROM transactions
			WHERE organization_id=%s AND created_at BETWEEN %s AND %s
			GROUP BY DATE(created_at)
			ORDER BY period ASC
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	case "2day":
		query = fmt.Sprintf(`
			SELECT
				DATE_TRUNC('day', created_at) -
					(EXTRACT(DOY FROM created_at)::int %% 2) * INTERVAL '1 day' AS period,
				SUM(CASE WHEN transaction_type=1 THEN amount ELSE 0 END) AS revenue,
				SUM(CASE WHEN transaction_type=2 THEN amount ELSE 0 END) AS expenses
			FROM transactions
			WHERE organization_id=%s AND created_at BETWEEN %s AND %s
			GROUP BY period
			ORDER BY period ASC
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	case "week":
		query = fmt.Sprintf(`
			SELECT
				DATE_TRUNC('week', created_at) AS period,
				SUM(CASE WHEN transaction_type=1 THEN amount ELSE 0 END) AS revenue,
				SUM(CASE WHEN transaction_type=2 THEN amount ELSE 0 END) AS expenses
			FROM transactions
			WHERE organization_id=%s AND created_at BETWEEN %s AND %s
			GROUP BY DATE_TRUNC('week', created_at)
			ORDER BY period ASC
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	default:
		query = fmt.Sprintf(`
			SELECT
				DATE_TRUNC('month', created_at) AS period,
				SUM(CASE WHEN transaction_type=1 THEN amount ELSE 0 END) AS revenue,
				SUM(CASE WHEN transaction_type=2 THEN amount ELSE 0 END) AS expenses
			FROM transactions
			WHERE organization_id=%s AND created_at BETWEEN %s AND %s
			GROUP BY DATE_TRUNC('month', created_at)
			ORDER BY period ASC
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	}

	rows, err := database.Query(r.db, query, orgID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]DashboardFinanceRow, 0)
	for rows.Next() {
		var period time.Time
		var revenue sql.NullFloat64
		var expenses sql.NullFloat64
		if err := rows.Scan(&period, &revenue, &expenses); err != nil {
			return nil, err
		}
		items = append(items, DashboardFinanceRow{
			Period:   period,
			Revenue:  revenue.Float64,
			Expenses: expenses.Float64,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
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

	var (
		messages *model.DashboardMessages
		revenue  *model.DashboardRevenue
		expenses *model.DashboardRevenue
	)

	var wg sync.WaitGroup
	errCh := make(chan error, 3)

	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := r.getMessages(orgID)
		if err != nil {
			errCh <- err
			return
		}
		messages = res
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := r.getRevenueExpenses(orgID, 1)
		if err != nil {
			errCh <- err
			return
		}
		revenue = res
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := r.getRevenueExpenses(orgID, 2)
		if err != nil {
			errCh <- err
			return
		}
		expenses = res
	}()

	wg.Wait()
	close(errCh)

	for e := range errCh {
		if e != nil {
			return nil, e
		}
	}

	if messages != nil {
		resp.Messages = *messages
	}
	if revenue != nil {
		resp.Revenue = *revenue
	}
	if expenses != nil {
		resp.Expenses = *expenses
	}

	return resp, nil
}

func (r *DashboardRepository) getThisMonthBounds(now time.Time) (time.Time, time.Time, time.Time, time.Time) {
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	startOfLastMonth := startOfMonth.AddDate(0, -1, 0)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	endOfThisMonth := startOfNextMonth.Add(-time.Nanosecond)
	endOfLastMonth := startOfMonth.Add(-time.Nanosecond)

	return startOfMonth, endOfThisMonth, startOfLastMonth, endOfLastMonth
}

func (r *DashboardRepository) getMessages(orgID string) (*model.DashboardMessages, error) {
	now := time.Now()
	startCur, endCur, startPrev, endPrev := r.getThisMonthBounds(now)

	query := fmt.Sprintf(`
		SELECT COUNT(message_id)
		FROM messages
		WHERE organization_id = %s AND created_at BETWEEN %s AND %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	var current int
	if err := database.QueryRow(r.db, query, orgID, startCur, endCur).Scan(&current); err != nil {
		return nil, err
	}

	var previous int
	if err := database.QueryRow(r.db, query, orgID, startPrev, endPrev).Scan(&previous); err != nil {
		return nil, err
	}

	return &model.DashboardMessages{
		Current:  current,
		Previous: previous,
	}, nil
}

func (r *DashboardRepository) getRevenueExpenses(orgID string, TransactionItem int) (*model.DashboardRevenue, error) {
	now := time.Now()
	startCur, endCur, startPrev, endPrev := r.getThisMonthBounds(now)

	query := fmt.Sprintf(`
		SELECT COUNT(transaction_id) AS total, SUM(amount) AS amount
		FROM transactions
		WHERE organization_id = %s AND transaction_type = %s AND created_at BETWEEN %s AND %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))

	var currentTotal int
	var currentAmount sql.NullFloat64
	if err := database.QueryRow(r.db, query, orgID, TransactionItem, startCur, endCur).Scan(&currentTotal, &currentAmount); err != nil {
		return nil, err
	}

	var previousTotal int
	var previousAmount sql.NullFloat64
	if err := database.QueryRow(r.db, query, orgID, TransactionItem, startPrev, endPrev).Scan(&previousTotal, &previousAmount); err != nil {
		return nil, err
	}

	metrics, err := r.getTransactionMetricsByType(orgID, TransactionItem, startCur, endCur)
	if err != nil {
		return nil, err
	}

	return &model.DashboardRevenue{
		Current:            currentTotal,
		TotalAmount:        currentAmount.Float64,
		Previous:           previousTotal,
		PrevAmount:         previousAmount.Float64,
		TransactionMetrics: metrics,
	}, nil
}

func (r *DashboardRepository) getTransactionMetricsByType(orgID string, TransactionItem int, from time.Time, to time.Time) ([]model.DashboardTransactionMetricItem, error) {
	query := fmt.Sprintf(`
		SELECT transaction_type, SUM(amount) AS value
		FROM transactions
		WHERE organization_id = %s AND transaction_type = %s AND created_at BETWEEN %s AND %s
		GROUP BY transaction_type
		ORDER BY value DESC
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))

	rows, err := database.Query(r.db, query, orgID, TransactionItem, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.DashboardTransactionMetricItem, 0)
	for rows.Next() {
		var transactionType int
		var value sql.NullFloat64
		if err := rows.Scan(&transactionType, &value); err != nil {
			return nil, err
		}

		label := ""
		if l, ok := configs.TransactionTypeLabel[transactionType]; ok {
			label = l
		}

		items = append(items, model.DashboardTransactionMetricItem{
			Label: label,
			Value: value.Float64,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *DashboardRepository) GetTopDestinations(orgID string) ([]model.DashboardTopDestination, error) {
	query := fmt.Sprintf(`
		SELECT city_id, SUM(total) AS total FROM (
			SELECT foi.city_id, COUNT(*) AS total FROM fleet_order_itinerary foi
			INNER JOIN fleet_orders fo ON fo.order_id=foi.order_id
			WHERE fo.organization_id=%s AND fo.status=1 GROUP BY foi.city_id
			UNION ALL
			SELECT foi.city_id, COUNT(*) AS total FROM tour_package_itineraries foi
			INNER JOIN tour_package_orders fo ON fo.tour_package_id=foi.package_id
			WHERE fo.organization_id=%s AND fo.status=1 GROUP BY foi.city_id
		) t GROUP BY city_id ORDER BY total DESC LIMIT 5
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	rows, err := database.Query(r.db, query, orgID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cityMap := getDashboardCitiesMap()
	items := make([]model.DashboardTopDestination, 0)

	for rows.Next() {
		var cityID sql.NullString
		var total int
		if err := rows.Scan(&cityID, &total); err != nil {
			return nil, err
		}
		id := strings.TrimSpace(cityID.String)
		items = append(items, model.DashboardTopDestination{
			CityID:    id,
			CityLabel: cityMap[id],
			Total:     total,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *DashboardRepository) GetTopPickupCity(orgID string) ([]model.DashboardTopPickupCity, error) {
	query := fmt.Sprintf(`
		SELECT pickup_city_id, SUM(total) AS total FROM (
			SELECT pickup_city_id, COUNT(*) AS total FROM tour_package_orders WHERE organization_id=%s AND status=1 GROUP BY pickup_city_id
			UNION ALL
			SELECT pickup_city_id, COUNT(*) AS total FROM fleet_orders WHERE organization_id=%s AND status=1 GROUP BY pickup_city_id
		) t GROUP BY pickup_city_id ORDER BY total DESC LIMIT 5
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	rows, err := database.Query(r.db, query, orgID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cityMap := getDashboardCitiesMap()
	items := make([]model.DashboardTopPickupCity, 0)

	for rows.Next() {
		var cityID sql.NullString
		var total int
		if err := rows.Scan(&cityID, &total); err != nil {
			return nil, err
		}
		id := strings.TrimSpace(cityID.String)
		items = append(items, model.DashboardTopPickupCity{
			PickupCityID:    id,
			PickupCityLabel: cityMap[id],
			Total:           total,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *DashboardRepository) GetTopFleets(orgID string) ([]model.DashboardTopFleet, error) {
	query := fmt.Sprintf(`
		SELECT fu.vehicle_id, fu.plate_number, COUNT(sf.unit_id) AS total
		FROM schedule_fleets sf
		INNER JOIN fleet_units fu ON fu.unit_id=sf.unit_id
		WHERE sf.organization_id=%s
		GROUP BY sf.unit_id, fu.vehicle_id, fu.plate_number
		ORDER BY total DESC LIMIT 5
	`, r.getPlaceholder(1))

	rows, err := database.Query(r.db, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.DashboardTopFleet, 0)
	for rows.Next() {
		var vehicleID sql.NullString
		var plateNumber sql.NullString
		var total int
		if err := rows.Scan(&vehicleID, &plateNumber, &total); err != nil {
			return nil, err
		}
		items = append(items, model.DashboardTopFleet{
			VehicleID:   strings.TrimSpace(vehicleID.String),
			PlateNumber: plateNumber.String,
			Total:       total,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *DashboardRepository) GetTopTourPackages(orgID string) ([]model.DashboardTopTourPackage, error) {
	query := fmt.Sprintf(`
		SELECT tp.package_name, COUNT(tpo.order_id) AS total
		FROM tour_package_orders tpo
		INNER JOIN tour_packages tp ON tpo.tour_package_id=tp.uuid
		WHERE tpo.organization_id=%s AND tpo.status=1
		GROUP BY tpo.tour_package_id, tp.package_name
		ORDER BY total DESC LIMIT 5
	`, r.getPlaceholder(1))

	rows, err := database.Query(r.db, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.DashboardTopTourPackage, 0)
	for rows.Next() {
		var name sql.NullString
		var total int
		if err := rows.Scan(&name, &total); err != nil {
			return nil, err
		}
		items = append(items, model.DashboardTopTourPackage{
			PackageName: name.String,
			Total:       total,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *DashboardRepository) GetTopDrivers(orgID string) ([]model.DashboardTopDriver, error) {
	now := time.Now()
	startCur, endCur, _, _ := r.getThisMonthBounds(now)

	query := fmt.Sprintf(`
		SELECT e.fullname, COUNT(sft.uuid) AS total
		FROM schedule_fleet_teams sft
		INNER JOIN employee e ON sft.driver_id=e.uuid
		WHERE sft.organization_id=%s AND sft.created_at BETWEEN %s AND %s
		GROUP BY sft.driver_id, e.fullname
		ORDER BY total DESC LIMIT 5
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	rows, err := database.Query(r.db, query, orgID, startCur, endCur)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.DashboardTopDriver, 0)
	for rows.Next() {
		var fullname sql.NullString
		var total int
		if err := rows.Scan(&fullname, &total); err != nil {
			return nil, err
		}
		items = append(items, model.DashboardTopDriver{
			Fullname: fullname.String,
			Total:    total,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *DashboardRepository) GetTopCustomers(orgID string) ([]model.DashboardTopCustomer, error) {
	query := fmt.Sprintf(`
		SELECT c.customer_name, COUNT(co.order_id) AS total
		FROM customer_orders co
		INNER JOIN customers c ON co.customer_id=c.customer_id
		WHERE co.organization_id=%s
		GROUP BY co.customer_id, c.customer_name
		ORDER BY total DESC LIMIT 5
	`, r.getPlaceholder(1))

	rows, err := database.Query(r.db, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.DashboardTopCustomer, 0)
	for rows.Next() {
		var name sql.NullString
		var total int
		if err := rows.Scan(&name, &total); err != nil {
			return nil, err
		}
		items = append(items, model.DashboardTopCustomer{
			CustomerName: name.String,
			Total:        total,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *DashboardRepository) getOrdersSummary(orgID string) (*model.DashboardOrdersSummary, error) {
	// Total orders (all time)
	var totalOrders int
	queryTotal := fmt.Sprintf(`
		SELECT COUNT(order_id)
		FROM fleet_orders
		WHERE organization_id = %s
	`, r.getPlaceholder(1))

	err := database.QueryRow(r.db, queryTotal, orgID).Scan(&totalOrders)
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
	err = database.QueryRow(r.db, queryMonth, orgID, startOfMonth, startOfNextMonth).Scan(&currentMonthCount)
	if err != nil {
		return nil, err
	}

	// Last Month
	err = database.QueryRow(r.db, queryMonth, orgID, startOfLastMonth, startOfMonth).Scan(&lastMonthCount)
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
	if err := database.QueryRow(r.db, qTotal, orgID, from, now).Scan(&total); err != nil {
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

	if err := database.QueryRow(r.db, qMonth, orgID, startOfMonth, startOfNextMonth).Scan(&currentMonthCount); err != nil {
		return nil, err
	}
	if err := database.QueryRow(r.db, qMonth, orgID, startOfLastMonth, startOfMonth).Scan(&lastMonthCount); err != nil {
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
	if err := database.QueryRow(r.db, qTotal, orgID, from, now).Scan(&total); err != nil {
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

	if err := database.QueryRow(r.db, qMonth, orgID, startOfMonth, startOfNextMonth).Scan(&currentMonthCount); err != nil {
		return nil, err
	}
	if err := database.QueryRow(r.db, qMonth, orgID, startOfLastMonth, startOfMonth).Scan(&lastMonthCount); err != nil {
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
