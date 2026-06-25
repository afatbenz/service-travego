package repository

import (
	"database/sql"
	"fmt"
	"os"
	"service-travego/database"
	"service-travego/utils"
	"strconv"
	"strings"
	"time"
)

type PrintManagementRepository struct {
	db     *sql.DB
	driver string
}

type PrintOrganizationInfo struct {
	OrganizationName string
	CompanyName      string
	CompanyAddress   string
	CompanyCity      string
	CompanyProvince  string
	CompanyPhone     string
	CompanyEmail     string
	CompanyPostal    string
	CompanyLogo      string
	CompanyWebsite   string
}

type PrintCustomerInfo struct {
	CustomerName    string
	CustomerCompany string
	CustomerAddress string
	CustomerCity    string
	CustomerPhone   string
	CustomerEmail   string
}

type PrintFleetOrderInfo struct {
	OrderID            string
	CreatedAt          time.Time
	StartDate          time.Time
	EndDate            time.Time
	PickupCityID       string
	PickupAddress      string
	AdditionalRequest  string
	TotalAmountInDB    float64
	InvoiceIDCandidate string
}

type PrintFleetOrderItem struct {
	OrderItemID      string
	FleetName        string
	FleetPrice       float64
	FleetQty         int
	FleetDiscount    float64
	AdditionalAmount float64
}

type PrintFleetOrderAddon struct {
	OrderItemID string
	AddonName   string
	AddonDesc   string
	AddonPrice  float64
}

type PrintOrganizationBank struct {
	BankCode        string
	BankAccount     string
	BankAccountName string
}

type PrintPaymentOrderInfo struct {
	InvoiceNumber   string
	PaymentType     int
	PaymentMethod   int
	PaymentAmount   float64
	RemainingAmount float64
	CreatedAt       time.Time
}

type PrintFleetTripExpense struct {
	TransactionItem string
	Description     string
	ExpenseAmount   float64
	ExpenseDate     time.Time
}

func NewPrintManagementRepository(db *sql.DB, driver string) *PrintManagementRepository {
	return &PrintManagementRepository{db: db, driver: driver}
}

func (r *PrintManagementRepository) placeholder(position int) string {
	if r.driver == "postgres" || r.driver == "pgx" {
		return "$" + strconv.Itoa(position)
	}
	return "?"
}

func (r *PrintManagementRepository) GetOrganizationInfo(organizationID string) (*PrintOrganizationInfo, error) {
	orgExpr := "organization_id = " + r.placeholder(1)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.placeholder(1)
	}
	query := fmt.Sprintf(`
		SELECT organization_name, company_name,
		       address as company_address, city as company_city, province as company_province,
		       phone as company_phone, email as company_email, postal_code as company_postal_code,
		       COALESCE(logo, '') as company_logo, COALESCE(domain_url, '') as company_website
		FROM organizations
		WHERE %s
		LIMIT 1
	`, orgExpr)

	var out PrintOrganizationInfo
	var companyName sql.NullString
	var companyAddress sql.NullString
	var companyCity sql.NullString
	var companyProvince sql.NullString
	var companyPhone sql.NullString
	var companyEmail sql.NullString
	var companyPostal sql.NullString
	var companyLogo sql.NullString
	var companyWebsite sql.NullString

	err := database.QueryRow(r.db, query, organizationID).Scan(
		&out.OrganizationName,
		&companyName,
		&companyAddress,
		&companyCity,
		&companyProvince,
		&companyPhone,
		&companyEmail,
		&companyPostal,
		&companyLogo,
		&companyWebsite,
	)
	if err != nil {
		return nil, err
	}
	if companyName.Valid {
		out.CompanyName = companyName.String
	}
	if companyAddress.Valid {
		out.CompanyAddress = companyAddress.String
	}
	if companyCity.Valid {
		out.CompanyCity = companyCity.String
	}
	if companyProvince.Valid {
		out.CompanyProvince = companyProvince.String
	}
	if companyPhone.Valid {
		out.CompanyPhone = companyPhone.String
	}
	if companyEmail.Valid {
		out.CompanyEmail = companyEmail.String
	}
	if companyPostal.Valid {
		out.CompanyPostal = companyPostal.String
	}
	if companyLogo.Valid {
		out.CompanyLogo = companyLogo.String
	}
	if companyWebsite.Valid {
		out.CompanyWebsite = companyWebsite.String
	}
	return &out, nil
}

func (r *PrintManagementRepository) GetCustomerInfo(orderID, organizationID string) (*PrintCustomerInfo, error) {
	orderExpr := "co.order_id = " + r.placeholder(1)
	orgExpr := "co.organization_id = " + r.placeholder(2)
	customerOrgExpr := "c.organization_id = co.organization_id"
	customerJoinExpr := "co.customer_id = c.customer_id"
	customerCityExpr := "COALESCE(c.customer_city, '')"
	if r.driver == "postgres" || r.driver == "pgx" {
		orderExpr = "co.order_id::text = " + r.placeholder(1)
		orgExpr = "co.organization_id::text = " + r.placeholder(2)
		customerOrgExpr = "c.organization_id::text = co.organization_id::text"
		customerJoinExpr = "co.customer_id::text = c.customer_id::text"
		customerCityExpr = "COALESCE(c.customer_city::text, '')"
	} else if r.driver == "mysql" {
		customerCityExpr = "COALESCE(CAST(c.customer_city AS CHAR), '')"
	}

	query := fmt.Sprintf(`
		SELECT COALESCE(c.customer_name, '') as customer_name,
		       COALESCE(c.customer_address, '') as customer_address,
		       %s as customer_city,
		       COALESCE(c.customer_phone, '') as customer_phone,
		       COALESCE(c.company_name, '') as customer_company,
		       COALESCE(c.customer_email, '') as customer_email
		FROM customer_orders co
		INNER JOIN customers c ON %s AND %s
		WHERE %s AND %s
		LIMIT 1
	`, customerCityExpr, customerJoinExpr, customerOrgExpr, orderExpr, orgExpr)

	var out PrintCustomerInfo
	var customerCity sql.NullString
	if err := database.QueryRow(r.db, query, orderID, organizationID).Scan(
		&out.CustomerName,
		&out.CustomerAddress,
		&customerCity,
		&out.CustomerPhone,
		&out.CustomerCompany,
		&out.CustomerEmail,
	); err != nil {
		return nil, err
	}
	if customerCity.Valid {
		out.CustomerCity = customerCity.String
	}
	return &out, nil
}

func (r *PrintManagementRepository) GetPaymentOrderForInvoice(organizationID, orderID string, invoiceNumber *string) (*PrintPaymentOrderInfo, error) {
	orgExpr := "organization_id = " + r.placeholder(1)
	orderExpr := "order_id = " + r.placeholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.placeholder(1)
		orderExpr = "order_id::text = " + r.placeholder(2)
	}

	args := []interface{}{organizationID, orderID}
	where := fmt.Sprintf("WHERE %s AND %s AND COALESCE(status, 0) > 0", orgExpr, orderExpr)

	if invoiceNumber != nil {
		invExpr := "invoice_number = " + r.placeholder(3)
		if r.driver == "postgres" || r.driver == "pgx" {
			invExpr = "invoice_number::text = " + r.placeholder(3)
		}
		where += " AND " + invExpr
		args = append(args, strings.TrimSpace(*invoiceNumber))
	}

	query := fmt.Sprintf(`
		SELECT COALESCE(invoice_number, '') as invoice_number,
		       COALESCE(payment_type, 0) as payment_type,
		       COALESCE(payment_method, 0) as payment_method,
		       COALESCE(payment_amount, 0) as payment_amount,
		       COALESCE(remaining_amount, 0) as remaining_amount,
		       created_at
		FROM payment_orders
		%s
		ORDER BY created_at DESC
		LIMIT 1
	`, where)

	var out PrintPaymentOrderInfo
	var inv sql.NullString
	if err := database.QueryRow(r.db, query, args...).Scan(
		&inv,
		&out.PaymentType,
		&out.PaymentMethod,
		&out.PaymentAmount,
		&out.RemainingAmount,
		&out.CreatedAt,
	); err != nil {
		return nil, err
	}
	if inv.Valid {
		out.InvoiceNumber = inv.String
	}
	return &out, nil
}

func (r *PrintManagementRepository) GetFleetOrderInfo(orderID, organizationID string) (*PrintFleetOrderInfo, error) {
	orderExpr := "order_id = " + r.placeholder(1)
	orgExpr := "organization_id = " + r.placeholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		orderExpr = "order_id::text = " + r.placeholder(1)
		orgExpr = "organization_id::text = " + r.placeholder(2)
	}
	query := fmt.Sprintf(`
		SELECT order_id, created_at, start_date, end_date,
		       pickup_city_id, pickup_location as pickup_address,
		       COALESCE(additional_request, '') as additional_request,
		       COALESCE(total_amount, 0) as total_amount
		FROM fleet_orders
		WHERE %s AND %s
		LIMIT 1
	`, orderExpr, orgExpr)

	var out PrintFleetOrderInfo
	var pickupCityID sql.NullString
	var pickupAddress sql.NullString
	var additionalRequest sql.NullString
	if err := database.QueryRow(r.db, query, orderID, organizationID).Scan(
		&out.OrderID,
		&out.CreatedAt,
		&out.StartDate,
		&out.EndDate,
		&pickupCityID,
		&pickupAddress,
		&additionalRequest,
		&out.TotalAmountInDB,
	); err != nil {
		return nil, err
	}
	if pickupCityID.Valid {
		out.PickupCityID = pickupCityID.String
	}
	if pickupAddress.Valid {
		out.PickupAddress = pickupAddress.String
	}
	if additionalRequest.Valid {
		out.AdditionalRequest = additionalRequest.String
	}
	out.InvoiceIDCandidate = out.OrderID
	return &out, nil
}

func (r *PrintManagementRepository) GetFleetOrderItems(orderID, organizationID string) ([]PrintFleetOrderItem, error) {
	orderExpr := "fo.order_id = " + r.placeholder(1)
	orgExpr := "fo.organization_id = " + r.placeholder(2)
	fleetJoinExpr := "f.uuid = fo.fleet_id"
	priceJoinExpr := "fp.uuid = fo.price_id"
	orderItemIDExpr := "COALESCE(fo.order_item_id, '')"
	if r.driver == "postgres" || r.driver == "pgx" {
		orderExpr = "fo.order_id::text = " + r.placeholder(1)
		orgExpr = "fo.organization_id::text = " + r.placeholder(2)
		fleetJoinExpr = "f.uuid::text = fo.fleet_id::text"
		priceJoinExpr = "fp.uuid::text = fo.price_id::text"
		orderItemIDExpr = "COALESCE(fo.order_item_id::text, '')"
	}

	query := fmt.Sprintf(`
		SELECT %s as order_item_id,
		       COALESCE(f.fleet_name, '') as fleet_name,
		       COALESCE(fp.price, 0) as fleet_price,
		       COALESCE(fo.quantity, 0) as fleet_qty,
		       COALESCE(fo.discount, 0) as fleet_discount,
		       COALESCE(fo.charge_amount, 0) as additional_amount
		FROM fleet_order_items fo
		LEFT JOIN fleets f ON %s
		LEFT JOIN fleet_prices fp ON %s
		WHERE %s AND %s
		ORDER BY COALESCE(f.fleet_name, '') ASC
	`, orderItemIDExpr, fleetJoinExpr, priceJoinExpr, orderExpr, orgExpr)

	if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
		fmt.Println("GetFleetOrderItems query:", query)
		fmt.Println("GetFleetOrderItems args:", orderID, organizationID)
	}

	rows, err := database.Query(r.db, query, orderID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PrintFleetOrderItem
	for rows.Next() {
		var it PrintFleetOrderItem
		if err := rows.Scan(&it.OrderItemID, &it.FleetName, &it.FleetPrice, &it.FleetQty, &it.FleetDiscount, &it.AdditionalAmount); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if len(items) > 0 {
		return items, nil
	}

	orderExpr = "fo.order_id = " + r.placeholder(1)
	orgExpr = "fo.organization_id = " + r.placeholder(2)
	fleetJoinExpr = "f.uuid = fo.fleet_id"
	priceJoinExpr = "fp.uuid = fo.price_id"
	if r.driver == "postgres" || r.driver == "pgx" {
		orderExpr = "fo.order_id::text = " + r.placeholder(1)
		orgExpr = "fo.organization_id::text = " + r.placeholder(2)
		fleetJoinExpr = "f.uuid::text = fo.fleet_id::text"
		priceJoinExpr = "fp.uuid::text = fo.price_id::text"
	}

	fallbackQuery := fmt.Sprintf(`
		SELECT
			COALESCE(f.fleet_name, '') as fleet_name,
			COALESCE(fp.price, 0) as fleet_price,
			COALESCE(fo.unit_qty, 0) as fleet_qty,
			COALESCE(fo.additional_amount, 0) as additional_amount
		FROM fleet_orders fo
		LEFT JOIN fleets f ON %s
		LEFT JOIN fleet_prices fp ON %s
		WHERE %s AND %s
		LIMIT 1
	`, fleetJoinExpr, priceJoinExpr, orderExpr, orgExpr)

	if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
		fmt.Println("GetFleetOrderItems fallback query:", fallbackQuery)
		fmt.Println("GetFleetOrderItems fallback args:", orderID, organizationID)
	}

	var it PrintFleetOrderItem
	it.OrderItemID = strings.TrimSpace(orderID)
	it.FleetDiscount = 0
	if err := database.QueryRow(r.db, fallbackQuery, orderID, organizationID).Scan(&it.FleetName, &it.FleetPrice, &it.FleetQty, &it.AdditionalAmount); err != nil {
		return nil, err
	}
	return []PrintFleetOrderItem{it}, nil
}

func (r *PrintManagementRepository) GetFleetOrderAddons(orderID, organizationID string) ([]PrintFleetOrderAddon, error) {
	orgExpr := "oi.organization_id = " + r.placeholder(1)
	orderExpr := "oi.order_id = " + r.placeholder(2)
	addonJoinExpr := "fa.uuid = foa.addon_id"
	itemJoinExpr := "oi.order_item_id = foa.order_item_id"
	orderItemIDExpr := "COALESCE(foa.order_item_id, '')"
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "oi.organization_id::text = " + r.placeholder(1)
		orderExpr = "oi.order_id::text = " + r.placeholder(2)
		addonJoinExpr = "fa.uuid::text = foa.addon_id::text"
		itemJoinExpr = "oi.order_item_id::text = foa.order_item_id::text"
		orderItemIDExpr = "COALESCE(foa.order_item_id::text, '')"
	}

	query := fmt.Sprintf(`
		SELECT %s as order_item_id,
		       COALESCE(fa.addon_name, '') as addon_name,
		       COALESCE(fa.addon_desc, '') as addon_desc,
		       COALESCE(foa.addon_price, 0) as addon_price
		FROM fleet_order_addons foa
		INNER JOIN fleet_order_items oi ON %s
		INNER JOIN fleet_addon fa ON %s
		WHERE %s AND %s
		ORDER BY fa.addon_name ASC
	`, orderItemIDExpr, itemJoinExpr, addonJoinExpr, orgExpr, orderExpr)

	if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
		fmt.Println("GetFleetOrderAddons query:", query)
		fmt.Println("GetFleetOrderAddons args:", organizationID, orderID)
	}

	rows, err := database.Query(r.db, query, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PrintFleetOrderAddon
	for rows.Next() {
		var it PrintFleetOrderAddon
		if err := rows.Scan(&it.OrderItemID, &it.AddonName, &it.AddonDesc, &it.AddonPrice); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if len(items) > 0 {
		return items, nil
	}

	orderExpr = "foa.order_id = " + r.placeholder(1)
	addonJoinExpr = "fa.uuid = foa.addon_id"
	if r.driver == "postgres" || r.driver == "pgx" {
		orderExpr = "foa.order_id::text = " + r.placeholder(1)
		addonJoinExpr = "fa.uuid::text = foa.addon_id::text"
	}

	fallbackQuery := fmt.Sprintf(`
		SELECT
			COALESCE(fa.addon_name, '') as addon_name,
			COALESCE(fa.addon_desc, '') as addon_desc,
			COALESCE(foa.addon_price, 0) as addon_price
		FROM fleet_order_addons foa
		INNER JOIN fleet_addon fa ON %s
		WHERE %s
		ORDER BY fa.addon_name ASC
	`, addonJoinExpr, orderExpr)

	if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
		fmt.Println("GetFleetOrderAddons fallback query:", fallbackQuery)
		fmt.Println("GetFleetOrderAddons fallback args:", orderID)
		_ = organizationID
	}

	fRows, err := database.Query(r.db, fallbackQuery, orderID)
	if err != nil {
		return nil, err
	}
	defer fRows.Close()

	fallbackItems := make([]PrintFleetOrderAddon, 0)
	for fRows.Next() {
		var it PrintFleetOrderAddon
		it.OrderItemID = strings.TrimSpace(orderID)
		if err := fRows.Scan(&it.AddonName, &it.AddonDesc, &it.AddonPrice); err != nil {
			return nil, err
		}
		fallbackItems = append(fallbackItems, it)
	}
	if err := fRows.Err(); err != nil {
		return nil, err
	}
	return fallbackItems, nil
}

func (r *PrintManagementRepository) GetOrganizationBankAccount(organizationID string) (*PrintOrganizationBank, error) {
	orgExpr := "organization_id = " + r.placeholder(1)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.placeholder(1)
	}
	query := fmt.Sprintf(`
		SELECT COALESCE(bank_code, '') as bank_code,
		       COALESCE(account_number, '') as bank_account,
		       COALESCE(account_name, '') as bank_account_name
		FROM organization_bank_accounts
		WHERE %s
		ORDER BY created_at ASC
		LIMIT 1
	`, orgExpr)

	var out PrintOrganizationBank
	if err := database.QueryRow(r.db, query, organizationID).Scan(&out.BankCode, &out.BankAccount, &out.BankAccountName); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *PrintManagementRepository) GetOrderIDByScheduleNumber(scheduleNumber, organizationID string) (string, error) {
	snExpr := "schedule_number = " + r.placeholder(1)
	orgExpr := "organization_id = " + r.placeholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		snExpr = "schedule_number::text = " + r.placeholder(1)
		orgExpr = "organization_id::text = " + r.placeholder(2)
	}
	query := fmt.Sprintf(`SELECT COALESCE(order_id, '') as order_id FROM schedule_fleets WHERE %s AND %s LIMIT 1`, snExpr, orgExpr)
	var out sql.NullString
	if err := database.QueryRow(r.db, query, strings.TrimSpace(scheduleNumber), strings.TrimSpace(organizationID)).Scan(&out); err != nil {
		return "", err
	}
	if out.Valid {
		if v := strings.TrimSpace(out.String); v != "" {
			return v, nil
		}
	}
	return "", sql.ErrNoRows
}

func (r *PrintManagementRepository) GetFleetTripTotals(scheduleNumber, organizationID, referenceID string) (float64, float64, error) {
	snExpr := "schedule_number = " + r.placeholder(1)
	orgExpr := "organization_id = " + r.placeholder(2)
	refExpr := "reference_id = " + r.placeholder(3)
	if r.driver == "postgres" || r.driver == "pgx" {
		snExpr = "schedule_number::text = " + r.placeholder(1)
		orgExpr = "organization_id::text = " + r.placeholder(2)
		refExpr = "reference_id::text = " + r.placeholder(3)
	}

	query := fmt.Sprintf(`
		SELECT
			SUM(CASE WHEN payment_type = 1 THEN amount ELSE 0 END) as total_expenses,
			SUM(CASE WHEN payment_type = 2 THEN amount ELSE 0 END) as total_reimburse
		FROM transaction_fleet_trips
		WHERE
			%s AND %s AND %s
			AND payment_type IN (1, 2)
	`, snExpr, orgExpr, refExpr)

	var totalExpenses sql.NullFloat64
	var totalReimburse sql.NullFloat64
	if err := database.QueryRow(r.db, query, strings.TrimSpace(scheduleNumber), strings.TrimSpace(organizationID), strings.TrimSpace(referenceID)).Scan(&totalExpenses, &totalReimburse); err != nil {
		return 0, 0, err
	}

	te := 0.0
	if totalExpenses.Valid {
		te = totalExpenses.Float64
	}
	tr := 0.0
	if totalReimburse.Valid {
		tr = totalReimburse.Float64
	}
	return te, tr, nil
}

func (r *PrintManagementRepository) GetFleetTripOperationalFee(scheduleNumber, organizationID string) (float64, error) {
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	if scheduleNumber == "" {
		return 0, nil
	}

	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(amount), 0) AS operational_fee
		FROM transactions
		WHERE reference_id = %s
	`, r.placeholder(1))

	var operationalFee sql.NullFloat64
	if err := database.QueryRow(r.db, query, scheduleNumber).Scan(&operationalFee); err != nil {
		return 0, err
	}
	if operationalFee.Valid {
		return operationalFee.Float64, nil
	}
	return 0, nil
}

func (r *PrintManagementRepository) GetFleetTripExpenseHistory(scheduleNumber, organizationID, referenceID string) ([]PrintFleetTripExpense, error) {
	snExpr := "schedule_number = " + r.placeholder(1)
	orgExpr := "organization_id = " + r.placeholder(2)
	refExpr := "reference_id = " + r.placeholder(3)
	itemExpr := "COALESCE(transaction_item, '')"
	if r.driver == "postgres" || r.driver == "pgx" {
		snExpr = "schedule_number::text = " + r.placeholder(1)
		orgExpr = "organization_id::text = " + r.placeholder(2)
		refExpr = "reference_id::text = " + r.placeholder(3)
		itemExpr = "COALESCE(transaction_item::text, '')"
	} else if r.driver == "mysql" {
		itemExpr = "COALESCE(CAST(transaction_item AS CHAR), '')"
	}

	query := fmt.Sprintf(`
		SELECT
			%s as transaction_item,
			COALESCE(description, '') as description,
			COALESCE(amount, 0) as expense_amount,
			created_at as expense_date
		FROM transaction_fleet_trips
		WHERE %s AND %s AND %s
		ORDER BY created_at ASC
	`, itemExpr, snExpr, orgExpr, refExpr)

	rows, err := database.Query(r.db, query, strings.TrimSpace(scheduleNumber), strings.TrimSpace(organizationID), strings.TrimSpace(referenceID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PrintFleetTripExpense, 0)
	for rows.Next() {
		var it PrintFleetTripExpense
		if err := rows.Scan(&it.TransactionItem, &it.Description, &it.ExpenseAmount, &it.ExpenseDate); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PrintManagementRepository) CountPaymentOrdersByOrganization(organizationID string) (int, error) {
	orgExpr := "organization_id = " + r.placeholder(1)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.placeholder(1)
	}
	query := fmt.Sprintf(`SELECT COUNT(1) FROM payment_orders WHERE %s AND COALESCE(status, 0) > 0`, orgExpr)
	var count int
	if err := database.QueryRow(r.db, query, organizationID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PrintManagementRepository) GenerateInvoiceNumber(orderType int, organizationID string, now time.Time) (string, error) {
	return utils.GenerateInvoiceNumber(r.db, r.driver, organizationID, orderType, now)
}

// GetSubscriptionDetailByInvoice retrieves subscription transaction details by invoice number
func (r *PrintManagementRepository) GetSubscriptionDetailByInvoice(invoiceNumber string) (transactionID string, packageID string, startDate time.Time, expiryDate time.Time, userID string, organizationID string, paymentMethod sql.NullString, createdAt time.Time, paymentAmount sql.NullFloat64, err error) {
	query := fmt.Sprintf("SELECT transaction_id, package_id, start_date, expiry_date, user_id, organization_id, payment_method, created_at, payment_amount FROM travego_transactions WHERE invoice_number = %s LIMIT 1", r.placeholder(1))
	var tID, pID, uID, oID sql.NullString
	var sDate, eDate, cDate sql.NullTime
	err = database.QueryRow(r.db, query, invoiceNumber).Scan(&tID, &pID, &sDate, &eDate, &uID, &oID, &paymentMethod, &cDate, &paymentAmount)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, "", "", sql.NullString{}, time.Time{}, sql.NullFloat64{}, err
	}
	return tID.String, pID.String, sDate.Time, eDate.Time, uID.String, oID.String, paymentMethod, cDate.Time, paymentAmount, nil
}
