package repository

import (
	"database/sql"
	"fmt"
	"service-travego/configs"
	"service-travego/model"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FleetRepository struct {
	db     *sql.DB
	driver string
}

func NewFleetRepository(db *sql.DB, driver string) *FleetRepository {
	return &FleetRepository{
		db:     db,
		driver: driver,
	}
}

func uuid2() string { return uuid.New().String() }

func (r *FleetRepository) ListFleets(req *model.ListFleetRequest) ([]model.FleetListItem, error) {
	base := `
        SELECT f.uuid AS fleet_id, ft.label AS fleet_type, f.fleet_name, f.capacity, f.engine, f.body, f.active, f.status, f.thumbnail
        FROM fleets f INNER JOIN fleet_types ft ON f.fleet_type = ft.id
    `
	where := make([]string, 0, 4)
	args := make([]interface{}, 0, 4)
	pos := 1
	if req.OrganizationID != "" {
		where = append(where, fmt.Sprintf("f.organization_id = %s", r.getPlaceholder(pos)))
		args = append(args, req.OrganizationID)
		pos++
	}
	if req.FleetType != "" {
		where = append(where, fmt.Sprintf("f.fleet_type = %s", r.getPlaceholder(pos)))
		args = append(args, req.FleetType)
		pos++
	}
	if req.FleetBody != "" {
		where = append(where, fmt.Sprintf("f.body = %s", r.getPlaceholder(pos)))
		args = append(args, req.FleetBody)
		pos++
	}
	if req.FleetEngine != "" {
		where = append(where, fmt.Sprintf("f.engine = %s", r.getPlaceholder(pos)))
		args = append(args, req.FleetEngine)
		pos++
	}
	if req.PickupLocation > 0 {
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM fleet_pickup fp WHERE fp.fleet_id = f.uuid AND fp.city_id = %s)", r.getPlaceholder(pos)))
		args = append(args, req.PickupLocation)
		pos++
	}
	query := base
	if len(where) > 0 {
		query = query + " WHERE " + strings.Join(where, " AND ")
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.FleetListItem
	for rows.Next() {
		var item model.FleetListItem
		if err := rows.Scan(&item.FleetID, &item.FleetType, &item.FleetName, &item.Capacity, &item.Engine, &item.Body, &item.Active, &item.Status, &item.Thumbnail); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *FleetRepository) CreateFleet(req *model.CreateFleetRequest) (string, error) {
	id := uuid2()
	now := time.Now()
	query := `
        INSERT INTO fleets (uuid, organization_id, fleet_type, fleet_name, capacity, description, engine, body, active, created_at, created_by, status)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9),
		r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12))

	// Status default 1 (Active/Draft?)
	_, err := r.db.Exec(query, id, req.OrganizationID, req.FleetType, req.FleetName, req.Capacity, req.Description,
		req.Engine, req.Body, true, now, req.CreatedBy, 1)

	if err != nil {
		return "", err
	}

	// Insert facilities
	if len(req.Facilities) > 0 {
		fQuery := fmt.Sprintf("INSERT INTO fleet_facilities (uuid, fleet_id, facility) VALUES (%s, %s, %s)",
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
		for _, fac := range req.Facilities {
			fID := uuid2()
			_, err := r.db.Exec(fQuery, fID, id, fac)
			if err != nil {
				return "", err
			}
		}
	}

	// Insert pickup
	if len(req.Pickup) > 0 {
		pQuery := fmt.Sprintf("INSERT INTO fleet_pickup (uuid, fleet_id, organization_id, city_id) VALUES (%s, %s, %s, %s)",
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))
		for _, p := range req.Pickup {
			pID := uuid2()
			_, err := r.db.Exec(pQuery, pID, id, req.OrganizationID, p.CityID)
			if err != nil {
				return "", err
			}
		}
	}

	// Insert addon
	if len(req.Addon) > 0 {
		aQuery := fmt.Sprintf("INSERT INTO fleet_addon (uuid, fleet_id, organization_id, addon_name, addon_desc, addon_price) VALUES (%s, %s, %s, %s, %s, %s)",
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
		for _, a := range req.Addon {
			aID := uuid2()
			_, err := r.db.Exec(aQuery, aID, id, req.OrganizationID, a.AddonName, a.AddonDesc, a.AddonPrice)
			if err != nil {
				return "", err
			}
		}
	}

	// Insert pricing
	if len(req.Pricing) > 0 {
		prQuery := fmt.Sprintf("INSERT INTO fleet_prices (uuid, fleet_id, organization_id, duration, rent_type, price, disc_amount, disc_price, uom) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)",
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9))
		for _, pr := range req.Pricing {
			prID := uuid2()
			_, err := r.db.Exec(prQuery, prID, id, req.OrganizationID, pr.Duration, pr.RentType, pr.Price, pr.DiscAmount, pr.DiscPrice, pr.Uom)
			if err != nil {
				return "", err
			}
		}
	}

	// Insert images
	if len(req.Images) > 0 {
		iQuery := fmt.Sprintf("INSERT INTO fleet_images (uuid, fleet_id, path_file) VALUES (%s, %s, %s)",
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
		for _, img := range req.Images {
			iID := uuid2()
			_, err := r.db.Exec(iQuery, iID, id, img.PathFile)
			if err != nil {
				return "", err
			}
		}
	}

	return id, nil
}

func (r *FleetRepository) GetFleetDetail(id, orgID string) (*model.FleetDetailResponse, error) {
	// Main fleet data
	query := fmt.Sprintf(`
        SELECT fleet_name, fleet_type, capacity, engine, body, description, active, status, thumbnail
        FROM fleets
        WHERE uuid = %s AND organization_id = %s
    `, r.getPlaceholder(1), r.getPlaceholder(2))

	var res model.FleetDetailResponse
	res.Meta.FleetID = id
	err := r.db.QueryRow(query, id, orgID).Scan(
		&res.Meta.FleetName, &res.Meta.FleetType, &res.Meta.Capacity, &res.Meta.Engine,
		&res.Meta.Body, &res.Meta.Description, &res.Meta.Active, &res.Meta.Status, &res.Meta.Thumbnail,
	)
	if err != nil {
		return nil, err
	}

	// Facilities
	res.Facilities, _ = r.GetFleetFacilities(id)

	// Pickup
	res.Pickup, _ = r.GetFleetPickup(orgID, id)

	// Pricing
	res.Pricing, _ = r.GetFleetPricing(orgID, id)

	// Addon
	res.Addon, _ = r.GetFleetAddon(orgID, id)

	// Images
	res.Images, _ = r.GetFleetImages(id)

	return &res, nil
}

func (r *FleetRepository) GetFleetFacilities(fleetID string) ([]string, error) {
	query := fmt.Sprintf("SELECT facility FROM fleet_facilities WHERE fleet_id = %s", r.getPlaceholder(1))
	rows, err := r.db.Query(query, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var facilities []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err == nil {
			facilities = append(facilities, f)
		}
	}
	return facilities, nil
}

func (r *FleetRepository) GetFleetPricing(orgID, fleetID string) ([]model.FleetPriceItem, error) {
	query := `
        SELECT uuid, duration, rent_type, price, disc_amount, disc_price, uom
        FROM fleet_prices
        WHERE fleet_id = %s
    `
	args := []interface{}{fleetID}
	if orgID != "" {
		query += " AND organization_id = %s"
		query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
		args = append(args, orgID)
	} else {
		query = fmt.Sprintf(query, r.getPlaceholder(1))
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.FleetPriceItem, 0)
	for rows.Next() {
		var it model.FleetPriceItem
		if err := rows.Scan(&it.UUID, &it.Duration, &it.RentType, &it.Price, &it.DiscAmount, &it.DiscPrice, &it.Uom); err != nil {
			return nil, err
		}
		it.RentTypeLabel = configs.RentType(it.RentType).String()
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) CreateOrder(req *model.CreateOrderRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	orderID := req.OrderID
	totalAmount := req.TotalAmount

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()

	// 1. Insert fleet_order
	orderQuery := fmt.Sprintf(`
		INSERT INTO fleet_orders (order_id, fleet_id, start_date, end_date, pickup_city_id, pickup_location, unit_qty, price_id, created_at, total_amount, status, payment_status, organization_id)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 2, 3, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11))

	_, err = tx.Exec(orderQuery, orderID, req.FleetID, req.StartDate, req.EndDate, req.PickupCityID, req.PickupLocation, req.Qty, req.PriceID, now, totalAmount, req.OrganizationID)
	if err != nil {
		fmt.Println("error create orders", err)
		return err
	}

	// 2. Insert fleet_orders_customers
	custID := uuid2()
	custQuery := fmt.Sprintf(`
		INSERT INTO fleet_order_customers (customer_id, order_id, customer_name, customer_phone, customer_email, customer_address, created_at, organization_id)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8))

	_, err = tx.Exec(custQuery, custID, orderID, req.Fullname, req.Phone, req.Email, req.Address, now, req.OrganizationID)
	if err != nil {
		fmt.Println("error create customer orders", err)
		return err
	}

	// 3. Insert fleet_orders_addon
	if len(req.Addons) > 0 {
		addonQuery := fmt.Sprintf(`
			INSERT INTO fleet_order_addons (order_addon_id, order_id, addon_id, addon_price, created_at)
			SELECT %s, %s, uuid, addon_price, %s FROM fleet_addon WHERE uuid = %s
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))
		for _, addonID := range req.Addons {
			id := uuid2()
			res, err := tx.Exec(addonQuery, id, orderID, now, addonID)
			if err != nil {
				fmt.Println("error create addon orders", err)
				return err
			}
			rows, _ := res.RowsAffected()
			if rows == 0 {
				return fmt.Errorf("addon not found: %s", addonID)
			}
		}
	}

	// 4. Insert fleet_order_destinations
	if len(req.Destinations) > 0 {
		destQuery := fmt.Sprintf(`
			INSERT INTO fleet_order_destinations (uuid, order_id, city_id, location, created_at)
			VALUES (%s, %s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5))
		for _, dest := range req.Destinations {
			id := uuid2()
			_, err := tx.Exec(destQuery, id, orderID, dest.CityID, dest.Location, now)
			if err != nil {
				fmt.Println("error create dest orders", err)
				return err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (r *FleetRepository) GetFleetOrderSummary(fleetID, priceID string) (*model.OrderFleetSummaryResponse, error) {
	query := fmt.Sprintf(`
		SELECT f.fleet_name, f.capacity, f.engine, f.body, f.description, f.active, f.thumbnail,
		       fp.duration, fp.rent_type, fp.price, fp.uom
		FROM fleets f
		JOIN fleet_prices fp ON fp.fleet_id = f.uuid
		WHERE f.uuid = %s AND fp.uuid = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	res := &model.OrderFleetSummaryResponse{}

	err := r.db.QueryRow(query, fleetID, priceID).Scan(
		&res.FleetName, &res.Capacity, &res.Engine, &res.Body, &res.Description, &res.Active, &res.Thumbnail,
		&res.Duration, &res.RentType, &res.Price, &res.Uom,
	)
	if err != nil {
		return nil, err
	}

	// Facilities
	fQuery := fmt.Sprintf("SELECT facility FROM fleet_facilities WHERE fleet_id = %s", r.getPlaceholder(1))
	rows, err := r.db.Query(fQuery, fleetID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var f string
			if err := rows.Scan(&f); err == nil {
				res.Facilities = append(res.Facilities, f)
			}
		}
	}

	// Pickup Points
	pQuery := fmt.Sprintf("SELECT city_id FROM fleet_pickup WHERE fleet_id = %s", r.getPlaceholder(1))
	pRows, err := r.db.Query(pQuery, fleetID)
	if err == nil {
		defer pRows.Close()
		for pRows.Next() {
			var cityID int
			if err := pRows.Scan(&cityID); err == nil {
				res.PickupPoints = append(res.PickupPoints, model.PickupPoint{CityID: cityID})
			}
		}
	}

	return res, nil
}

func (r *FleetRepository) GetFleetPrices(orgID, fleetID string) ([]model.FleetPriceItem, error) {
	query := fmt.Sprintf(`
		SELECT uuid, duration, rent_type, price, disc_amount, disc_price, uom
		FROM fleet_prices
		WHERE fleet_id = %s ORDER BY price
	`, r.getPlaceholder(1))

	rows, err := r.db.Query(query, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.FleetPriceItem
	for rows.Next() {
		var it model.FleetPriceItem
		var discAmount, discPrice sql.NullFloat64
		if err := rows.Scan(&it.UUID, &it.Duration, &it.RentType, &it.Price, &discAmount, &discPrice, &it.Uom); err == nil {
			if discAmount.Valid {
				it.DiscAmount = discAmount.Float64
			}
			if discPrice.Valid {
				it.DiscPrice = discPrice.Float64
			}
			items = append(items, it)
		}
	}
	return items, nil
}

// GetCities from DB if needed? No, currently reading from JSON in Service.
// But fleet_pickup uses city_id (int).
// The user asked to fix GetFleetPickup TODO.
// If cities are in JSON, we can't join in SQL.
// So we just return IDs and let Service map them.

func (r *FleetRepository) GetFleetPickup(orgID, fleetID string) ([]model.FleetPickupItem, error) {
	query := `
        SELECT uuid, city_id, '' as city_name 
        FROM fleet_pickup
        WHERE fleet_id = %s
    `
	args := []interface{}{fleetID}
	if orgID != "" {
		query += " AND organization_id = %s"
		query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
		args = append(args, orgID)
	} else {
		query = fmt.Sprintf(query, r.getPlaceholder(1))
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.FleetPickupItem
	for rows.Next() {
		var it model.FleetPickupItem
		if err := rows.Scan(&it.UUID, &it.CityID, &it.CityName); err == nil {
			items = append(items, it)
		}
	}
	return items, nil
}

func (r *FleetRepository) GetOrderList(req *model.GetOrderListRequest) ([]model.OrderListItem, int, error) {
	offset := (req.Page - 1) * req.Limit
	query := fmt.Sprintf(`
		SELECT fo.order_id, fo.created_at, fo.total_amount, fo.status, fo.payment_status,
		       COALESCE(foc.customer_name, '') as customer_name,
		       COALESCE(foc.customer_email, '') as customer_email,
		       COALESCE(foc.customer_phone, '') as customer_phone
		FROM fleet_orders fo
		LEFT JOIN fleet_order_customers foc ON fo.order_id = foc.order_id
		WHERE fo.organization_id = %s
	`, r.getPlaceholder(1))

	args := []interface{}{req.OrganizationID}

	if req.Status != 0 {
		query += fmt.Sprintf(" AND fo.status = %s", r.getPlaceholder(len(args)+1))
		args = append(args, req.Status)
	}

	query += fmt.Sprintf(" ORDER BY fo.created_at DESC LIMIT %s OFFSET %s", r.getPlaceholder(len(args)+1), r.getPlaceholder(len(args)+2))
	args = append(args, req.Limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []model.OrderListItem
	for rows.Next() {
		var it model.OrderListItem
		var createdAt time.Time
		if err := rows.Scan(&it.OrderID, &createdAt, &it.TotalAmount, &it.Status, &it.PaymentStatus, &it.CustomerName, &it.CustomerEmail, &it.CustomerPhone); err == nil {
			it.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
			items = append(items, it)
		}
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM fleet_orders WHERE organization_id = %s", r.getPlaceholder(1))
	countArgs := []interface{}{req.OrganizationID}
	if req.Status != 0 {
		countQuery += fmt.Sprintf(" AND status = %s", r.getPlaceholder(len(countArgs)+1))
		countArgs = append(countArgs, req.Status)
	}
	var total int
	r.db.QueryRow(countQuery, countArgs...).Scan(&total)

	return items, total, nil
}

func (r *FleetRepository) GetOrderDetail(orderID, priceID, organizationID string) (*model.OrderDetailResponse, error) {
	// Reusing FindOrderDetail logic but adding priceID check if needed
	// For now just call FindOrderDetail as priceID is inside order table
	return r.FindOrderDetail(orderID, organizationID)
}

func (r *FleetRepository) GetFleetOrderTotalAmount(orderID, priceID, organizationID string) (float64, error) {
	query := fmt.Sprintf("SELECT total_amount FROM fleet_orders WHERE order_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
	var amount float64
	err := r.db.QueryRow(query, orderID, organizationID).Scan(&amount)
	return amount, err
}

func (r *FleetRepository) GetFleetOrderPaymentsByOrderID(orderID, organizationID string) ([]model.FleetOrderPayment, error) {
	query := fmt.Sprintf(`
		SELECT order_payment_id, payment_type, payment_percentage, payment_amount, total_amount, payment_remaining, status, created_at
		FROM fleet_order_payment
		WHERE order_id = %s AND organization_id = %s
		ORDER BY created_at ASC
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	rows, err := r.db.Query(query, orderID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.FleetOrderPayment
	for rows.Next() {
		var p model.FleetOrderPayment
		if err := rows.Scan(&p.OrderPaymentID, &p.PaymentType, &p.PaymentPercentage, &p.PaymentAmount, &p.TotalAmount, &p.PaymentRemaining, &p.Status, &p.CreatedAt); err == nil {
			p.OrderID = orderID
			p.OrganizationID = organizationID
			items = append(items, p)
		}
	}
	return items, nil
}

func (r *FleetRepository) CreateOrderPayment(payment *model.FleetOrderPayment, history *model.OrderPaymentHistory) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()

	// Insert Payment
	payQuery := fmt.Sprintf(`
		INSERT INTO fleet_order_payment (order_payment_id, order_id, organization_id, payment_method, payment_type, payment_percentage, payment_amount, total_amount, payment_remaining, status, created_at, bank_code, account_number, account_name, unique_code)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14), r.getPlaceholder(15))

	_, err = tx.Exec(payQuery, payment.OrderPaymentID, payment.OrderID, payment.OrganizationID, payment.PaymentMethod, payment.PaymentType, payment.PaymentPercentage, payment.PaymentAmount, payment.TotalAmount, payment.PaymentRemaining, payment.Status, now, payment.BankCode, payment.AccountNumber, payment.AccountName, payment.UniqueCode)
	if err != nil {
		return err
	}

	// Insert History
	histQuery := fmt.Sprintf(`
		INSERT INTO fleet_order_payment_history (payment_history_id, order_id, bank_code, bank_account_id, account_number, account_name, created_at, organization_id, payment_amount, unique_code)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10))

	_, err = tx.Exec(histQuery, history.PaymentHistoryID, history.OrderID, history.BankCode, history.BankAccountID, history.AccountNumber, history.AccountName, now, history.OrganizationID, history.PaymentAmount, history.UniqueCode)
	if err != nil {
		return err
	}

	// Update Order Payment Status if needed (e.g. PendingVerification)
	// Usually handled by logic, but maybe update order table?
	// fleet_orders has payment_status.
	// 2 = PendingVerification?
	// The service seems to handle status logic.

	return tx.Commit()
}

func (r *FleetRepository) UpdateFleetOrderPaymentStatus(orderID, organizationID string, oldStatus, newStatus int) error {
	query := fmt.Sprintf(`
		UPDATE fleet_order_payment
		SET status = %s
		WHERE order_id = %s AND organization_id = %s AND status = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))

	_, err := r.db.Exec(query, newStatus, orderID, organizationID, oldStatus)
	return err
}

func (r *FleetRepository) FindOrderDetail(orderID, organizationID string) (*model.OrderDetailResponse, error) {
	query := fmt.Sprintf(`
        SELECT 
            fo.order_id, fo.created_at, fo.price_id,
            f.fleet_name, 
            fp.rent_type, fp.duration, COALESCE(fp.uom, '') as duration_uom, fp.price, 
            fo.unit_qty, fo.total_amount,
            fo.pickup_location, fo.pickup_city_id, fo.start_date, fo.end_date,
            COALESCE(foc.customer_name, '') as customer_name, COALESCE(foc.customer_phone, '') as customer_phone, COALESCE(foc.customer_email, '') as customer_email, COALESCE(foc.customer_address, '') as customer_address
        FROM fleet_orders fo
        JOIN fleets f ON fo.fleet_id = f.uuid
        JOIN fleet_prices fp ON fo.price_id = fp.uuid
        LEFT JOIN fleet_order_customers foc ON fo.order_id = foc.order_id
        WHERE fo.order_id = %s AND fo.organization_id = %s
    `, r.getPlaceholder(1), r.getPlaceholder(2))

	var res model.OrderDetailResponse
	var createdAt time.Time
	var pickupCityID string
	var startDate, endDate time.Time

	err := r.db.QueryRow(query, orderID, organizationID).Scan(
		&res.OrderID, &createdAt, &res.PriceID,
		&res.FleetName,
		&res.RentType, &res.Duration, &res.DurationUom, &res.Price,
		&res.Quantity, &res.TotalAmount,
		&res.Pickup.PickupLocation, &pickupCityID, &startDate, &endDate,
		&res.Customer.CustomerName, &res.Customer.CustomerPhone, &res.Customer.CustomerEmail, &res.Customer.CustomerAddress,
	)
	if err != nil {
		fmt.Println("Error querying order detail:", err)
		return nil, err
	}
	res.OrderDate = createdAt.Format("2006-01-02 15:04:05")
	res.Pickup.PickupCity = pickupCityID
	res.Pickup.StartDate = startDate.Format("2006-01-02")
	res.Pickup.EndDate = endDate.Format("2006-01-02")

	// Destinations
	destQuery := fmt.Sprintf(`SELECT city_id, location FROM fleet_order_destinations WHERE order_id = %s`, r.getPlaceholder(1))
	dRows, err := r.db.Query(destQuery, orderID)
	if err == nil {
		defer dRows.Close()
		for dRows.Next() {
			var d model.OrderDetailDest
			var cID string
			if err := dRows.Scan(&cID, &d.Location); err == nil {
				d.City = cID
				res.Destination = append(res.Destination, d)
			}
		}
	}

	// Addons
	addonQuery := fmt.Sprintf(`
        SELECT fa.addon_name, fa.addon_price
        FROM fleet_order_addons foa 
        JOIN fleet_addon fa ON foa.addon_id = fa.uuid 
        WHERE foa.order_id = %s
    `, r.getPlaceholder(1))
	aRows, err := r.db.Query(addonQuery, orderID)
	if err != nil {
		return nil, err
	}
	defer aRows.Close()
	for aRows.Next() {
		var a model.OrderDetailAddon
		if err := aRows.Scan(&a.AddonName, &a.AddonPrice); err == nil {
			res.Addon = append(res.Addon, a)
		}
	}

	// Payments
	paymentQuery := fmt.Sprintf(`
		SELECT 
			ba.bank_code, ba.account_name, ba.account_number, bl.name as bank_name, 
			op.payment_type, op.payment_percentage, op.payment_amount, op.total_amount, 
			op.payment_remaining, op.status, op.created_at, op.order_payment_id
		FROM fleet_order_payment op
		LEFT JOIN organization_bank_accounts ba ON op.bank_account_id = ba.bank_account_id
		LEFT JOIN bank_lists bl ON ba.bank_code = bl.code
		WHERE op.order_id = %s
		ORDER BY op.created_at DESC
	`, r.getPlaceholder(1))

	pRows, err := r.db.Query(paymentQuery, orderID)
	if err == nil {
		defer pRows.Close()
		var allStatus1 bool = true
		var hasPayment bool = false
		for pRows.Next() {
			hasPayment = true
			var pd model.PaymentDetail
			var bankCode, accName, accNum, bankName sql.NullString
			var createdAt time.Time
			var orderPaymentID string

			if err := pRows.Scan(
				&bankCode, &accName, &accNum, &bankName,
				&pd.PaymentType, &pd.PaymentPercentage, &pd.PaymentAmount, &pd.TotalAmount,
				&pd.PaymentRemaining, &pd.Status, &createdAt, &orderPaymentID,
			); err == nil {
				pd.BankCode = bankCode.String
				pd.AccountName = accName.String
				pd.AccountNumber = accNum.String
				pd.BankName = bankName.String
				pd.PaymentDate = createdAt.Format("2006-01-02 15:04:05")

				res.Payment = append(res.Payment, pd)

				if pd.Status != 1 {
					allStatus1 = false
				}
			}
		}

		// Determine overall payment status
		if !hasPayment {
			res.PaymentStatus = "Belum Bayar"
		} else if allStatus1 {
			res.PaymentStatus = "Lunas"
		}
	}

	return &res, nil
}

func (r *FleetRepository) UpdatePaymentEvidence(orderID, organizationID, filePath string) error {
	// Find latest payment ID
	subQuery := fmt.Sprintf(`
		SELECT order_payment_id 
		FROM fleet_order_payment 
		WHERE order_id = %s AND organization_id = %s 
		ORDER BY created_at DESC 
		LIMIT 1
	`, r.getPlaceholder(2), r.getPlaceholder(3))

	query := fmt.Sprintf(`
		UPDATE fleet_order_payment
		SET evidence_file = %s
		WHERE order_payment_id = (%s)
	`, r.getPlaceholder(1), subQuery)

	_, err := r.db.Exec(query, filePath, orderID, organizationID)
	return err
}

func (r *FleetRepository) GetPartnerOrderList(orgID string) ([]model.PartnerOrderListItem, error) {
	query := fmt.Sprintf(`
        SELECT fo.order_id, f.fleet_name, fo.start_date, fo.end_date, fo.unit_qty, fo.payment_status, 
               p.duration, p.uom, fo.total_amount, p.rent_type
        FROM fleet_orders fo 
        INNER JOIN fleets f ON fo.fleet_id = f.uuid 
        INNER JOIN fleet_prices p ON p.uuid = fo.price_id 
        WHERE f.organization_id = %s
        ORDER BY fo.created_at DESC
    `, r.getPlaceholder(1))

	rows, err := r.db.Query(query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.PartnerOrderListItem, 0)
	for rows.Next() {
		var it model.PartnerOrderListItem
		var startDate, endDate time.Time
		var rentType int
		if err := rows.Scan(
			&it.OrderID, &it.FleetName, &startDate, &endDate, &it.UnitQty,
			&it.PaymentStatus, &it.Duration, &it.Uom, &it.TotalAmount, &rentType,
		); err != nil {
			return nil, err
		}
		it.StartDate = startDate
		it.EndDate = endDate

		switch rentType {
		case 1:
			it.RentType = "Cititour"
		case 2:
			it.RentType = "Overland"
		case 3:
			it.RentType = "Pickup / Drop"
		default:
			it.RentType = "Unknown"
		}

		items = append(items, it)
	}
	return items, nil
}

func (r *FleetRepository) GetPartnerOrderDetail(orderID, orgID string) (*model.OrderDetailResponse, error) {
	query := fmt.Sprintf(`
        SELECT 
            fo.order_id, fo.created_at, fo.price_id,
            f.fleet_name, 
            fp.rent_type, fp.duration, COALESCE(fp.uom, '') as duration_uom, fp.price, 
            fo.unit_qty, fo.total_amount,
            fo.pickup_location, fo.pickup_city_id, fo.start_date, fo.end_date,
            COALESCE(foc.customer_name, '') as customer_name, COALESCE(foc.customer_phone, '') as customer_phone, COALESCE(foc.customer_email, '') as customer_email, COALESCE(foc.customer_address, '') as customer_address
        FROM fleet_orders fo
        JOIN fleets f ON fo.fleet_id = f.uuid
        JOIN fleet_prices fp ON fo.price_id = fp.uuid
        LEFT JOIN fleet_order_customers foc ON fo.order_id = foc.order_id
        WHERE fo.order_id = %s AND f.organization_id = %s
    `, r.getPlaceholder(1), r.getPlaceholder(2))

	var res model.OrderDetailResponse
	var createdAt time.Time
	var pickupCityID string
	var startDate, endDate time.Time

	err := r.db.QueryRow(query, orderID, orgID).Scan(
		&res.OrderID, &createdAt, &res.PriceID,
		&res.FleetName,
		&res.RentType, &res.Duration, &res.DurationUom, &res.Price,
		&res.Quantity, &res.TotalAmount,
		&res.Pickup.PickupLocation, &pickupCityID, &startDate, &endDate,
		&res.Customer.CustomerName, &res.Customer.CustomerPhone, &res.Customer.CustomerEmail, &res.Customer.CustomerAddress,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found or access denied")
		}
		fmt.Println("Error querying order detail:", err)
		return nil, err
	}
	res.OrderDate = createdAt.Format("2006-01-02 15:04:05")
	res.Pickup.PickupCity = pickupCityID
	res.Pickup.StartDate = startDate.Format("2006-01-02")
	res.Pickup.EndDate = endDate.Format("2006-01-02")

	// Destinations
	destQuery := fmt.Sprintf(`SELECT city_id, location FROM fleet_order_destinations WHERE order_id = %s`, r.getPlaceholder(1))
	dRows, err := r.db.Query(destQuery, orderID)
	if err == nil {
		defer dRows.Close()
		for dRows.Next() {
			var d model.OrderDetailDest
			var cID string
			if err := dRows.Scan(&cID, &d.Location); err == nil {
				d.City = cID
				res.Destination = append(res.Destination, d)
			}
		}
	}

	// Addons
	addonQuery := fmt.Sprintf(`
        SELECT fa.addon_name, fa.addon_price
        FROM fleet_order_addons foa 
        JOIN fleet_addon fa ON foa.addon_id = fa.uuid 
        WHERE foa.order_id = %s
    `, r.getPlaceholder(1))
	aRows, err := r.db.Query(addonQuery, orderID)
	if err == nil {
		defer aRows.Close()
		for aRows.Next() {
			var a model.OrderDetailAddon
			if err := aRows.Scan(&a.AddonName, &a.AddonPrice); err == nil {
				res.Addon = append(res.Addon, a)
			}
		}
	}
	return &res, nil
}

func (r *FleetRepository) GetFleetAddon(orgID, fleetID string) ([]model.FleetAddonItem, error) {
	query := `SELECT uuid, addon_name, addon_desc, addon_price FROM fleet_addon WHERE fleet_id = %s`
	args := []interface{}{fleetID}
	if orgID != "" {
		query += " AND organization_id = %s"
		query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
		args = append(args, orgID)
	} else {
		query = fmt.Sprintf(query, r.getPlaceholder(1))
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.FleetAddonItem
	for rows.Next() {
		var it model.FleetAddonItem
		if err := rows.Scan(&it.UUID, &it.AddonName, &it.AddonDesc, &it.AddonPrice); err == nil {
			items = append(items, it)
		}
	}
	return items, nil
}

func (r *FleetRepository) GetFleetImages(fleetID string) ([]model.FleetImageItem, error) {
	query := fmt.Sprintf("SELECT uuid, path_file FROM fleet_images WHERE fleet_id = %s", r.getPlaceholder(1))
	rows, err := r.db.Query(query, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.FleetImageItem
	for rows.Next() {
		var it model.FleetImageItem
		if err := rows.Scan(&it.UUID, &it.PathFile); err == nil {
			items = append(items, it)
		}
	}
	return items, nil
}

func (r *FleetRepository) GetServiceFleets(page, perPage int) ([]model.ServiceFleetItem, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage

	var groupConcat string
	if r.driver == "postgres" {
		groupConcat = "STRING_AGG(CAST(city_id AS VARCHAR), ',')"
	} else {
		groupConcat = "GROUP_CONCAT(city_id)"
	}

	query := fmt.Sprintf(`
        SELECT f.uuid, f.fleet_name, f.fleet_type, f.capacity, f.production_year, f.engine, f.body, f.description, f.thumbnail, f.created_at,
        (SELECT MIN(price) FROM fleet_prices WHERE fleet_id = f.uuid) as price,
        (SELECT uom FROM fleet_prices WHERE fleet_id = f.uuid ORDER BY price ASC LIMIT 1) as uom,
        (SELECT %s FROM fleet_pickup WHERE fleet_id = f.uuid) as cities
        FROM fleets f
        WHERE f.active = true
        ORDER BY f.created_at DESC
        LIMIT %d OFFSET %d
    `, groupConcat, perPage, offset)

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.ServiceFleetItem
	for rows.Next() {
		var it model.ServiceFleetItem
		var price sql.NullFloat64
		var uom sql.NullString
		var cities sql.NullString

		if err := rows.Scan(
			&it.FleetID, &it.FleetName, &it.FleetType, &it.Capacity, &it.ProductionYear, &it.Engine, &it.Body, &it.Description, &it.Thumbnail, &it.CreatedAt,
			&price, &uom, &cities,
		); err != nil {
			return nil, err
		}

		if price.Valid {
			it.OriginalPrice = price.Float64
		}
		if uom.Valid {
			it.Uom = uom.String
		}
		if cities.Valid {
			it.Cities = strings.Split(cities.String, ",")
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *FleetRepository) GetAvailableCities(orgID string) ([]int, error) {
	query := fmt.Sprintf(`
		SELECT city_id
		FROM fleet_pickup
		WHERE organization_id = %s
		GROUP BY city_id
	`, r.getPlaceholder(1))

	rows, err := r.db.Query(query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cities []int
	for rows.Next() {
		var cityID int
		if err := rows.Scan(&cityID); err == nil {
			cities = append(cities, cityID)
		}
	}
	return cities, nil
}

func (r *FleetRepository) GetFleetOrgID(fleetID string) (string, error) {
	query := fmt.Sprintf("SELECT organization_id FROM fleets WHERE uuid = %s", r.getPlaceholder(1))
	var orgID string
	err := r.db.QueryRow(query, fleetID).Scan(&orgID)
	return orgID, err
}

func (r *FleetRepository) GetFleetDetailMeta(orgID, fleetID string) (*model.FleetDetailMeta, error) {
	query := `
        SELECT uuid, fleet_type, fleet_name, capacity, production_year, engine, body, description, thumbnail, active, status, created_at, created_by, updated_at, updated_by
        FROM fleets
        WHERE uuid = %s
    `
	args := []interface{}{fleetID}
	if orgID != "" {
		query += " AND organization_id = %s"
		query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
		args = append(args, orgID)
	} else {
		query = fmt.Sprintf(query, r.getPlaceholder(1))
	}

	var meta model.FleetDetailMeta
	// Note: using explicit fields to avoid confusion
	// Need to handle nulls if necessary, but assuming fields are not null for now or struct handles them.
	// Check struct definition: CreatedAt is string, DB likely timestamp.
	// If DB is timestamp, Scan into time.Time then format.
	var createdAt time.Time
	var updatedAt sql.NullTime
	var updatedBy sql.NullString
	// FleetDetailMeta: CreatedAt string `json:"created_at"`

	err := r.db.QueryRow(query, args...).Scan(
		&meta.FleetID, &meta.FleetType, &meta.FleetName, &meta.Capacity, &meta.ProductionYear, &meta.Engine, &meta.Body, &meta.Description, &meta.Thumbnail, &meta.Active, &meta.Status,
		&createdAt, &meta.CreatedBy, &updatedAt, &updatedBy,
	)
	if err != nil {
		return nil, err
	}
	meta.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
	if updatedAt.Valid {
		meta.UpdatedAt = updatedAt.Time.Format("2006-01-02 15:04:05")
	}
	if updatedBy.Valid {
		meta.UpdatedBy = updatedBy.String
	}

	return &meta, nil
}

func (r *FleetRepository) GetPriceByID(priceID string) (float64, int, error) {
	query := fmt.Sprintf("SELECT price, rent_type FROM fleet_prices WHERE uuid = %s", r.getPlaceholder(1))
	var price float64
	var rentType int
	err := r.db.QueryRow(query, priceID).Scan(&price, &rentType)
	return price, rentType, err
}

func (r *FleetRepository) GetAddonPriceSum(addonIDs []string) (float64, error) {
	if len(addonIDs) == 0 {
		return 0, nil
	}
	// Create placeholders for IN clause
	placeholders := make([]string, len(addonIDs))
	args := make([]interface{}, len(addonIDs))
	for i, id := range addonIDs {
		placeholders[i] = r.getPlaceholder(i + 1)
		args[i] = id
	}
	query := fmt.Sprintf("SELECT COALESCE(SUM(addon_price), 0) FROM fleet_addon WHERE uuid IN (%s)", strings.Join(placeholders, ","))
	var total float64
	err := r.db.QueryRow(query, args...).Scan(&total)
	return total, err
}

func (r *FleetRepository) GetOrderCountByOrgID(orgID string) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM fleet_orders WHERE organization_id = %s", r.getPlaceholder(1))
	var count int
	err := r.db.QueryRow(query, orgID).Scan(&count)
	return count, err
}

func (r *FleetRepository) getPlaceholder(pos int) string {
	if r.driver == "postgres" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}
