package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"service-travego/model"
	"strings"
	"sync"
	"time"
)

type PackageConfig struct {
	Packages []struct {
		PackageID   string `json:"package_id"`
		PackageName string `json:"package_name"`
	} `json:"packages"`
}

var (
	packagesOnce   sync.Once
	packageNameMap map[string]string
)

func loadPackages() {
	packagesOnce.Do(func() {
		packageNameMap = make(map[string]string)
		file, err := os.Open("config/packages.json")
		if err != nil {
			return
		}
		defer file.Close()

		var config PackageConfig
		if err := json.NewDecoder(file).Decode(&config); err != nil {
			return
		}

		for _, pkg := range config.Packages {
			packageNameMap[pkg.PackageID] = pkg.PackageName
		}
	})
}

type SystemRepository struct {
	db     *sql.DB
	driver string
}

func NewSystemRepository(db *sql.DB, driver string) *SystemRepository {
	loadPackages()
	return &SystemRepository{
		db:     db,
		driver: driver,
	}
}

func (r *SystemRepository) GetPackageName(packageID string) string {
	if name, ok := packageNameMap[packageID]; ok {
		return name
	}
	return packageID
}

func (r *SystemRepository) getPlaceholder(pos int) string {
	if r.driver == "postgres" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}

func calculateDateRanges(period string) (currentStart, currentEnd, lastStart, lastEnd time.Time, periodLabel string) {
	now := time.Now()
	var year, month int

	switch period {
	case "this_month":
		year = now.Year()
		month = int(now.Month())
		currentStart = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		currentEnd = currentStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
		lastYear := year
		lastMonth := month - 1
		if lastMonth == 0 {
			lastMonth = 12
			lastYear--
		}
		lastStart = time.Date(lastYear, time.Month(lastMonth), 1, 0, 0, 0, 0, time.Local)
		lastEnd = lastStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
		periodLabel = fmt.Sprintf("%d-%02d", year, month)
	case "last_month":
		year = now.Year()
		month = int(now.Month()) - 1
		if month == 0 {
			month = 12
			year--
		}
		currentStart = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		currentEnd = currentStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
		lastYear := year
		lastMonth := month - 1
		if lastMonth == 0 {
			lastMonth = 12
			lastYear--
		}
		lastStart = time.Date(lastYear, time.Month(lastMonth), 1, 0, 0, 0, 0, time.Local)
		lastEnd = lastStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
		periodLabel = fmt.Sprintf("%d-%02d", year, month)
	case "this_year":
		year = now.Year()
		currentStart = time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
		currentEnd = currentStart.AddDate(1, 0, 0).Add(-time.Nanosecond)
		lastStart = time.Date(year-1, 1, 1, 0, 0, 0, 0, time.Local)
		lastEnd = lastStart.AddDate(1, 0, 0).Add(-time.Nanosecond)
		periodLabel = fmt.Sprintf("%d", year)
	case "last_year":
		year = now.Year() - 1
		currentStart = time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
		currentEnd = currentStart.AddDate(1, 0, 0).Add(-time.Nanosecond)
		lastStart = time.Date(year-1, 1, 1, 0, 0, 0, 0, time.Local)
		lastEnd = lastStart.AddDate(1, 0, 0).Add(-time.Nanosecond)
		periodLabel = fmt.Sprintf("%d", year)
	case "all_time":
		currentStart = time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local)
		currentEnd = now
		lastStart = time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local)
		lastEnd = now
		periodLabel = "All Time"
	default:
		year = now.Year()
		month = int(now.Month())
		currentStart = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		currentEnd = currentStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
		lastYear := year
		lastMonth := month - 1
		if lastMonth == 0 {
			lastMonth = 12
			lastYear--
		}
		lastStart = time.Date(lastYear, time.Month(lastMonth), 1, 0, 0, 0, 0, time.Local)
		lastEnd = lastStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
		periodLabel = fmt.Sprintf("%d-%02d", year, month)
	}

	return
}

func (r *SystemRepository) getSingleCount(query string, args ...interface{}) (int64, error) {
	var count sql.NullInt64
	err := r.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}
	if count.Valid {
		return count.Int64, nil
	}
	return 0, nil
}

func (r *SystemRepository) getSingleFloat(query string, args ...interface{}) (float64, error) {
	var val sql.NullFloat64
	err := r.db.QueryRow(query, args...).Scan(&val)
	if err != nil {
		return 0, err
	}
	if val.Valid {
		return val.Float64, nil
	}
	return 0, nil
}

func getPeriodTrunc(period string, transactionDate time.Time) string {
	now := time.Now()
	currentMonth := now.Month()

	switch period {
	case "this_month", "last_month":
		// 3 hari sekali
		groupDay := ((transactionDate.Day() - 1) / 3) * 3
		return fmt.Sprintf("%04d-%02d-%02d", transactionDate.Year(), transactionDate.Month(), groupDay+1)
	case "this_year":
		if currentMonth <= 6 {
			// 3 pekan sekali (21 hari)
			groupDay := ((transactionDate.YearDay() - 1) / 21) * 21
			groupDate := time.Date(transactionDate.Year(), 1, 1, 0, 0, 0, 0, time.Local).AddDate(0, 0, groupDay)
			return fmt.Sprintf("%04d-%02d-%02d", groupDate.Year(), groupDate.Month(), groupDate.Day())
		}
		// tiap bulan
		return fmt.Sprintf("%04d-%02d", transactionDate.Year(), transactionDate.Month())
	case "last_year", "all_time":
		// tiap bulan
		return fmt.Sprintf("%04d-%02d", transactionDate.Year(), transactionDate.Month())
	default:
		return fmt.Sprintf("%04d-%02d", transactionDate.Year(), transactionDate.Month())
	}
}

func (r *SystemRepository) GetSummarize(period string) (*model.SystemSummarymarizeResponse, error) {
	if r.driver != "postgres" {
		return nil, fmt.Errorf("unsupported driver")
	}

	currentStart, currentEnd, lastStart, lastEnd, periodLabel := calculateDateRanges(period)

	var (
		revenueCurrent, revenueLast             float64
		totalUsersCurrent, totalUsersLast       int64
		activeUsersCurrent, activeUsersLast     int64
		organizationsCurrent, organizationsLast int64
		totalVisitCurrent, totalVisitLast       int64
	)

	revenueQuery := "SELECT COALESCE(SUM(payment_amount), 0) FROM travego_transactions WHERE updated_at IS NOT NULL AND status = 1 AND created_at BETWEEN $1 AND $2"
	revenueCurrent, _ = r.getSingleFloat(revenueQuery, currentStart, currentEnd)
	revenueLast, _ = r.getSingleFloat(revenueQuery, lastStart, lastEnd)

	totalUsersQuery := "SELECT COUNT(user_id) FROM users WHERE is_admin IS NULL AND is_active = true AND created_at BETWEEN $1 AND $2"
	totalUsersCurrent, _ = r.getSingleCount(totalUsersQuery, currentStart, currentEnd)
	totalUsersLast, _ = r.getSingleCount(totalUsersQuery, lastStart, lastEnd)

	totalVisitQuery := "SELECT COALESCE(SUM(count), 0) FROM travego_visitors WHERE period::DATE BETWEEN $1 AND $2"
	totalVisitCurrent, _ = r.getSingleCount(totalVisitQuery, currentStart, currentEnd)
	totalVisitLast, _ = r.getSingleCount(totalVisitQuery, lastStart, lastEnd)

	activeUsersQuery := `
		SELECT COUNT(DISTINCT u.user_id) 
		FROM users u 
		INNER JOIN organization_users ou ON ou.user_id = u.user_id 
		INNER JOIN _subscription s ON ou.organization_id = s.organization_id 
		WHERE u.is_admin IS NULL AND s.expiry_date >= NOW() AND u.created_at BETWEEN $1 AND $2
	`
	activeUsersCurrent, _ = r.getSingleCount(activeUsersQuery, currentStart, currentEnd)
	activeUsersLast, _ = r.getSingleCount(activeUsersQuery, lastStart, lastEnd)

	organizationsQuery := "SELECT COUNT(*) FROM organizations WHERE created_at BETWEEN $1 AND $2"
	organizationsCurrent, _ = r.getSingleCount(organizationsQuery, currentStart, currentEnd)
	organizationsLast, _ = r.getSingleCount(organizationsQuery, lastStart, lastEnd)

	// Get transaction metrics
	metricsQuery := `
		SELECT 
			transaction_date, 
			package_id, 
			COALESCE(SUM(payment_amount), 0) as revenue
		FROM travego_transactions
		WHERE status = 1 AND updated_at IS NOT NULL
		GROUP BY transaction_date, package_id
		ORDER BY transaction_date ASC, package_id ASC
	`
	rows, err := r.db.Query(metricsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type tempMetric struct {
		periodKey string
		packageID string
		revenue   float64
	}

	var tempMetrics []tempMetric
	for rows.Next() {
		var tDate time.Time
		var pkg string
		var rev float64
		if err := rows.Scan(&tDate, &pkg, &rev); err != nil {
			return nil, err
		}
		periodKey := getPeriodTrunc(period, tDate)
		tempMetrics = append(tempMetrics, tempMetric{
			periodKey: periodKey,
			packageID: pkg,
			revenue:   rev,
		})
	}

	periodMetrics := make(map[string]map[string]float64)
	for _, tm := range tempMetrics {
		if _, ok := periodMetrics[tm.periodKey]; !ok {
			periodMetrics[tm.periodKey] = make(map[string]float64)
		}
		periodMetrics[tm.periodKey][tm.packageID] += tm.revenue
	}

	var metrics []model.MetricPeriod
	for periodKey, pkgMap := range periodMetrics {
		var items []model.MetricItem
		for pkgID, rev := range pkgMap {
			packageName, ok := packageNameMap[pkgID]
			if !ok {
				packageName = pkgID
			}
			items = append(items, model.MetricItem{
				PackageName: packageName,
				Revenue:     rev,
			})
		}
		metrics = append(metrics, model.MetricPeriod{
			Period: periodKey,
			Items:  items,
		})
	}

	// Get visitor metrics
	visitorQuery := `SELECT count, period FROM travego_visitors ORDER BY period ASC`
	visitorRows, err := r.db.Query(visitorQuery)
	if err != nil {
		return nil, err
	}
	defer visitorRows.Close()

	visitorPeriodMetrics := make(map[string]int64)
	for visitorRows.Next() {
		var count int64
		var vPeriod time.Time
		if err := visitorRows.Scan(&count, &vPeriod); err != nil {
			return nil, err
		}
		groupedPeriod := getPeriodTrunc(period, vPeriod)
		visitorPeriodMetrics[groupedPeriod] += count
	}

	var visitorMetrics []model.VisitorMetricPeriod
	for periodKey, total := range visitorPeriodMetrics {
		visitorMetrics = append(visitorMetrics, model.VisitorMetricPeriod{
			Period:     periodKey,
			TotalVisit: total,
		})
	}

	// Get active user metrics - use user created date for period grouping
	activeUserQuery := `
		SELECT DISTINCT u.created_at
		FROM users u 
		INNER JOIN organization_users ou ON ou.user_id = u.user_id 
		INNER JOIN _subscription s ON ou.organization_id = s.organization_id 
		WHERE u.is_admin IS NULL AND s.expiry_date >= NOW()
	`
	activeUserRows, err := r.db.Query(activeUserQuery)
	if err != nil {
		return nil, err
	}
	defer activeUserRows.Close()

	activeUserPeriodMetrics := make(map[string]int64)
	for activeUserRows.Next() {
		var uCreatedAt time.Time
		if err := activeUserRows.Scan(&uCreatedAt); err != nil {
			return nil, err
		}
		groupedPeriod := getPeriodTrunc(period, uCreatedAt)
		activeUserPeriodMetrics[groupedPeriod]++
	}

	var activeUserMetrics []model.ActiveUserMetricPeriod
	for periodKey, total := range activeUserPeriodMetrics {
		activeUserMetrics = append(activeUserMetrics, model.ActiveUserMetricPeriod{
			Period:      periodKey,
			ActiveUsers: total,
		})
	}

	return &model.SystemSummarymarizeResponse{
		Revenue: model.PeriodSummary{
			CurrentPeriod: int64(revenueCurrent),
			LastPeriod:    int64(revenueLast),
		},
		TotalUsers: model.PeriodSummary{
			CurrentPeriod: totalUsersCurrent,
			LastPeriod:    totalUsersLast,
		},
		TotalVisit: model.PeriodSummary{
			CurrentPeriod: totalVisitCurrent,
			LastPeriod:    totalVisitLast,
		},
		ActiveUsers: model.PeriodSummary{
			CurrentPeriod: activeUsersCurrent,
			LastPeriod:    activeUsersLast,
		},
		Organization: model.PeriodSummary{
			CurrentPeriod: organizationsCurrent,
			LastPeriod:    organizationsLast,
		},
		Period:            periodLabel,
		Matrics:           metrics,
		VisitorMatrics:    visitorMetrics,
		ActiveUserMatrics: activeUserMetrics,
	}, nil
}

func (r *SystemRepository) GetDeviceList(search, status string) ([]model.DeviceListItem, error) {
	if r.driver != "postgres" {
		return nil, fmt.Errorf("unsupported driver")
	}

	baseQuery := `
		SELECT COALESCE(ac.device_id, ''), COALESCE(ac.device_name, ''), COALESCE(ac.device_token, ''),
		       o.organization_name, COALESCE(o.company_name, ''), ac.account as account_number,
		       ac.created_at, ac.updated_at
		FROM assistant_customers ac
		INNER JOIN organizations o ON o.organization_id = ac.organization_id
	`
	var args []interface{}
	pos := 1

	var conditions []string

	switch status {
	case "verified":
		conditions = append(conditions, `ac.device_id IS NOT NULL AND ac.device_id != ''`)
	case "unverified":
		conditions = append(conditions, `ac.device_id IS NULL OR ac.device_id = ''`)
	}

	if search != "" {
		conditions = append(conditions, fmt.Sprintf(`(o.organization_name ILIKE $%d OR o.company_name ILIKE $%d)`, pos, pos+1))
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
		pos += 2
	}

	if len(conditions) > 0 {
		baseQuery += ` WHERE ` + strings.Join(conditions, " AND ")
	}

	baseQuery += ` ORDER BY COALESCE(ac.updated_at, ac.created_at) DESC`

	rows, err := r.db.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type tempDevice struct {
		deviceID         sql.NullString
		deviceName       sql.NullString
		deviceToken      sql.NullString
		organizationName sql.NullString
		companyName      sql.NullString
		accountNumber    sql.NullString
		createdAt        sql.NullTime
		updatedAt        sql.NullTime
	}

	var tempList []tempDevice
	for rows.Next() {
		var t tempDevice
		if err := rows.Scan(
			&t.deviceID, &t.deviceName, &t.deviceToken,
			&t.organizationName, &t.companyName, &t.accountNumber,
			&t.createdAt, &t.updatedAt,
		); err != nil {
			return nil, err
		}
		tempList = append(tempList, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]model.DeviceListItem, 0, len(tempList))
	for _, t := range tempList {
		item := model.DeviceListItem{
			DeviceID:         t.deviceID.String,
			DeviceName:       t.deviceName.String,
			DeviceToken:      t.deviceToken.String,
			OrganizationName: t.organizationName.String,
			CompanyName:      t.companyName.String,
			AccountNumber:    t.accountNumber.String,
		}
		if t.createdAt.Valid {
			item.CreatedAt = t.createdAt.Time.Format("2006-01-02 15:04:05")
		}
		if t.updatedAt.Valid {
			item.UpdatedAt = t.updatedAt.Time.Format("2006-01-02 15:04:05")
		}
		out = append(out, item)
	}

	return out, nil
}

func (r *SystemRepository) UpdateDevice(account string, action string, enableData *model.DeviceEnableRequest) error {
	if r.driver != "postgres" {
		return fmt.Errorf("unsupported driver")
	}

	if action == "disable" {
		query := `
			UPDATE assistant_customers
			SET device_id = NULL, device_token = NULL, device_name = NULL,
			    updated_at = NOW()
			WHERE account = $1
		`
		result, err := r.db.Exec(query, account)
		if err != nil {
			return err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return sql.ErrNoRows
		}
		return nil
	}

	if action == "enable" {
		query := `
			UPDATE assistant_customers
			SET device_id = $1, device_name = $2, device_token = $3,
			    updated_at = NOW()
			WHERE account = $4
		`
		result, err := r.db.Exec(query, enableData.DeviceID, enableData.DeviceName, enableData.DeviceToken, account)
		if err != nil {
			return err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return sql.ErrNoRows
		}
		return nil
	}

	return fmt.Errorf("unknown action: %s", action)
}

type rawOrganization struct {
	OrganizationID   sql.NullString
	OrganizationCode sql.NullString
	OrganizationName sql.NullString
	CompanyName      sql.NullString
	Address          sql.NullString
	City             sql.NullString
	Province         sql.NullString
	Phone            sql.NullString
	Logo             sql.NullString
	PackageID        sql.NullString
	ExpiryDate       sql.NullTime
}

func (r *SystemRepository) GetOrganizations(search string, status string) ([]rawOrganization, error) {
	if r.driver != "postgres" {
		return nil, fmt.Errorf("unsupported driver")
	}

	query := `
		SELECT o.organization_id, o.organization_code, o.organization_name,
		       o.company_name, o.address, o.city, o.province, o.phone, o.logo,
		       s.package_id, s.expiry_date
		FROM organizations o
		LEFT JOIN _subscription s ON o.organization_id = s.organization_id
		WHERE 1=1
	`
	var args []interface{}
	pos := 1

	if search != "" {
		query += fmt.Sprintf(` AND (o.organization_name ILIKE $%d OR o.company_name ILIKE $%d OR o.organization_code ILIKE $%d OR o.phone ILIKE $%d)`, pos, pos+1, pos+2, pos+3)
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
		pos += 4
	}

	if status == "active" {
		query += fmt.Sprintf(` AND s.expiry_date >= NOW()`)
	} else if status == "inactive" {
		query += fmt.Sprintf(` AND (s.expiry_date IS NULL OR s.expiry_date < NOW())`)
	}

	query += ` ORDER BY o.organization_name ASC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []rawOrganization
	for rows.Next() {
		var t rawOrganization
		if err := rows.Scan(
			&t.OrganizationID, &t.OrganizationCode, &t.OrganizationName,
			&t.CompanyName, &t.Address, &t.City, &t.Province, &t.Phone, &t.Logo,
			&t.PackageID, &t.ExpiryDate,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

type rawUser struct {
	Fullname         sql.NullString
	Phone            sql.NullString
	Email            sql.NullString
	Avatar           sql.NullString
	OrganizationName sql.NullString
	OrganizationRole sql.NullInt64
	IsActive         sql.NullBool
}

func (r *SystemRepository) GetUsers(search string, isActive string) ([]rawUser, error) {
	if r.driver != "postgres" {
		return nil, fmt.Errorf("unsupported driver")
	}

	query := `
		SELECT u.fullname, u.phone, u.email, u.avatar,
		       o.organization_name, ou.organization_role, u.is_active
		FROM users u
		INNER JOIN organization_users ou ON u.user_id = ou.user_id
		INNER JOIN organizations o ON o.organization_id = ou.organization_id
		WHERE 1=1
	`
	var args []interface{}
	pos := 1

	if search != "" {
		query += fmt.Sprintf(` AND (u.fullname ILIKE $%d OR u.email ILIKE $%d OR o.organization_name ILIKE $%d OR u.phone ILIKE $%d)`, pos, pos+1, pos+2, pos+3)
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
		pos += 4
	}

	if isActive == "true" {
		query += ` AND u.is_active = true`
	} else if isActive == "false" {
		query += ` AND u.is_active = false`
	}

	query += ` ORDER BY u.fullname ASC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []rawUser
	for rows.Next() {
		var t rawUser
		if err := rows.Scan(
			&t.Fullname, &t.Phone, &t.Email, &t.Avatar,
			&t.OrganizationName, &t.OrganizationRole, &t.IsActive,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

type rawSystemMessage struct {
	MessageID  sql.NullString
	TopicID    sql.NullInt64
	Fullname   sql.NullString
	CompanyName sql.NullString
	Email      sql.NullString
	Whatsapp   sql.NullString
	Scale      sql.NullString
	Messages   sql.NullString
	CreatedAt  sql.NullTime
	IsRead     sql.NullBool
}

func (r *SystemRepository) GetMessages() ([]model.SystemMessageItem, error) {
	query := `
		SELECT message_id, topic_id, fullname, company_name, email, whatsapp, scale, messages, created_at, is_read
		FROM travego_messages
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	topicMap := map[int64]string{
		1: "Demo Kerja",
		2: "Harga dan Penawaran",
		3: "Bantuan Teknis",
		4: "Kerja Sama",
		5: "Lainnya",
	}

	var out []model.SystemMessageItem
	for rows.Next() {
		var t rawSystemMessage
		if err := rows.Scan(&t.MessageID, &t.TopicID, &t.Fullname, &t.CompanyName, &t.Email, &t.Whatsapp, &t.Scale, &t.Messages, &t.CreatedAt, &t.IsRead); err != nil {
			return nil, err
		}
		item := model.SystemMessageItem{
			MessageID:   t.MessageID.String,
			Fullname:    t.Fullname.String,
			CompanyName: t.CompanyName.String,
			Email:       t.Email.String,
			Whatsapp:    t.Whatsapp.String,
			Scale:       t.Scale.String,
			Messages:    t.Messages.String,
			IsRead:      t.IsRead.Bool,
		}
		if t.TopicID.Valid {
			item.TopicID = int(t.TopicID.Int64)
			if label, ok := topicMap[t.TopicID.Int64]; ok {
				item.TopicLabel = label
			} else {
				item.TopicLabel = "Lainnya"
			}
		}
		if t.CreatedAt.Valid {
			item.CreatedAt = t.CreatedAt.Time.Format("2006-01-02 15:04:05")
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SystemRepository) ReadMessage(messageID string) error {
	query := "UPDATE travego_messages SET is_read = true WHERE message_id = " + r.getPlaceholder(1)
	_, err := r.db.Exec(query, messageID)
	return err
}

