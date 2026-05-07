package repository

import (
	"database/sql"
	"fmt"
	"service-travego/database"
	"strconv"
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
	FleetName        string
	FleetPrice       float64
	FleetQty         int
	FleetDiscount    float64
	AdditionalAmount float64
}

type PrintFleetOrderAddon struct {
	AddonName  string
	AddonDesc  string
	AddonPrice float64
}

type PrintOrganizationBank struct {
	BankCode        string
	BankAccount     string
	BankAccountName string
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
		       COALESCE(c.company_name, '') as customer_company
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
	); err != nil {
		return nil, err
	}
	if customerCity.Valid {
		out.CustomerCity = customerCity.String
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
	if r.driver == "postgres" || r.driver == "pgx" {
		orderExpr = "fo.order_id::text = " + r.placeholder(1)
		orgExpr = "fo.organization_id::text = " + r.placeholder(2)
		fleetJoinExpr = "f.uuid::text = fo.fleet_id::text"
		priceJoinExpr = "fp.uuid::text = fo.price_id::text"
	}

	query := fmt.Sprintf(`
		SELECT COALESCE(f.fleet_name, '') as fleet_name,
		       COALESCE(fp.price, 0) as fleet_price,
		       COALESCE(fo.quantity, 0) as fleet_qty,
		       COALESCE(fo.discount, 0) as fleet_discount,
		       COALESCE(fo.charge_amount, 0) as additional_amount
		FROM fleet_order_items fo
		INNER JOIN fleets f ON %s
		INNER JOIN fleet_prices fp ON %s
		WHERE %s AND %s
		ORDER BY f.fleet_name ASC
	`, fleetJoinExpr, priceJoinExpr, orderExpr, orgExpr)

	rows, err := database.Query(r.db, query, orderID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PrintFleetOrderItem
	for rows.Next() {
		var it PrintFleetOrderItem
		if err := rows.Scan(&it.FleetName, &it.FleetPrice, &it.FleetQty, &it.FleetDiscount, &it.AdditionalAmount); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *PrintManagementRepository) GetFleetOrderAddons(orderID, organizationID string) ([]PrintFleetOrderAddon, error) {
	orderExpr := "foa.order_id = " + r.placeholder(1)
	orgExpr := "foa.organization_id = " + r.placeholder(2)
	joinExpr := "fa.uuid = foa.addon_id"
	if r.driver == "postgres" || r.driver == "pgx" {
		orderExpr = "foa.order_id::text = " + r.placeholder(1)
		orgExpr = "foa.organization_id::text = " + r.placeholder(2)
		joinExpr = "fa.uuid::text = foa.addon_id::text"
	}

	query := fmt.Sprintf(`
		SELECT COALESCE(fa.addon_name, '') as addon_name,
		       COALESCE(fa.addon_desc, '') as addon_desc,
		       COALESCE(fa.addon_price, 0) as addon_price
		FROM fleet_order_addons foa
		INNER JOIN fleet_addon fa ON %s
		WHERE %s AND %s
		ORDER BY fa.addon_name ASC
	`, joinExpr, orderExpr, orgExpr)

	rows, err := database.Query(r.db, query, orderID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PrintFleetOrderAddon
	for rows.Next() {
		var it PrintFleetOrderAddon
		if err := rows.Scan(&it.AddonName, &it.AddonDesc, &it.AddonPrice); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
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
