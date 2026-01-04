package repository

import (
	"database/sql"
	"fmt"
	"log"
	"service-travego/model"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FleetRepository struct {
	db     *sql.DB
	driver string
}

func NewFleetRepository(db *sql.DB, driver string) *FleetRepository {
	return &FleetRepository{db: db, driver: driver}
}

func (r *FleetRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

func (r *FleetRepository) CreateFleetWithDetails(uuid, createdBy, organizationID string, req *model.CreateFleetRequest) error {
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

	fleetQuery := fmt.Sprintf(`
        INSERT INTO fleets (uuid, fleet_name, fleet_type, capacity, production_year, engine, body, description, thumbnail, created_at, created_by, organization_id, active, status)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1)
    `,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10),
		r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13),
	)

	argsFleet := []interface{}{uuid, req.FleetName, req.FleetType, req.Capacity, req.ProductionYear, req.Engine, req.Body, req.Description, req.Thumbnail, now, createdBy, organizationID, req.Active}
	_, err = tx.Exec(fleetQuery, argsFleet...)
	if err != nil {
		log.Printf("[ERROR] Insert fleets failed - driver=%s, err=%v\nSQL: %s\nArgs: %#v", r.driver, err, fleetQuery, argsFleet)
		return err
	}

	if len(req.PickupPoint) > 0 {
		pickupQuery := fmt.Sprintf(`
            INSERT INTO fleet_pickup (uuid, fleet_id, city_id, created_by, organization_id, created_at, updated_by, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		)
		for _, cityID := range req.PickupPoint {
			pu := uuid2()
			args := []interface{}{pu, uuid, cityID, createdBy, organizationID, now, createdBy, now}
			_, err = tx.Exec(pickupQuery, args...)
			if err != nil {
				log.Printf("[ERROR] Insert fleet_pickup failed - driver=%s, err=%v\nSQL: %s\nArgs: %#v", r.driver, err, pickupQuery, args)
				return err
			}
		}
	}

	if len(req.Facilities) > 0 {
		facQuery := fmt.Sprintf(`
            INSERT INTO fleet_facilities (uuid, fleet_id, facility, created_by, organization_id, created_at, updated_by, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		)
		for _, facility := range req.Facilities {
			fu := uuid2()
			args := []interface{}{fu, uuid, facility, createdBy, organizationID, now, createdBy, now}
			_, err = tx.Exec(facQuery, args...)
			if err != nil {
				log.Printf("[ERROR] Insert fleet_facilities failed - driver=%s, err=%v\nSQL: %s\nArgs: %#v", r.driver, err, facQuery, args)
				return err
			}
		}
	}

	if len(req.Prices) > 0 {
		priceQuery := fmt.Sprintf(`
            INSERT INTO fleet_prices (uuid, fleet_id, duration, rent_type, price, uom, created_by, organization_id, created_at, updated_by, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11),
		)
		for _, p := range req.Prices {
			pru := uuid2()
			args := []interface{}{pru, uuid, p.Duration, p.RentCategory, p.Price, p.Uom, createdBy, organizationID, now, createdBy, now}
			_, err = tx.Exec(priceQuery, args...)
			if err != nil {
				log.Printf("[ERROR] Insert fleet_prices failed - driver=%s, err=%v\nSQL: %s\nArgs: %#v", r.driver, err, priceQuery, args)
				return err
			}
		}
	}

	if len(req.Addon) > 0 {
		addonQuery := fmt.Sprintf(`
            INSERT INTO fleet_addon (uuid, fleet_id, addon_name, addon_desc, addon_price, created_by, organization_id, created_at, updated_by, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10),
		)
		for _, a := range req.Addon {
			au := uuid2()
			args := []interface{}{au, uuid, a.AddonName, a.Description, a.Price, createdBy, organizationID, now, createdBy, now}
			_, err = tx.Exec(addonQuery, args...)
			if err != nil {
				log.Printf("[ERROR] Insert fleet_addon failed - driver=%s, err=%v\nSQL: %s\nArgs: %#v", r.driver, err, addonQuery, args)
				return err
			}
		}
	}

	if len(req.BodyImages) > 0 {
		imgQuery := fmt.Sprintf(`
            INSERT INTO fleet_images (uuid, fleet_id, path_file)
            VALUES (%s, %s, %s)
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3),
		)
		for _, path := range req.BodyImages {
			if path == "" {
				continue
			}
			iu := uuid2()
			args := []interface{}{iu, uuid, path}
			_, err = tx.Exec(imgQuery, args...)
			if err != nil {
				log.Printf("[ERROR] Insert fleet_images failed - driver=%s, err=%v\nSQL: %s\nArgs: %#v", r.driver, err, imgQuery, args)
				return err
			}
		}
	}

	err = tx.Commit()
	return err
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
	query = query + " ORDER BY f.created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.FleetListItem, 0)
	for rows.Next() {
		var it model.FleetListItem
		if err := rows.Scan(&it.FleetID, &it.FleetType, &it.FleetName, &it.Capacity, &it.Engine, &it.Body, &it.Active, &it.Status, &it.Thumbnail); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) GetPriceByID(priceID string) (float64, int, error) {
	query := `SELECT price, rent_type FROM fleet_prices WHERE uuid = %s`
	query = fmt.Sprintf(query, r.getPlaceholder(1))
	var price float64
	var rentType int
	err := r.db.QueryRow(query, priceID).Scan(&price, &rentType)
	return price, rentType, err
}

func (r *FleetRepository) GetAddonPriceSum(addonIDs []string) (float64, error) {
	if len(addonIDs) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(addonIDs))
	args := make([]interface{}, len(addonIDs))
	for i, id := range addonIDs {
		placeholders[i] = r.getPlaceholder(i + 1)
		args[i] = id
	}
	query := fmt.Sprintf(`SELECT COALESCE(SUM(addon_price), 0) FROM fleet_addon WHERE uuid IN (%s)`, strings.Join(placeholders, ","))
	var total float64
	err := r.db.QueryRow(query, args...).Scan(&total)
	return total, err
}

func (r *FleetRepository) CreateFleetOrder(orderID string, totalAmount float64, req *model.CreateOrderRequest) error {
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

	// 4. Insert fleet_orders_destination
	if len(req.Destinations) > 0 {
		destQuery := fmt.Sprintf(`
			INSERT INTO fleet_order_destinations (order_destination_id, order_id, city_id, location, created_at)
			VALUES (%s, %s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5))
		for _, dest := range req.Destinations {
			id := uuid2()
			_, err = tx.Exec(destQuery, id, orderID, dest.CityID, dest.Location, now)
			if err != nil {
				fmt.Println(err)
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *FleetRepository) GetFleetOrderSummary(fleetID, priceID string) (*model.OrderFleetSummaryResponse, error) {
	query := `
        SELECT f.fleet_name, f.capacity, f.engine, f.body, COALESCE(f.description, ''), f.active, COALESCE(f.thumbnail, ''),
               fp.duration, fp.rent_type, fp.price, COALESCE(fp.uom, '')
        FROM fleets f
        JOIN fleet_prices fp ON f.uuid = fp.fleet_id
        WHERE f.uuid = %s AND fp.uuid = %s
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))

	var res model.OrderFleetSummaryResponse
	err := r.db.QueryRow(query, fleetID, priceID).Scan(
		&res.FleetName, &res.Capacity, &res.Engine, &res.Body, &res.Description, &res.Active, &res.Thumbnail,
		&res.Duration, &res.RentType, &res.Price, &res.Uom,
	)
	if err != nil {
		return nil, err
	}

	// Fetch facilities
	facilities, err := r.GetFleetFacilities(fleetID)
	if err != nil {
		// Log error or ignore? Usually safe to ignore if just empty
		res.Facilities = []string{}
	} else {
		res.Facilities = facilities
	}

	// Fetch pickup points
	pickupQuery := fmt.Sprintf("SELECT city_id FROM fleet_pickup WHERE fleet_id = %s", r.getPlaceholder(1))
	pRows, err := r.db.Query(pickupQuery, fleetID)
	if err != nil {
		res.PickupPoints = []model.PickupPoint{}
	} else {
		defer pRows.Close()
		var pickups []model.PickupPoint
		for pRows.Next() {
			var p model.PickupPoint
			if err := pRows.Scan(&p.CityID); err == nil {
				pickups = append(pickups, p)
			}
		}
		res.PickupPoints = pickups
	}

	return &res, nil
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
	res.Pickup.PickupCity = pickupCityID // Store ID temporarily
	res.Pickup.StartDate = startDate.Format("2006-01-02")
	res.Pickup.EndDate = endDate.Format("2006-01-02")

	// Destinations
	destQuery := fmt.Sprintf(`SELECT city_id, location FROM fleet_order_destinations WHERE order_id = %s`, r.getPlaceholder(1))
	dRows, err := r.db.Query(destQuery, orderID)
	if err != nil {
		fmt.Println("Error querying order destinations:", err)
		return nil, err
	}
	defer dRows.Close()
	for dRows.Next() {
		var d model.OrderDetailDest
		var cID string
		if err := dRows.Scan(&cID, &d.Location); err == nil {
			d.City = cID // Store ID temporarily
			res.Destination = append(res.Destination, d)
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
			op.payment_remaining, op.status, op.created_at as payment_date,
			COALESCE(NULLIF(op.unique_code, '')::INTEGER, 0) as unique_code
		FROM fleet_order_payment op 
		INNER JOIN organization_bank_accounts ba ON op.payment_method = ba.bank_account_id 
		INNER JOIN bank_list bl ON bl.code = ba.bank_code 
		WHERE op.order_id = %s AND op.status > 0
	`, r.getPlaceholder(1))

	pRows, err := r.db.Query(paymentQuery, orderID)
	if err != nil {
		fmt.Println("Error querying order payments:", err)
		return nil, err
	}
	defer pRows.Close()

	res.Payment = make([]model.PaymentDetail, 0)
	for pRows.Next() {
		var p model.PaymentDetail
		var paymentDate time.Time
		err := pRows.Scan(
			&p.BankCode, &p.AccountName, &p.AccountNumber, &p.BankName,
			&p.PaymentType, &p.PaymentPercentage, &p.PaymentAmount, &p.TotalAmount,
			&p.PaymentRemaining, &p.Status, &paymentDate, &p.UniqueCode,
		)
		if err != nil {
			fmt.Println("Error scanning payment row:", err)
			continue
		}
		p.PaymentDate = paymentDate.Format("2006-01-02 15:04:05")
		res.Payment = append(res.Payment, p)
	}

	if len(res.Payment) == 0 {
		res.PaymentStatus = "Dibatalkan"
	} else {
		countType2 := 0
		anyStatus1 := false
		allStatus1 := true
		hasStatus10 := false
		for _, p := range res.Payment {
			if p.PaymentType == 2 {
				countType2++
			}
			if p.Status == 1 {
				anyStatus1 = true
			}
			if p.Status != 1 {
				allStatus1 = false
			}
			if p.Status == 10 {
				hasStatus10 = true
			}
		}
		if countType2 > 0 && len(res.Payment) > 1 && anyStatus1 && !allStatus1 {
			res.PaymentStatus = "Belum Lunas"
		} else if hasStatus10 {
			res.PaymentStatus = "Menunggu verifikasi"
		} else if allStatus1 {
			res.PaymentStatus = "Lunas"
		}
	}

	return &res, nil
}

func (r *FleetRepository) GetFleetOrderTotalAmount(orderID, priceID, organizationID string) (float64, error) {
	query := fmt.Sprintf("SELECT total_amount FROM fleet_orders WHERE order_id = %s AND price_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	var amount float64
	err := r.db.QueryRow(query, orderID, priceID, organizationID).Scan(&amount)
	return amount, err
}

func (r *FleetRepository) CreateOrderPayment(p *model.FleetOrderPayment, h *model.OrderPaymentHistory) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := fmt.Sprintf(`
		INSERT INTO fleet_order_payment (
			order_payment_id, order_id, organization_id, payment_method, 
			payment_type, payment_percentage, payment_amount, total_amount, 
			payment_remaining, status, created_at, unique_code
		) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12),
	)

	_, err = tx.Exec(query,
		p.OrderPaymentID, p.OrderID, p.OrganizationID, p.PaymentMethod,
		p.PaymentType, p.PaymentPercentage, p.PaymentAmount, p.TotalAmount,
		p.PaymentRemaining, p.Status, p.CreatedAt, p.UniqueCode,
	)
	if err != nil {
		return err
	}

	queryHistory := fmt.Sprintf(`
		INSERT INTO order_payment_history (
			payment_history_id, order_id, bank_code, bank_account_id, 
			account_number, account_name, created_at, organization_id, payment_amount, unique_code
		) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		r.getPlaceholder(9), r.getPlaceholder(10),
	)

	_, err = tx.Exec(queryHistory,
		h.PaymentHistoryID, h.OrderID, h.BankCode, h.BankAccountID,
		h.AccountNumber, h.AccountName, h.CreatedAt, h.OrganizationID, h.PaymentAmount, h.UniqueCode,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *FleetRepository) GetFleetOrderPaymentsByOrderID(orderID, organizationID string) ([]model.FleetOrderPayment, error) {
	query := fmt.Sprintf(`
		SELECT 
			order_payment_id, order_id, organization_id, payment_method, 
			payment_type, payment_percentage, payment_amount, total_amount, 
			payment_remaining, status, created_at
		FROM fleet_order_payment
		WHERE order_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	rows, err := r.db.Query(query, orderID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []model.FleetOrderPayment
	for rows.Next() {
		var p model.FleetOrderPayment
		err := rows.Scan(
			&p.OrderPaymentID, &p.OrderID, &p.OrganizationID, &p.PaymentMethod,
			&p.PaymentType, &p.PaymentPercentage, &p.PaymentAmount, &p.TotalAmount,
			&p.PaymentRemaining, &p.Status, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, nil
}

func (r *FleetRepository) UpdateFleetOrderPaymentStatus(orderID, organizationID string, currentStatus, newStatus int) error {
	query := fmt.Sprintf(`
		UPDATE fleet_order_payment
		SET status = %s
		WHERE order_id = %s AND organization_id = %s AND status = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))
	_, err := r.db.Exec(query, newStatus, orderID, organizationID, currentStatus)
	return err
}

func (r *FleetRepository) GetServiceFleets(page, perPage int) ([]model.ServiceFleetItem, error) {
	query := `
		SELECT DISTINCT 
			f.uuid as fleet_id, 
			f.fleet_name, 
			ft.label as fleet_type, 
			f.capacity, 
			f.production_year, 
			f.engine, 
			f.body, 
			f.description, 
			f.thumbnail, 
			mp.price AS original_price, 
			mp.duration, 
			mp.uom, 
			f.created_at, 
			ho.discount_type, 
			ho.discount_value 
		FROM fleets f 
		INNER JOIN fleet_types ft ON f.fleet_type = ft.id 
		INNER JOIN ( 
			SELECT fleet_id, price, duration, uom 
			FROM fleet_prices fp1 
			WHERE price = (SELECT MIN(price) FROM fleet_prices WHERE fleet_id = fp1.fleet_id) 
			GROUP BY fleet_id, price, duration, uom 
		) mp ON mp.fleet_id = f.uuid 
		LEFT JOIN hot_offers ho ON ho.product_id = f.uuid 
	`

	limit := perPage
	offset := 0
	if page > 0 {
		offset = (page - 1) * limit
	}

	query += fmt.Sprintf(" LIMIT %s OFFSET %s", r.getPlaceholder(1), r.getPlaceholder(2))

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.ServiceFleetItem
	for rows.Next() {
		var item model.ServiceFleetItem
		var originalPrice sql.NullFloat64
		var discountType sql.NullString
		var discountValue sql.NullFloat64
		var description sql.NullString
		var thumbnail sql.NullString
		var createdAt sql.NullTime

		// Scan matching the order
		err := rows.Scan(
			&item.FleetID,
			&item.FleetName,
			&item.FleetType,
			&item.Capacity,
			&item.ProductionYear,
			&item.Engine,
			&item.Body,
			&description,
			&thumbnail,
			&originalPrice,
			&item.Duration,
			&item.Uom,
			&createdAt,
			&discountType,
			&discountValue,
		)
		if err != nil {
			return nil, err
		}

		if description.Valid {
			item.Description = description.String
		}
		if thumbnail.Valid {
			item.Thumbnail = thumbnail.String
		}
		if originalPrice.Valid {
			item.OriginalPrice = originalPrice.Float64
		}
		if createdAt.Valid {
			item.CreatedAt = createdAt.Time
		}
		if discountType.Valid {
			val := discountType.String
			item.DiscountType = &val
		}
		if discountValue.Valid {
			val := discountValue.Float64
			item.DiscountValue = &val
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch pickup cities separately
	if len(items) > 0 {
		fleetIDs := make([]string, len(items))
		for i, it := range items {
			fleetIDs[i] = fmt.Sprintf("'%s'", it.FleetID)
		}

		pickupQuery := fmt.Sprintf(`
			SELECT fleet_id, city_id 
			FROM fleet_pickup 
			WHERE fleet_id IN (%s)
		`, strings.Join(fleetIDs, ","))

		pRows, err := r.db.Query(pickupQuery)
		if err != nil {
			// Log error but return items (partial success) or return error?
			// Usually partial success if critical data is there. But let's return error to be safe or ignore.
			// Let's just return items without cities if this fails, or log.
			// For now, let's return error.
			return nil, err
		}
		defer pRows.Close()

		pickupMap := make(map[string][]string)
		for pRows.Next() {
			var fID string
			var cID int
			if err := pRows.Scan(&fID, &cID); err == nil {
				pickupMap[fID] = append(pickupMap[fID], strconv.Itoa(cID))
			}
		}

		for i := range items {
			if cities, ok := pickupMap[items[i].FleetID]; ok {
				items[i].Cities = cities
			} else {
				items[i].Cities = []string{}
			}
		}
	}

	return items, nil
}

func (r *FleetRepository) GetFleetOrgID(fleetID string) (string, error) {
	query := `SELECT organization_id FROM fleets WHERE uuid = %s`
	query = fmt.Sprintf(query, r.getPlaceholder(1))
	var orgID string
	err := r.db.QueryRow(query, fleetID).Scan(&orgID)
	if err != nil {
		return "", err
	}
	return orgID, nil
}

func (r *FleetRepository) GetFleetDetailMeta(orgID, fleetID string) (*model.FleetDetailMeta, error) {
	query := `
        SELECT f.uuid AS fleet_id, ft.label AS fleet_type, f.fleet_name, f.capacity,
               COALESCE(f.production_year, 0) AS production_year, f.engine, f.body,
               COALESCE(f.fuel_type, '') AS fuel_type, COALESCE(f.transmission, '') AS transmission,
               COALESCE(f.description, '') AS description, f.thumbnail,
               f.created_at, u.fullname AS created_by, f.updated_at, COALESCE(u2.fullname, '') AS updated_by
        FROM fleets f
        INNER JOIN fleet_types ft ON f.fleet_type = ft.id
        INNER JOIN users u ON u.user_id = f.created_by
        LEFT JOIN users u2 ON u2.user_id = f.updated_by
        WHERE f.organization_id = %s AND f.uuid = %s
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
	row := r.db.QueryRow(query, orgID, fleetID)
	var meta model.FleetDetailMeta
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	var createdBy string
	var updatedBy string
	err := row.Scan(&meta.FleetID, &meta.FleetType, &meta.FleetName, &meta.Capacity,
		&meta.ProductionYear, &meta.Engine, &meta.Body,
		&meta.FuelType, &meta.Transmission,
		&meta.Description, &meta.Thumbnail,
		&createdAt, &createdBy, &updatedAt, &updatedBy)
	if err != nil {
		return nil, err
	}
	if createdAt.Valid {
		meta.CreatedAt = createdAt.Time.Format(time.RFC3339)
	}
	meta.CreatedBy = createdBy
	if updatedAt.Valid {
		meta.UpdatedAt = updatedAt.Time.Format(time.RFC3339)
	} else {
		meta.UpdatedAt = ""
	}
	meta.UpdatedBy = updatedBy
	return &meta, nil
}

func (r *FleetRepository) GetFleetFacilities(fleetID string) ([]string, error) {
	query := `
        SELECT COALESCE(facility, '') AS facility FROM fleet_facilities WHERE fleet_id = %s
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1))
	rows, err := r.db.Query(query, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]string, 0)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) GetFleetPickup(orgID, fleetID string) ([]model.FleetPickupItem, error) {
	query := `
        SELECT uuid, city_id FROM fleet_pickup WHERE organization_id = %s AND fleet_id = %s
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
	rows, err := r.db.Query(query, orgID, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.FleetPickupItem, 0)
	for rows.Next() {
		var it model.FleetPickupItem
		if err := rows.Scan(&it.UUID, &it.CityID); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) GetFleetAddon(orgID, fleetID string) ([]model.FleetAddonItem, error) {
	query := `
        SELECT uuid,
               COALESCE(addon_name, '') AS addon_name,
               COALESCE(addon_desc, '') AS addon_desc,
               COALESCE(addon_price, 0) AS addon_price
        FROM fleet_addon WHERE organization_id = %s AND fleet_id = %s
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
	rows, err := r.db.Query(query, orgID, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.FleetAddonItem, 0)
	for rows.Next() {
		var it model.FleetAddonItem
		if err := rows.Scan(&it.UUID, &it.AddonName, &it.AddonDesc, &it.AddonPrice); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) GetFleetPrices(orgID, fleetID string) ([]model.FleetPriceItem, error) {
	query := `
        SELECT uuid, duration, rent_type, price,
               COALESCE(disc_amount, 0) AS disc_amount,
               COALESCE(disc_price, 0)  AS disc_price,
               COALESCE(uom, '') AS uom
        FROM fleet_prices WHERE organization_id = %s AND fleet_id = %s
        ORDER BY price ASC
    `

	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))
	rows, err := r.db.Query(query, orgID, fleetID)
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
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) GetFleetImages(fleetID string) ([]model.FleetImageItem, error) {
	query := `
        SELECT uuid, path_file FROM fleet_images WHERE fleet_id = %s
    `
	query = fmt.Sprintf(query, r.getPlaceholder(1))
	rows, err := r.db.Query(query, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.FleetImageItem, 0)
	for rows.Next() {
		var it model.FleetImageItem
		if err := rows.Scan(&it.UUID, &it.PathFile); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FleetRepository) GetOrderCountByOrgID(orgID string) (int, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM fleet_orders WHERE organization_id = %s
	`, r.getPlaceholder(1))

	var count int
	err := r.db.QueryRow(query, orgID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *FleetRepository) GetOrderList(req *model.GetOrderListRequest) ([]model.OrderListItem, int, error) {
	where := []string{fmt.Sprintf("fo.organization_id = %s", r.getPlaceholder(1))}
	args := []interface{}{req.OrganizationID}
	paramIdx := 2

	if req.Status > 0 {
		where = append(where, fmt.Sprintf("fo.status = %s", r.getPlaceholder(paramIdx)))
		args = append(args, req.Status)
		paramIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM fleet_orders fo WHERE %s", whereClause)
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch data
	query := fmt.Sprintf(`
		SELECT 
			fo.order_id, 
			fo.created_at, 
			fo.total_amount, 
			fo.status, 
			fo.payment_status,
			COALESCE(foc.customer_name, ''), 
			COALESCE(foc.customer_email, ''), 
			COALESCE(foc.customer_phone, '')
		FROM fleet_orders fo
		LEFT JOIN fleet_order_customers foc ON fo.order_id = foc.order_id
		WHERE %s
		ORDER BY fo.created_at DESC
	`, whereClause)

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := (req.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	query += fmt.Sprintf(" LIMIT %s OFFSET %s", r.getPlaceholder(paramIdx), r.getPlaceholder(paramIdx+1))
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]model.OrderListItem, 0)
	for rows.Next() {
		var it model.OrderListItem
		var createdAt time.Time
		if err := rows.Scan(
			&it.OrderID,
			&createdAt,
			&it.TotalAmount,
			&it.Status,
			&it.PaymentStatus,
			&it.CustomerName,
			&it.CustomerEmail,
			&it.CustomerPhone,
		); err != nil {
			return nil, 0, err
		}
		it.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		items = append(items, it)
	}

	return items, total, nil
}

func (r *FleetRepository) GetOrderDetail(orderID, priceID, organizationID string) (*model.OrderDetailResponse, error) {
	query := fmt.Sprintf(`
        SELECT 
            fo.order_id, fo.created_at, 
            f.fleet_name, 
            fp.rent_type, fp.duration, COALESCE(fp.uom, '') as duration_uom, fp.price, 
            fo.unit_qty, fo.total_amount,
            fo.pickup_location, fo.pickup_city_id, fo.start_date, fo.end_date,
            COALESCE(foc.customer_name, '') as customer_name, COALESCE(foc.customer_phone, '') as customer_phone, COALESCE(foc.customer_email, '') as customer_email, COALESCE(foc.customer_address, '') as customer_address
        FROM fleet_orders fo
        JOIN fleets f ON fo.fleet_id = f.uuid
        JOIN fleet_prices fp ON fo.price_id = fp.uuid
        LEFT JOIN fleet_order_customers foc ON fo.order_id = foc.order_id
        WHERE fo.order_id = %s AND fo.price_id = %s AND fo.organization_id = %s
    `, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	var res model.OrderDetailResponse
	var createdAt time.Time
	var pickupCityID string
	var startDate, endDate time.Time

	err := r.db.QueryRow(query, orderID, priceID, organizationID).Scan(
		&res.OrderID, &createdAt,
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
	res.Pickup.PickupCity = pickupCityID // Store ID temporarily
	res.Pickup.StartDate = startDate.Format("2006-01-02")
	res.Pickup.EndDate = endDate.Format("2006-01-02")

	// Destinations
	destQuery := fmt.Sprintf(`SELECT city_id, location FROM fleet_order_destinations WHERE order_id = %s`, r.getPlaceholder(1))
	dRows, err := r.db.Query(destQuery, orderID)
	if err != nil {
		fmt.Println("Error querying order destinations:", err)
		return nil, err
	}
	defer dRows.Close()
	for dRows.Next() {
		var d model.OrderDetailDest
		var cID string
		if err := dRows.Scan(&cID, &d.Location); err == nil {
			d.City = cID // Store ID temporarily
			res.Destination = append(res.Destination, d)
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
			op.payment_remaining, op.status, op.created_at as payment_date,
			COALESCE(NULLIF(op.unique_code, '')::INTEGER, 0) as unique_code
		FROM fleet_order_payment op 
		INNER JOIN organization_bank_accounts ba ON op.payment_method = ba.bank_account_id 
		INNER JOIN bank_list bl ON bl.code = ba.bank_code 
		WHERE op.order_id = %s AND op.status > 0
	`, r.getPlaceholder(1))

	pRows, err := r.db.Query(paymentQuery, orderID)
	if err != nil {
		fmt.Println("Error querying order payments:", err)
		return nil, err
	}
	defer pRows.Close()

	res.Payment = make([]model.PaymentDetail, 0)
	for pRows.Next() {
		var p model.PaymentDetail
		var paymentDate time.Time
		err := pRows.Scan(
			&p.BankCode, &p.AccountName, &p.AccountNumber, &p.BankName,
			&p.PaymentType, &p.PaymentPercentage, &p.PaymentAmount, &p.TotalAmount,
			&p.PaymentRemaining, &p.Status, &paymentDate, &p.UniqueCode,
		)
		if err != nil {
			fmt.Println("Error scanning payment row:", err)
			continue
		}
		p.PaymentDate = paymentDate.Format("2006-01-02 15:04:05")
		res.Payment = append(res.Payment, p)
	}

	if len(res.Payment) == 0 {
		res.PaymentStatus = "Dibatalkan"
	} else {
		countType2 := 0
		anyStatus1 := false
		allStatus1 := true
		hasStatus10 := false
		for _, p := range res.Payment {
			if p.PaymentType == 2 {
				countType2++
			}
			if p.Status == 1 {
				anyStatus1 = true
			}
			if p.Status != 1 {
				allStatus1 = false
			}
			if p.Status == 10 {
				hasStatus10 = true
			}
		}
		if countType2 > 0 && len(res.Payment) > 1 && anyStatus1 && !allStatus1 {
			res.PaymentStatus = "Belum Lunas"
		} else if hasStatus10 {
			res.PaymentStatus = "Menunggu verifikasi"
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
