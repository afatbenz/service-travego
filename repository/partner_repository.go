package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"service-travego/database"
	"service-travego/model"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	partnerCitiesOnce sync.Once
	partnerCitiesMap  map[string]string
)

func getPartnerCitiesMap() map[string]string {
	partnerCitiesOnce.Do(func() {
		partnerCitiesMap = map[string]string{}
		f, err := os.Open("config/location.json")
		if err != nil {
			fmt.Printf("Error opening location.json: %v\n", err)
			return
		}
		defer f.Close()
		var loc model.Location
		if err := json.NewDecoder(f).Decode(&loc); err != nil {
			fmt.Printf("Error decoding location.json: %v\n", err)
			return
		}
		for _, c := range loc.Cities {
			partnerCitiesMap[strings.TrimSpace(c.ID)] = c.Name
		}
	})
	return partnerCitiesMap
}

type PartnerRepository struct {
	db     *sql.DB
	driver string
}

func NewPartnerRepository(db *sql.DB, driver string) *PartnerRepository {
	return &PartnerRepository{db: db, driver: driver}
}

func (r *PartnerRepository) getPlaceholder(pos int) string {
	if r.driver == "postgres" || r.driver == "pgx" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}

func (r *PartnerRepository) GetCityLabel(cityID *int) string {
	if cityID == nil {
		return ""
	}
	m := getPartnerCitiesMap()
	if label, ok := m[fmt.Sprintf("%d", *cityID)]; ok {
		return label
	}
	return ""
}

func (r *PartnerRepository) List(orgID, partnerName, startDate, endDate string) ([]model.OperationPartner, error) {
	args := make([]interface{}, 0, 4)
	whereClauses := make([]string, 0, 5)
	placeholder := func() string {
		return fmt.Sprintf("$%d", len(args)+1)
	}

	whereClauses = append(whereClauses, fmt.Sprintf("op.organization_id::text = %s", placeholder()))
	args = append(args, orgID)
	whereClauses = append(whereClauses,
		"NULLIF(BTRIM(op.partner_name), '') IS NOT NULL",
		"NULLIF(BTRIM(op.partner_phone), '') IS NOT NULL",
		"NULLIF(BTRIM(op.pic_name), '') IS NOT NULL",
	)

	if partnerName = strings.TrimSpace(partnerName); partnerName != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("op.partner_name ILIKE %s", placeholder()))
		args = append(args, "%"+partnerName+"%")
	}

	revenueDateClauses := []string{"transaction_date IS NOT NULL"}
	if startDate = strings.TrimSpace(startDate); startDate != "" {
		revenueDateClauses = append(revenueDateClauses, fmt.Sprintf("transaction_date >= %s", placeholder()))
		args = append(args, startDate)
	}
	if endDate = strings.TrimSpace(endDate); endDate != "" {
		revenueDateClauses = append(revenueDateClauses, fmt.Sprintf("transaction_date < %s", placeholder()))
		args = append(args, endDate)
	}

	query := fmt.Sprintf(`
		SELECT
			op.partner_id,
			op.partner_name,
			op.partner_address,
			op.partner_city,
			op.partner_phone,
			op.partner_email,
			op.pic_name,
			op.created_at,
			op.organization_id,
			COUNT(DISTINCT (fuo.unit_id::text, COALESCE(fu.vehicle_id, ''), COALESCE(fu.plate_number, '')))
				FILTER (WHERE fuo.unit_id IS NOT NULL) AS total_unit,
			COALESCE(revenue.total_revenue, 0) AS total_revenue
		FROM operation_partner op
		LEFT JOIN fleet_unit_ownership fuo ON fuo.partner_id::text = op.partner_id::text
			AND fuo.organization_id::text = op.organization_id::text
		LEFT JOIN fleet_units fu ON fu.unit_id::text = fuo.unit_id::text
			AND fu.organization_id::text = fuo.organization_id::text
		LEFT JOIN LATERAL (
			SELECT COALESCE(SUM(COALESCE(t.total_amount / NULLIF(q.total_qty, 0), 0)), 0) AS total_revenue
			FROM (
				SELECT DISTINCT sf.order_id::text AS order_id
				FROM schedule_fleets sf
				INNER JOIN fleet_unit_ownership fuo2 ON fuo2.unit_id::text = sf.unit_id::text
					AND fuo2.organization_id::text = sf.organization_id::text
				WHERE fuo2.partner_id::text = op.partner_id::text
				  AND sf.organization_id::text = op.organization_id::text
			) sf
			INNER JOIN (
				SELECT reference_id::text AS order_id, SUM(amount) AS total_amount
				FROM transactions
				WHERE %s
				  AND transaction_type = 1
				GROUP BY reference_id::text
			) t ON t.order_id = sf.order_id
			INNER JOIN (
				SELECT order_id::text AS order_id, SUM(quantity) AS total_qty
				FROM fleet_order_items
				GROUP BY order_id::text
			) q ON q.order_id = sf.order_id
		) revenue ON true
		WHERE %s
		GROUP BY
			op.partner_id,
			op.partner_name,
			op.partner_address,
			op.partner_city,
			op.partner_phone,
			op.partner_email,
			op.pic_name,
			op.created_at,
			op.created_by,
			op.updated_at,
			op.updated_by,
			op.organization_id,
			revenue.total_revenue
		ORDER BY op.created_at DESC
	`,
		strings.Join(revenueDateClauses, "\n\t\t\t\t  AND "),
		strings.Join(whereClauses, "\n\t\t  AND "),
	)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.OperationPartner
	for rows.Next() {
		var p model.OperationPartner
		err := rows.Scan(
			&p.PartnerID, &p.PartnerName, &p.PartnerAddress, &p.PartnerCity, &p.PartnerPhone, &p.PartnerEmail, &p.PicName,
			&p.CreatedAt,

			&p.OrganizationID, &p.TotalUnit, &p.TotalRevenue,
		)
		if err != nil {
			return nil, err
		}
		p.PartnerCityLabel = r.GetCityLabel(p.PartnerCity)
		result = append(result, p)
	}
	return result, nil
}

func (r *PartnerRepository) Create(req model.CreateOperationPartnerRequest, orgID, userID string) (*model.OperationPartner, error) {
	partnerID := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO operation_partner (partner_id, partner_name, partner_address, partner_city, partner_phone, partner_email, pic_name, created_at, created_by, updated_at, updated_by, organization_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	if r.driver == "mysql" {
		query = strings.ReplaceAll(query, "$1", "?")
		query = strings.ReplaceAll(query, "$2", "?")
		query = strings.ReplaceAll(query, "$3", "?")
		query = strings.ReplaceAll(query, "$4", "?")
		query = strings.ReplaceAll(query, "$5", "?")
		query = strings.ReplaceAll(query, "$6", "?")
		query = strings.ReplaceAll(query, "$7", "?")
		query = strings.ReplaceAll(query, "$8", "?")
		query = strings.ReplaceAll(query, "$9", "?")
		query = strings.ReplaceAll(query, "$10", "?")
		query = strings.ReplaceAll(query, "$11", "?")
		query = strings.ReplaceAll(query, "$12", "?")
	}

	_, err := r.db.Exec(query, partnerID, req.PartnerName, req.PartnerAddress, req.PartnerCity, req.PartnerPhone, req.PartnerEmail, req.PicName, now, userID, now, userID, orgID)
	if err != nil {
		return nil, err
	}

	return r.GetByID(partnerID, orgID, nil)
}

func (r *PartnerRepository) Update(req model.UpdateOperationPartnerRequest, orgID, userID string) (*model.OperationPartner, error) {
	now := time.Now()

	query := `
		UPDATE operation_partner
		SET partner_name = $1, partner_address = $2, partner_city = $3, partner_phone = $4, partner_email = $5, pic_name = $6, updated_at = $7, updated_by = $8
		WHERE partner_id = $9 AND organization_id = $10
	`
	if r.driver == "mysql" {
		query = strings.ReplaceAll(query, "$1", "?")
		query = strings.ReplaceAll(query, "$2", "?")
		query = strings.ReplaceAll(query, "$3", "?")
		query = strings.ReplaceAll(query, "$4", "?")
		query = strings.ReplaceAll(query, "$5", "?")
		query = strings.ReplaceAll(query, "$6", "?")
		query = strings.ReplaceAll(query, "$7", "?")
		query = strings.ReplaceAll(query, "$8", "?")
		query = strings.ReplaceAll(query, "$9", "?")
		query = strings.ReplaceAll(query, "$10", "?")
	}

	_, err := r.db.Exec(query, req.PartnerName, req.PartnerAddress, req.PartnerCity, req.PartnerPhone, req.PartnerEmail, req.PartnerPic, now, userID, req.PartnerID, orgID)
	if err != nil {
		return nil, err
	}

	return r.GetByID(req.PartnerID, orgID, nil)
}

func (r *PartnerRepository) GetByID(partnerID, orgID string, filter *model.OperationPartnerDetailRequest) (*model.OperationPartner, error) {
	args := make([]interface{}, 0, 4)
	args = append(args, partnerID, orgID)

	tripCond := ""

	if filter != nil {
		if v := strings.TrimSpace(filter.TripStartDate); v != "" {
			tripCond += fmt.Sprintf(" AND fo.end_date >= %s", r.getPlaceholder(len(args)+1))
			args = append(args, v)
		}
		if v := strings.TrimSpace(filter.TripEndDate); v != "" {
			tripCond += fmt.Sprintf(" AND fo.start_date <= %s", r.getPlaceholder(len(args)+1))
			args = append(args, v)
		}
	}

	query := fmt.Sprintf(`
		SELECT 
			op.partner_name, 
			op.partner_address, 
			op.partner_city, 
			op.partner_phone, 
			op.pic_name, 
			op.partner_email, 
			op.created_at AS join_date, 
			
			(SELECT COUNT(fuo.unit_id) 
			 FROM fleet_unit_ownership fuo 
			 WHERE fuo.partner_id = op.partner_id AND fuo.organization_id = op.organization_id) AS total_units, 
			  
			(SELECT COUNT(sf.uuid) 
			 FROM fleet_units fu 
			 INNER JOIN schedule_fleets sf ON sf.unit_id = fu.unit_id 
			 INNER JOIN fleet_unit_ownership fuo ON fuo.unit_id = fu.unit_id 
			 INNER JOIN fleet_orders fo ON fo.order_id = sf.order_id
			 WHERE fuo.partner_id = op.partner_id
			   AND fuo.organization_id = op.organization_id
			   AND sf.organization_id = op.organization_id
			   %s) AS total_schedule
		FROM operation_partner op 
		WHERE op.partner_id = %s AND op.organization_id = %s
		LIMIT 1
	`,
		tripCond,
		r.getPlaceholder(1), r.getPlaceholder(2),
	)

	var p model.OperationPartner
	var joinDate time.Time
	err := r.db.QueryRow(query, args...).Scan(
		&p.PartnerName,
		&p.PartnerAddress,
		&p.PartnerCity,
		&p.PartnerPhone,
		&p.PicName,
		&p.PartnerEmail,
		&joinDate,
		&p.TotalUnits,
		&p.TotalSchedule,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	p.PartnerID = partnerID
	p.OrganizationID = &orgID
	p.JoinDate = &joinDate
	p.CreatedAt = &joinDate
	p.TotalUnit = p.TotalUnits
	p.PartnerCityLabel = r.GetCityLabel(p.PartnerCity)
	return &p, nil
}

func (r *PartnerRepository) GetDetailMetrics(partnerID, orgID string, req *model.OperationPartnerDetailRequest) (float64, float64, int64, error) {
	transactionStartDate := "0001-01-01"
	transactionEndDate := "9999-12-31"
	tripStartDate := "0001-01-01"
	tripEndDate := "9999-12-31"

	if req != nil {
		if v := strings.TrimSpace(req.TransactionStartDate); v != "" {
			transactionStartDate = v
		}
		if v := strings.TrimSpace(req.TransactionEndDate); v != "" {
			transactionEndDate = v
		}
		if v := strings.TrimSpace(req.TripStartDate); v != "" {
			tripStartDate = v
		}
		if v := strings.TrimSpace(req.TripEndDate); v != "" {
			tripEndDate = v
		}
	}

	parseFloat64 := func(v interface{}) (float64, bool) {
		switch vv := v.(type) {
		case nil:
			return 0, true
		case float64:
			return vv, true
		case float32:
			return float64(vv), true
		case int64:
			return float64(vv), true
		case int32:
			return float64(vv), true
		case int:
			return float64(vv), true
		case []byte:
			f, err := strconv.ParseFloat(string(vv), 64)
			return f, err == nil
		case string:
			f, err := strconv.ParseFloat(vv, 64)
			return f, err == nil
		default:
			return 0, false
		}
	}

	parseInt64 := func(v interface{}) (int64, bool) {
		switch vv := v.(type) {
		case nil:
			return 0, true
		case int64:
			return vv, true
		case int32:
			return int64(vv), true
		case int:
			return int64(vv), true
		case float64:
			return int64(vv), true
		case float32:
			return int64(vv), true
		case []byte:
			i, err := strconv.ParseInt(string(vv), 10, 64)
			return i, err == nil
		case string:
			i, err := strconv.ParseInt(vv, 10, 64)
			return i, err == nil
		default:
			return 0, false
		}
	}

	sfOrderExpr := "sf.order_id::text"
	transactionReferenceExpr := "reference_id::text"
	fleetOrderItemExpr := "order_id::text"
	revenuePartnerExpr := "fuo.partner_id::text = " + r.getPlaceholder(1)
	revenueOrgExpr := "sf.organization_id::text = " + r.getPlaceholder(2)
	expensesPartnerExpr := "fuo.partner_id::text = " + r.getPlaceholder(5)
	expensesOrgExpr := "sf.organization_id::text = " + r.getPlaceholder(6)
	expensesReferenceExpr := "t.reference_id::text = sf.schedule_number::text OR t.reference_id::text = sf.order_id::text"
	totalBookingPartnerExpr := "fuo.partner_id::text = " + r.getPlaceholder(9)
	totalBookingOrgExpr := "sf.organization_id::text = " + r.getPlaceholder(10)
	totalBookingFleetOrderOrgExpr := "fo2.organization_id::text = " + r.getPlaceholder(11)

	if r.driver == "mysql" {
		sfOrderExpr = "sf.order_id"
		transactionReferenceExpr = "reference_id"
		fleetOrderItemExpr = "order_id"
		revenuePartnerExpr = "fuo.partner_id = " + r.getPlaceholder(1)
		revenueOrgExpr = "sf.organization_id = " + r.getPlaceholder(2)
		expensesPartnerExpr = "fuo.partner_id = " + r.getPlaceholder(5)
		expensesOrgExpr = "sf.organization_id = " + r.getPlaceholder(6)
		expensesReferenceExpr = "t.reference_id = sf.schedule_number OR t.reference_id = sf.order_id"
		totalBookingPartnerExpr = "fuo.partner_id = " + r.getPlaceholder(9)
		totalBookingOrgExpr = "sf.organization_id = " + r.getPlaceholder(10)
		totalBookingFleetOrderOrgExpr = "fo2.organization_id = " + r.getPlaceholder(11)
	}

	query := fmt.Sprintf(`
		SELECT
			(
				SELECT COALESCE(SUM(COALESCE(t.total_amount / NULLIF(q.total_qty, 0), 0)), 0)
				FROM (
					SELECT DISTINCT %s AS order_id
					FROM schedule_fleets sf
					INNER JOIN fleet_unit_ownership fuo ON fuo.unit_id = sf.unit_id
					WHERE %s
					  AND %s
				) sf
				INNER JOIN (
					SELECT %s AS order_id, SUM(amount) AS total_amount
					FROM transactions
					WHERE transaction_date IS NOT NULL
					  AND transaction_date >= %s AND transaction_date < %s
					  AND transaction_type = 1
					GROUP BY %s
				) t ON t.order_id = sf.order_id
				INNER JOIN (
					SELECT %s AS order_id, SUM(quantity) AS total_qty
					FROM fleet_order_items
					GROUP BY %s
				) q ON q.order_id = sf.order_id
			) AS revenue,
			(
				SELECT COALESCE(SUM(t.amount), 0)
				FROM schedule_fleets sf
				INNER JOIN fleet_unit_ownership fuo ON fuo.unit_id = sf.unit_id
				INNER JOIN transactions t ON (%s)
				WHERE %s
				  AND %s
				  AND t.transaction_type = 2
				  AND t.transaction_date IS NOT NULL
				  AND t.transaction_date >= %s AND t.transaction_date < %s
			) AS expenses,
			(
				SELECT COALESCE(COUNT(DISTINCT sf.schedule_number), 0)
				FROM schedule_fleets sf
				INNER JOIN fleet_orders fo2 ON fo2.order_id = sf.order_id
				INNER JOIN fleet_unit_ownership fuo ON fuo.unit_id = sf.unit_id
				WHERE %s
				  AND %s
				  AND %s
				  AND fo2.status = 1
				  AND fo2.start_date >= %s
				  AND fo2.end_date < %s
			) AS total_booking
	`,
		sfOrderExpr,
		revenuePartnerExpr,
		revenueOrgExpr,
		transactionReferenceExpr,
		r.getPlaceholder(3),
		r.getPlaceholder(4),
		transactionReferenceExpr,
		fleetOrderItemExpr,
		fleetOrderItemExpr,
		expensesReferenceExpr,
		expensesPartnerExpr,
		expensesOrgExpr,
		r.getPlaceholder(7),
		r.getPlaceholder(8),
		totalBookingPartnerExpr,
		totalBookingOrgExpr,
		totalBookingFleetOrderOrgExpr,
		r.getPlaceholder(12),
		r.getPlaceholder(13),
	)

	var totalRevenueAny interface{}
	var totalExpensesAny interface{}
	var totalBookingAny interface{}

	err := database.QueryRow(
		r.db,
		query,
		partnerID,
		orgID,
		transactionStartDate,
		transactionEndDate,
		partnerID,
		orgID,
		transactionStartDate,
		transactionEndDate,
		partnerID,
		orgID,
		orgID,
		tripStartDate,
		tripEndDate,
	).Scan(&totalRevenueAny, &totalExpensesAny, &totalBookingAny)
	if err != nil {
		return 0, 0, 0, err
	}

	totalRevenue, ok := parseFloat64(totalRevenueAny)
	if !ok {
		totalRevenue = 0
	}

	totalExpenses, ok := parseFloat64(totalExpensesAny)
	if !ok {
		totalExpenses = 0
	}

	totalBooking, ok := parseInt64(totalBookingAny)
	if !ok {
		totalBooking = 0
	}

	return totalRevenue, totalExpenses, totalBooking, nil
}

func (r *PartnerRepository) GetOrCreateByNamePhone(orgID, userID, partnerName, partnerPhone string, partnerEmail *string) (string, error) {
	query := `
		SELECT partner_id
		FROM operation_partner
		WHERE partner_name = $1 AND partner_phone = $2 AND organization_id = $3
	`
	if r.driver == "mysql" {
		query = strings.ReplaceAll(query, "$1", "?")
		query = strings.ReplaceAll(query, "$2", "?")
		query = strings.ReplaceAll(query, "$3", "?")
	}

	var partnerID string
	err := r.db.QueryRow(query, partnerName, partnerPhone, orgID).Scan(&partnerID)
	if err == nil {
		return partnerID, nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	createReq := model.CreateOperationPartnerRequest{
		PartnerName:  partnerName,
		PartnerPhone: partnerPhone,
		PartnerEmail: partnerEmail,
		PicName:      partnerName,
	}

	partner, err := r.Create(createReq, orgID, userID)
	if err != nil {
		return "", err
	}

	return partner.PartnerID, nil
}

func (r *PartnerRepository) GetPartnerFleetUnits(partnerID, orgID string, req *model.OperationPartnerDetailRequest) ([]model.PartnerFleetUnit, error) {
	transactionStartDate := "0001-01-01"
	transactionEndDate := "9999-12-31"
	tripStartDate := "0001-01-01"
	tripEndDate := "9999-12-31"

	if req != nil {
		if v := strings.TrimSpace(req.TransactionStartDate); v != "" {
			transactionStartDate = v
		}
		if v := strings.TrimSpace(req.TransactionEndDate); v != "" {
			transactionEndDate = v
		}
		if v := strings.TrimSpace(req.TripStartDate); v != "" {
			tripStartDate = v
		}
		if v := strings.TrimSpace(req.TripEndDate); v != "" {
			tripEndDate = v
		}
	}

	parseFloat64 := func(v interface{}) (float64, bool) {
		switch vv := v.(type) {
		case nil:
			return 0, true
		case float64:
			return vv, true
		case float32:
			return float64(vv), true
		case int64:
			return float64(vv), true
		case int32:
			return float64(vv), true
		case int:
			return float64(vv), true
		case []byte:
			f, err := strconv.ParseFloat(string(vv), 64)
			return f, err == nil
		case string:
			f, err := strconv.ParseFloat(vv, 64)
			return f, err == nil
		default:
			return 0, false
		}
	}

	parseInt64 := func(v interface{}) (int64, bool) {
		switch vv := v.(type) {
		case nil:
			return 0, true
		case int64:
			return vv, true
		case int32:
			return int64(vv), true
		case int:
			return int64(vv), true
		case float64:
			return int64(vv), true
		case float32:
			return int64(vv), true
		case []byte:
			i, err := strconv.ParseInt(string(vv), 10, 64)
			return i, err == nil
		case string:
			i, err := strconv.ParseInt(vv, 10, 64)
			return i, err == nil
		default:
			return 0, false
		}
	}

	fleetOrderOrgExpr := "fo.organization_id::text = fuo.organization_id::text"
	revenueOrderExpr := "sf.order_id::text"
	revenueReferenceExpr := "reference_id::text"
	revenueFleetOrderItemExpr := "order_id::text"
	expenseReferenceExpr := "t.reference_id::text IN (sf.schedule_number::text, sf.order_id::text)"
	partnerExpr := "fuo.partner_id::text = " + r.getPlaceholder(1)
	orgExpr := "fuo.organization_id::text = " + r.getPlaceholder(2)
	bookingStartExpr := r.getPlaceholder(3)
	bookingEndExpr := r.getPlaceholder(4)
	revenueStartExpr := r.getPlaceholder(5)
	revenueEndExpr := r.getPlaceholder(6)
	expenseStartExpr := r.getPlaceholder(7)
	expenseEndExpr := r.getPlaceholder(8)

	query := fmt.Sprintf(`
		SELECT
			f.fleet_name,
			ft.label AS fleet_type,
			fu.plate_number,
			fu.vehicle_id,
			fu.unit_id,
			COALESCE(booking.total_booking, 0) AS total_booking,
			COALESCE(revenue.total_revenue, 0) AS total_revenue,
			COALESCE(expenses.total_expenses, 0) AS total_expenses
		FROM fleets f
		INNER JOIN fleet_units fu ON fu.fleet_id = f.uuid
		INNER JOIN (
			SELECT DISTINCT unit_id, partner_id, organization_id
			FROM fleet_unit_ownership
		) fuo ON fuo.unit_id = fu.unit_id
		INNER JOIN fleet_types ft ON ft.id = f.fleet_type
		LEFT JOIN LATERAL (
			SELECT COALESCE(COUNT(DISTINCT sf.schedule_number), 0) AS total_booking
			FROM schedule_fleets sf
			INNER JOIN fleet_orders fo ON fo.order_id = sf.order_id
			WHERE sf.unit_id = fu.unit_id
			  AND sf.organization_id = fuo.organization_id
			  AND %s
			  AND fo.status = 1
			  AND fo.start_date >= %s
			  AND fo.end_date < %s
		) booking ON true
		LEFT JOIN LATERAL (
			SELECT COALESCE(SUM(COALESCE(t.total_amount / NULLIF(q.total_qty, 0), 0)), 0) AS total_revenue
			FROM (
				SELECT DISTINCT %s AS order_id
				FROM schedule_fleets sf
				WHERE sf.unit_id = fu.unit_id
				  AND sf.organization_id = fuo.organization_id
			) sf
			INNER JOIN (
				SELECT %s AS order_id, SUM(amount) AS total_amount
				FROM transactions
				WHERE transaction_date IS NOT NULL
				  AND transaction_date >= %s AND transaction_date < %s
				  AND transaction_type = 1
				GROUP BY %s
			) t ON t.order_id = sf.order_id
			INNER JOIN (
				SELECT %s AS order_id, SUM(quantity) AS total_qty
				FROM fleet_order_items
				GROUP BY %s
			) q ON q.order_id = sf.order_id
		) revenue ON true
		LEFT JOIN LATERAL (
			SELECT COALESCE(SUM(t.amount), 0) AS total_expenses
			FROM schedule_fleets sf
			INNER JOIN transactions t ON %s
			WHERE sf.unit_id = fu.unit_id
			  AND sf.organization_id = fuo.organization_id
			  AND t.transaction_type = 2
			  AND t.transaction_date IS NOT NULL
			  AND t.transaction_date >= %s AND t.transaction_date < %s
		) expenses ON true
		WHERE %s AND %s
	`,
		fleetOrderOrgExpr,
		bookingStartExpr,
		bookingEndExpr,
		revenueOrderExpr,
		revenueReferenceExpr,
		revenueStartExpr,
		revenueEndExpr,
		revenueReferenceExpr,
		revenueFleetOrderItemExpr,
		revenueFleetOrderItemExpr,
		expenseReferenceExpr,
		expenseStartExpr,
		expenseEndExpr,
		partnerExpr,
		orgExpr,
	)

	rows, err := r.db.Query(query, partnerID, orgID, tripStartDate, tripEndDate, transactionStartDate, transactionEndDate, transactionStartDate, transactionEndDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.PartnerFleetUnit
	for rows.Next() {
		var fu model.PartnerFleetUnit
		var totalBookingAny interface{}
		var totalRevenueAny interface{}
		var totalExpensesAny interface{}
		err := rows.Scan(&fu.FleetName, &fu.FleetType, &fu.PlateNumber, &fu.VehicleID, &fu.UnitID, &totalBookingAny, &totalRevenueAny, &totalExpensesAny)
		if err != nil {
			return nil, err
		}
		totalBooking, ok := parseInt64(totalBookingAny)
		if !ok {
			totalBooking = 0
		}
		totalRevenue, ok := parseFloat64(totalRevenueAny)
		if !ok {
			totalRevenue = 0
		}
		totalExpenses, ok := parseFloat64(totalExpensesAny)
		if !ok {
			totalExpenses = 0
		}
		fu.TotalBooking = totalBooking
		fu.TotalRevenue = totalRevenue
		fu.TotalExpenses = totalExpenses
		result = append(result, fu)
	}
	return result, nil
}
