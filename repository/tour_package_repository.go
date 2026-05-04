package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"service-travego/database"
	"service-travego/model"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TourPackageRepository struct {
	db     *sql.DB
	driver string
}

type CreateTourPackageOrderInput struct {
	OrderID          string
	OrganizationID   string
	UserID           string
	TourPackageID    string
	CustomerID       string
	StartDate        string
	EndDate          string
	PickupAddress    string
	PickupCityID     string
	DiscountAmount   float64
	AdditionalAmount float64
	OfficialPax      int
	MemberPax        int
	TotalPax         int
	TotalAmount      float64
	AddonIDs         []string
}

type UpdateTourPackageOrderInput struct {
	OrderID          string
	OrganizationID   string
	UserID           string
	TourPackageID    string
	CustomerID       string
	StartDate        string
	EndDate          string
	PickupAddress    string
	PickupCityID     string
	DiscountAmount   float64
	AdditionalAmount float64
	OfficialPax      int
	MemberPax        int
	TotalPax         int
	TotalAmount      float64
	AddonIDs         []string
}

const softDeleteTourPackagePostgres = `
UPDATE tour_packages
SET status = 0, updated_at = $1, updated_by = $2
WHERE uuid = $3 AND organization_id::text = $4
`

const softDeleteTourPackageMySQL = `
UPDATE tour_packages
SET status = 0, updated_at = ?, updated_by = ?
WHERE uuid = ? AND organization_id = ?
`

func NewTourPackageRepository(db *sql.DB, driver string) *TourPackageRepository {
	return &TourPackageRepository{
		db:     db,
		driver: driver,
	}
}

func (r *TourPackageRepository) getPlaceholder(pos int) string {
	if r.driver == "postgres" || r.driver == "pgx" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}

func scanRowsToMaps(rows *sql.Rows) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	items := make([]map[string]interface{}, 0)
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		out := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			v := values[i]
			if b, ok := v.([]byte); ok {
				out[col] = string(b)
			} else {
				out[col] = v
			}
		}
		items = append(items, out)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func scanSingleRowToMap(rows *sql.Rows) (map[string]interface{}, error) {
	items, err := scanRowsToMaps(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, sql.ErrNoRows
	}
	return items[0], nil
}

func (r *TourPackageRepository) SoftDeleteTourPackage(ctx context.Context, orgID, userID, packageID string) error {
	query := softDeleteTourPackageMySQL
	if r.driver == "postgres" || r.driver == "pgx" {
		query = softDeleteTourPackagePostgres
	}
	res, err := database.ExecContext(ctx, r.db, query, time.Now(), userID, packageID, orgID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *TourPackageRepository) GetTourPackagesByOrgID(orgID string) ([]model.TourPackageListItem, error) {
	query := `
		SELECT 
			tp.uuid AS package_id,
			tp.package_name,
			tp.thumbnail,
			tp.package_description,
			tp.status,
			tp.active,
			MIN(tpp.min_pax) AS min_pax,
			MIN(tpp.price) AS min_price,
			MAX(tpp.min_pax) AS max_pax,
			MAX(tpp.price) AS max_price
		FROM tour_packages tp
		LEFT JOIN tour_package_prices tpp 
			ON tpp.package_id = tp.uuid
		WHERE tp.organization_id = %s 
		  AND tp.status = 1 AND tp.active = true
		GROUP BY tp.uuid, tp.package_name, tp.thumbnail, tp.package_description, tp.status, tp.active
	`

	// Set query placeholder
	query = fmt.Sprintf(query, r.getPlaceholder(1))

	rows, err := database.Query(r.db, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.TourPackageListItem{} // Initialize empty slice
	for rows.Next() {
		var item model.TourPackageListItem

		var thumbnail, description sql.NullString
		var status sql.NullInt64
		var active sql.NullBool
		var minPax, maxPax sql.NullInt64
		var minPrice, maxPrice sql.NullFloat64

		err := rows.Scan(
			&item.PackageID,
			&item.PackageName,
			&thumbnail,
			&description,
			&status,
			&active,
			&minPax,
			&minPrice,
			&maxPax,
			&maxPrice,
		)
		if err != nil {
			return nil, err
		}

		if thumbnail.Valid {
			item.Thumbnail = thumbnail.String
		}
		if description.Valid {
			item.PackageDescription = description.String
		}
		if minPax.Valid {
			item.MinPax = int(minPax.Int64)
		}
		if maxPax.Valid {
			item.MaxPax = int(maxPax.Int64)
		}
		if minPrice.Valid {
			item.MinPrice = minPrice.Float64
			item.Price = minPrice.Float64
		}
		if maxPrice.Valid {
			item.MaxPrice = maxPrice.Float64
		}
		if status.Valid {
			item.Status = int(status.Int64)
		}
		if active.Valid {
			item.Active = active.Bool
		}

		items = append(items, item)
	}

	return items, nil
}

func (r *TourPackageRepository) ListTourPackageOrders(orgID string) ([]model.TourPackageOrderListItem, error) {
	query := fmt.Sprintf(`
		SELECT
			tpo.order_id,
			tp.package_name,
			tp.uuid AS package_id,
			tpo.total_pax,
			tpo.customer_id,
			c.customer_name,
			tpo.start_date,
			tpo.end_date
		FROM tour_package_orders tpo
		INNER JOIN tour_packages tp ON tp.uuid = tpo.tour_package_id
		INNER JOIN customers c ON c.customer_id = tpo.customer_id
		WHERE tp.organization_id = %s AND c.organization_id = %s
		ORDER BY tpo.start_date DESC
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	rows, err := database.Query(r.db, query, orgID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.TourPackageOrderListItem, 0)
	for rows.Next() {
		var it model.TourPackageOrderListItem
		var packageName sql.NullString
		var customerName sql.NullString
		var totalPax sql.NullInt64
		var startDate sql.NullTime
		var endDate sql.NullTime

		if err := rows.Scan(
			&it.OrderID,
			&packageName,
			&it.PackageID,
			&totalPax,
			&it.CustomerID,
			&customerName,
			&startDate,
			&endDate,
		); err != nil {
			return nil, err
		}

		if packageName.Valid {
			it.PackageName = packageName.String
		}
		if customerName.Valid {
			it.CustomerName = customerName.String
		}
		if totalPax.Valid {
			it.TotalPax = int(totalPax.Int64)
		}
		if startDate.Valid {
			t := startDate.Time
			it.StartDate = &t
		}
		if endDate.Valid {
			t := endDate.Time
			it.EndDate = &t
		}

		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *TourPackageRepository) CustomerExistsByOrgID(orgID, customerID string) (bool, error) {
	query := fmt.Sprintf(
		"SELECT 1 FROM customers WHERE customer_id = %s AND organization_id = %s LIMIT 1",
		r.getPlaceholder(1),
		r.getPlaceholder(2),
	)
	var one int
	err := database.QueryRow(r.db, query, customerID, orgID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *TourPackageRepository) TourPackageExistsByOrgID(orgID, packageID string) (bool, error) {
	query := fmt.Sprintf(
		"SELECT 1 FROM tour_packages WHERE uuid = %s AND organization_id = %s LIMIT 1",
		r.getPlaceholder(1),
		r.getPlaceholder(2),
	)
	var one int
	err := database.QueryRow(r.db, query, packageID, orgID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *TourPackageRepository) TourPackageOrderExistsByOrgID(orgID, orderID string) (bool, error) {
	query := fmt.Sprintf(
		"SELECT 1 FROM tour_package_orders WHERE order_id = %s AND organization_id = %s LIMIT 1",
		r.getPlaceholder(1),
		r.getPlaceholder(2),
	)
	var one int
	err := database.QueryRow(r.db, query, orderID, orgID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *TourPackageRepository) GetTourPackagePriceByID(orgID, priceID string) (float64, bool, error) {
	query := fmt.Sprintf(
		"SELECT price FROM tour_package_prices WHERE uuid = %s AND organization_id = %s LIMIT 1",
		r.getPlaceholder(1),
		r.getPlaceholder(2),
	)
	var price sql.NullFloat64
	err := database.QueryRow(r.db, query, priceID, orgID).Scan(&price)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if !price.Valid {
		return 0, false, nil
	}
	return price.Float64, true, nil
}

func (r *TourPackageRepository) GetTourPackageAddonTotalByIDs(orgID string, addonIDs []string) (float64, bool, error) {
	if len(addonIDs) == 0 {
		return 0, true, nil
	}
	ph := make([]string, 0, len(addonIDs))
	args := make([]interface{}, 0, len(addonIDs)+1)
	pos := 1
	for _, id := range addonIDs {
		ph = append(ph, r.getPlaceholder(pos))
		args = append(args, id)
		pos++
	}
	orgPH := r.getPlaceholder(pos)
	args = append(args, orgID)

	query := fmt.Sprintf(
		"SELECT COUNT(*), COALESCE(SUM(price), 0) FROM tour_package_addons WHERE uuid IN (%s) AND organization_id = %s",
		strings.Join(ph, ","),
		orgPH,
	)
	var cnt int
	var total float64
	if err := database.QueryRow(r.db, query, args...).Scan(&cnt, &total); err != nil {
		return 0, false, err
	}
	return total, cnt == len(addonIDs), nil
}

func (r *TourPackageRepository) GetTourPackageOrderCountByOrgID(orgID string) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM tour_package_orders WHERE organization_id = %s", r.getPlaceholder(1))
	var count int
	err := database.QueryRow(r.db, query, orgID).Scan(&count)
	return count, err
}

func (r *TourPackageRepository) GetOrganizationCodeByOrgID(orgID string) (string, error) {
	query := fmt.Sprintf("SELECT organization_code FROM organizations WHERE organization_id = %s", r.getPlaceholder(1))
	var code string
	err := database.QueryRow(r.db, query, orgID).Scan(&code)
	return code, err
}

func (r *TourPackageRepository) CreateTourPackageOrder(ctx context.Context, in CreateTourPackageOrderInput) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	orderUUID := uuid.New().String()

	query := `INSERT INTO tour_package_orders (uuid, order_id, tour_package_id, customer_id, start_date, end_date, pickup_address, pickup_city_id, discount_amount, additional_amount, official_pax, member_pax, total_pax, total_amount, created_at, created_by, status, payment_status, organization_id) VALUES `
	if r.driver == "postgres" || r.driver == "pgx" {
		query += `($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`
	} else {
		query += `(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}

	if _, err := database.TxExecContext(
		ctx,
		tx,
		query,
		orderUUID,
		in.OrderID,
		in.TourPackageID,
		in.CustomerID,
		in.StartDate,
		in.EndDate,
		in.PickupAddress,
		in.PickupCityID,
		in.DiscountAmount,
		in.AdditionalAmount,
		in.OfficialPax,
		in.MemberPax,
		in.TotalPax,
		in.TotalAmount,
		now,
		in.UserID,
		1,
		0,
		in.OrganizationID,
	); err != nil {
		return err
	}

	if len(in.AddonIDs) > 0 {
		addonQuery := `INSERT INTO tour_package_order_addons (order_id, organization_id, addon_id, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			addonQuery += `($1, $2, $3, $4, $5)`
		} else {
			addonQuery += `(?, ?, ?, ?, ?)`
		}
		for _, addonID := range in.AddonIDs {
			if _, err := database.TxExecContext(ctx, tx, addonQuery, in.OrderID, in.OrganizationID, addonID, now, in.UserID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *TourPackageRepository) UpdateTourPackageOrder(ctx context.Context, in UpdateTourPackageOrderInput) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()

	updateQuery := fmt.Sprintf(`
		UPDATE tour_package_orders
		SET
			tour_package_id = %s,
			customer_id = %s,
			start_date = %s,
			end_date = %s,
			pickup_address = %s,
			pickup_city_id = %s,
			discount_amount = %s,
			additional_amount = %s,
			official_pax = %s,
			member_pax = %s,
			total_pax = %s,
			total_amount = %s,
			updated_at = %s,
			updated_by = %s
		WHERE order_id = %s AND organization_id = %s
	`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
		r.getPlaceholder(3),
		r.getPlaceholder(4),
		r.getPlaceholder(5),
		r.getPlaceholder(6),
		r.getPlaceholder(7),
		r.getPlaceholder(8),
		r.getPlaceholder(9),
		r.getPlaceholder(10),
		r.getPlaceholder(11),
		r.getPlaceholder(12),
		r.getPlaceholder(13),
		r.getPlaceholder(14),
		r.getPlaceholder(15),
		r.getPlaceholder(16),
	)

	res, err := database.TxExecContext(
		ctx,
		tx,
		updateQuery,
		in.TourPackageID,
		in.CustomerID,
		in.StartDate,
		in.EndDate,
		in.PickupAddress,
		in.PickupCityID,
		in.DiscountAmount,
		in.AdditionalAmount,
		in.OfficialPax,
		in.MemberPax,
		in.TotalPax,
		in.TotalAmount,
		now,
		in.UserID,
		in.OrderID,
		in.OrganizationID,
	)
	if err != nil {
		return err
	}
	ra, _ := res.RowsAffected()
	if ra == 0 {
		return sql.ErrNoRows
	}

	delQuery := fmt.Sprintf(
		"DELETE FROM tour_package_order_addons WHERE order_id = %s AND organization_id = %s",
		r.getPlaceholder(1),
		r.getPlaceholder(2),
	)
	if _, err := database.TxExecContext(ctx, tx, delQuery, in.OrderID, in.OrganizationID); err != nil {
		return err
	}

	if len(in.AddonIDs) > 0 {
		addonQuery := `INSERT INTO tour_package_order_addons (order_id, organization_id, addon_id, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			addonQuery += `($1, $2, $3, $4, $5)`
		} else {
			addonQuery += `(?, ?, ?, ?, ?)`
		}
		for _, addonID := range in.AddonIDs {
			if _, err := database.TxExecContext(ctx, tx, addonQuery, in.OrderID, in.OrganizationID, addonID, now, in.UserID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *TourPackageRepository) GetTourPackageOrderDetail(ctx context.Context, orgID, orderID string) (map[string]interface{}, []map[string]interface{}, error) {
	orderExpr := "order_id = " + r.getPlaceholder(1)
	orgExpr := "organization_id = " + r.getPlaceholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(2)
	}

	query := "SELECT * FROM tour_package_orders WHERE " + orderExpr + " AND " + orgExpr + " LIMIT 1"
	orderRows, err := database.QueryContext(ctx, r.db, query, orderID, orgID)
	if err != nil {
		return nil, nil, err
	}
	defer orderRows.Close()

	orderMap, err := scanSingleRowToMap(orderRows)
	if err != nil {
		return nil, nil, err
	}
	if _, ok := orderMap["order_id"]; !ok {
		orderMap["order_id"] = orderID
	}

	addons := make([]map[string]interface{}, 0)
	oaOrgExpr := "oa.organization_id = " + r.getPlaceholder(2)
	aOrgExpr := "a.organization_id = " + r.getPlaceholder(3)
	if r.driver == "postgres" || r.driver == "pgx" {
		oaOrgExpr = "oa.organization_id::text = " + r.getPlaceholder(2)
		aOrgExpr = "a.organization_id::text = " + r.getPlaceholder(3)
	}
	addonQuery := fmt.Sprintf(`
		SELECT a.uuid, a.price, a.description
		FROM tour_package_order_addons oa
		INNER JOIN tour_package_addons a ON a.uuid = oa.addon_id
		WHERE oa.order_id = %s AND %s AND %s
		ORDER BY oa.created_at ASC
	`, r.getPlaceholder(1), oaOrgExpr, aOrgExpr)
	addonRows, err := database.QueryContext(ctx, r.db, addonQuery, orderID, orgID, orgID)
	if err != nil {
		return nil, nil, err
	}
	defer addonRows.Close()
	addons, err = scanRowsToMaps(addonRows)
	if err != nil {
		return nil, nil, err
	}

	return orderMap, addons, nil
}

func (r *TourPackageRepository) GetCustomerInfoByOrgID(ctx context.Context, orgID, customerID string) (map[string]interface{}, error) {
	custIDExpr := "customer_id = " + r.getPlaceholder(1)
	orgExpr := "organization_id = " + r.getPlaceholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(2)
	}

	query := `
		SELECT customer_id, customer_name, customer_phone, customer_email
		FROM customers
		WHERE ` + custIDExpr + ` AND ` + orgExpr + `
		LIMIT 1
	`
	rows, err := database.QueryContext(ctx, r.db, query, customerID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSingleRowToMap(rows)
}

func (r *TourPackageRepository) CreateTourPackage(ctx context.Context, req *model.CreateTourPackageRequest, packageID, orgID, userID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()

	// 1. Insert into tour_packages
	query := `INSERT INTO tour_packages (uuid, package_name, package_type, package_description, active, thumbnail, organization_id, created_by, created_at, status) VALUES `
	if r.driver == "postgres" || r.driver == "pgx" {
		query += `($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	} else {
		query += `(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}

	_, err = database.TxExecContext(ctx, tx, query,
		packageID,
		req.PackageName,
		req.PackageType,
		req.PackageDescription,
		req.Active,
		req.Thumbnail,
		orgID,
		userID,
		now,
		1, // Default status 1
	)
	if err != nil {
		log.Printf("[ERROR] CreateTourPackage failed - Path: %s, Error: %v", ctx.Value("path"), err)
		return err
	}

	// 2. Addons
	if len(req.Addons) > 0 {
		addonQuery := `INSERT INTO tour_package_addons (uuid, package_id, organization_id, description, price, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			addonQuery += `($1, $2, $3, $4, $5, $6, $7)`
		} else {
			addonQuery += `(?, ?, ?, ?, ?, ?, ?)`
		}

		for _, addon := range req.Addons {
			_, err = database.TxExecContext(ctx, tx, addonQuery, uuid.New().String(), packageID, orgID, addon.Description, addon.Price, now, userID)
			if err != nil {
				return err
			}
		}
	}

	// 3. Facilities
	if len(req.Facilities) > 0 {
		facilityQuery := `INSERT INTO tour_package_facilities (uuid, package_id, organization_id, facility, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			facilityQuery += `($1, $2, $3, $4, $5, $6)`
		} else {
			facilityQuery += `(?, ?, ?, ?, ?, ?)`
		}

		for _, facility := range req.Facilities {
			_, err = database.TxExecContext(ctx, tx, facilityQuery, uuid.New().String(), packageID, orgID, facility, now, userID)
			if err != nil {
				return err
			}
		}
	}

	// 4. Itineraries
	if len(req.Itineraries) > 0 {
		itinQuery := `INSERT INTO tour_package_itineraries (uuid, package_id, organization_id, day, activity, location, city_id, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			itinQuery += `($1, $2, $3, $4, $5, $6, $7, $8, $9)`
		} else {
			itinQuery += `(?, ?, ?, ?, ?, ?, ?, ?, ?)`
		}

		for _, day := range req.Itineraries {
			for _, act := range day.Activities {
				activityTime := act.Time
				if activityTime == "" {
					activityTime = "00:00:00"
				}
				_, err = database.TxExecContext(ctx, tx, itinQuery, uuid.New().String(), packageID, orgID, activityTime, act.Description, act.Location, act.City.ID, now, userID)
				if err != nil {
					return err
				}
			}
		}
	}

	// 5. Pickup Areas
	if len(req.PickupAreas) > 0 {
		pickupQuery := `INSERT INTO tour_package_pickup (uuid, package_id, organization_id, city_id, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			pickupQuery += `($1, $2, $3, $4, $5, $6)`
		} else {
			pickupQuery += `(?, ?, ?, ?, ?, ?)`
		}

		for _, area := range req.PickupAreas {
			_, err = database.TxExecContext(ctx, tx, pickupQuery, uuid.New().String(), packageID, orgID, area.ID, now, userID)
			if err != nil {
				return err
			}
		}
	}

	// 6. Prices
	if len(req.Pricing) > 0 {
		priceQuery := `INSERT INTO tour_package_prices (uuid, package_id, organization_id, min_pax, max_pax, price, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			priceQuery += `($1, $2, $3, $4, $5, $6, $7, $8)`
		} else {
			priceQuery += `(?, ?, ?, ?, ?, ?, ?, ?)`
		}

		for _, price := range req.Pricing {
			_, err = database.TxExecContext(ctx, tx, priceQuery, uuid.New().String(), packageID, orgID, price.MinPax, price.MaxPax, price.Price, now, userID)
			if err != nil {
				return err
			}
		}
	}

	// 7. Images
	if len(req.Images) > 0 {
		imageQuery := `INSERT INTO tour_package_images (uuid, package_id, organization_id, image_path, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			imageQuery += `($1, $2, $3, $4, $5, $6)`
		} else {
			imageQuery += `(?, ?, ?, ?, ?, ?)`
		}

		for _, img := range req.Images {
			_, err = database.TxExecContext(ctx, tx, imageQuery, uuid.New().String(), packageID, orgID, img, now, userID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *TourPackageRepository) UpdateTourPackage(ctx context.Context, req *model.UpdateTourPackageRequest, orgID, userID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()

	updateQuery := `UPDATE tour_packages SET package_name = %s, package_type = %s, package_description = %s, active = %s, thumbnail = %s, updated_at = %s, updated_by = %s WHERE uuid = %s AND organization_id = %s`
	updateQuery = fmt.Sprintf(
		updateQuery,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
		r.getPlaceholder(3),
		r.getPlaceholder(4),
		r.getPlaceholder(5),
		r.getPlaceholder(6),
		r.getPlaceholder(7),
		r.getPlaceholder(8),
		r.getPlaceholder(9),
	)

	res, err := database.TxExecContext(
		ctx,
		tx,
		updateQuery,
		req.PackageName,
		req.PackageType,
		req.PackageDescription,
		req.Active,
		req.Thumbnail,
		now,
		userID,
		req.PackageID,
		orgID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return sql.ErrNoRows
	}

	if req.Addons != nil {
		keep := make([]string, 0, len(req.Addons))
		for _, it := range req.Addons {
			if it.UUID == "" {
				newID := uuid.New().String()
				ins := `INSERT INTO tour_package_addons (uuid, package_id, organization_id, description, price, created_at, created_by) VALUES `
				if r.driver == "postgres" || r.driver == "pgx" {
					ins += `($1, $2, $3, $4, $5, $6, $7)`
				} else {
					ins += `(?, ?, ?, ?, ?, ?, ?)`
				}
				if _, err := database.TxExecContext(ctx, tx, ins, newID, req.PackageID, orgID, it.Description, it.Price, now, userID); err != nil {
					return err
				}
				keep = append(keep, newID)
				continue
			}

			upd := `UPDATE tour_package_addons SET description = %s, price = %s WHERE uuid = %s AND package_id = %s AND organization_id = %s`
			upd = fmt.Sprintf(upd, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5))
			if _, err := database.TxExecContext(ctx, tx, upd, it.Description, it.Price, it.UUID, req.PackageID, orgID); err != nil {
				return err
			}
			keep = append(keep, it.UUID)
		}

		if len(keep) == 0 {
			del := fmt.Sprintf("DELETE FROM tour_package_addons WHERE package_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := database.TxExecContext(ctx, tx, del, req.PackageID, orgID); err != nil {
				return err
			}
		} else {
			ph := make([]string, 0, len(keep))
			args := make([]interface{}, 0, 2+len(keep))
			args = append(args, req.PackageID, orgID)
			for i, id := range keep {
				ph = append(ph, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			del := fmt.Sprintf("DELETE FROM tour_package_addons WHERE package_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(ph, ","))
			if _, err := database.TxExecContext(ctx, tx, del, args...); err != nil {
				return err
			}
		}
	}

	if req.Facilities != nil {
		keep := make([]string, 0, len(req.Facilities))
		for _, it := range req.Facilities {
			if it.UUID == "" {
				newID := uuid.New().String()
				ins := `INSERT INTO tour_package_facilities (uuid, package_id, organization_id, facility, created_at, created_by) VALUES `
				if r.driver == "postgres" || r.driver == "pgx" {
					ins += `($1, $2, $3, $4, $5, $6)`
				} else {
					ins += `(?, ?, ?, ?, ?, ?)`
				}
				if _, err := database.TxExecContext(ctx, tx, ins, newID, req.PackageID, orgID, it.Facility, now, userID); err != nil {
					return err
				}
				keep = append(keep, newID)
				continue
			}
			upd := `UPDATE tour_package_facilities SET facility = %s, updated_at = %s, updated_by = %s WHERE uuid = %s AND package_id = %s AND organization_id = %s`
			upd = fmt.Sprintf(upd, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
			if _, err := database.TxExecContext(ctx, tx, upd, it.Facility, now, userID, it.UUID, req.PackageID, orgID); err != nil {
				return err
			}
			keep = append(keep, it.UUID)
		}

		if len(keep) == 0 {
			del := fmt.Sprintf("DELETE FROM tour_package_facilities WHERE package_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := database.TxExecContext(ctx, tx, del, req.PackageID, orgID); err != nil {
				return err
			}
		} else {
			ph := make([]string, 0, len(keep))
			args := make([]interface{}, 0, 2+len(keep))
			args = append(args, req.PackageID, orgID)
			for i, id := range keep {
				ph = append(ph, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			del := fmt.Sprintf("DELETE FROM tour_package_facilities WHERE package_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(ph, ","))
			if _, err := database.TxExecContext(ctx, tx, del, args...); err != nil {
				return err
			}
		}
	}

	if req.PickupAreas != nil {
		keep := make([]string, 0, len(req.PickupAreas))
		for _, it := range req.PickupAreas {
			if it.UUID == "" {
				newID := uuid.New().String()
				ins := `INSERT INTO tour_package_pickup (uuid, package_id, organization_id, city_id, created_at, created_by) VALUES `
				if r.driver == "postgres" || r.driver == "pgx" {
					ins += `($1, $2, $3, $4, $5, $6)`
				} else {
					ins += `(?, ?, ?, ?, ?, ?)`
				}
				if _, err := database.TxExecContext(ctx, tx, ins, newID, req.PackageID, orgID, it.ID, now, userID); err != nil {
					return err
				}
				keep = append(keep, newID)
				continue
			}
			upd := `UPDATE tour_package_pickup SET city_id = %s, updated_at = %s, updated_by = %s WHERE uuid = %s AND package_id = %s AND organization_id = %s`
			upd = fmt.Sprintf(upd, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
			if _, err := database.TxExecContext(ctx, tx, upd, it.ID, now, userID, it.UUID, req.PackageID, orgID); err != nil {
				return err
			}
			keep = append(keep, it.UUID)
		}

		if len(keep) == 0 {
			del := fmt.Sprintf("DELETE FROM tour_package_pickup WHERE package_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := database.TxExecContext(ctx, tx, del, req.PackageID, orgID); err != nil {
				return err
			}
		} else {
			ph := make([]string, 0, len(keep))
			args := make([]interface{}, 0, 2+len(keep))
			args = append(args, req.PackageID, orgID)
			for i, id := range keep {
				ph = append(ph, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			del := fmt.Sprintf("DELETE FROM tour_package_pickup WHERE package_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(ph, ","))
			if _, err := database.TxExecContext(ctx, tx, del, args...); err != nil {
				return err
			}
		}
	}

	if req.Pricing != nil {
		keep := make([]string, 0, len(req.Pricing))
		for _, it := range req.Pricing {
			if it.UUID == "" {
				newID := uuid.New().String()
				ins := `INSERT INTO tour_package_prices (uuid, package_id, organization_id, min_pax, max_pax, price, created_at, created_by) VALUES `
				if r.driver == "postgres" || r.driver == "pgx" {
					ins += `($1, $2, $3, $4, $5, $6, $7, $8)`
				} else {
					ins += `(?, ?, ?, ?, ?, ?, ?, ?)`
				}
				if _, err := database.TxExecContext(ctx, tx, ins, newID, req.PackageID, orgID, it.MinPax, it.MaxPax, it.Price, now, userID); err != nil {
					return err
				}
				keep = append(keep, newID)
				continue
			}
			upd := `UPDATE tour_package_prices SET min_pax = %s, max_pax = %s, price = %s, updated_at = %s, updated_by = %s WHERE uuid = %s AND package_id = %s AND organization_id = %s`
			upd = fmt.Sprintf(upd, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8))
			if _, err := database.TxExecContext(ctx, tx, upd, it.MinPax, it.MaxPax, it.Price, now, userID, it.UUID, req.PackageID, orgID); err != nil {
				return err
			}
			keep = append(keep, it.UUID)
		}

		if len(keep) == 0 {
			del := fmt.Sprintf("DELETE FROM tour_package_prices WHERE package_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := database.TxExecContext(ctx, tx, del, req.PackageID, orgID); err != nil {
				return err
			}
		} else {
			ph := make([]string, 0, len(keep))
			args := make([]interface{}, 0, 2+len(keep))
			args = append(args, req.PackageID, orgID)
			for i, id := range keep {
				ph = append(ph, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			del := fmt.Sprintf("DELETE FROM tour_package_prices WHERE package_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(ph, ","))
			if _, err := database.TxExecContext(ctx, tx, del, args...); err != nil {
				return err
			}
		}
	}

	if req.Images != nil {
		keep := make([]string, 0, len(req.Images))
		for _, it := range req.Images {
			if it.UUID == "" {
				newID := uuid.New().String()
				ins := `INSERT INTO tour_package_images (uuid, package_id, organization_id, image_path, created_at, created_by) VALUES `
				if r.driver == "postgres" || r.driver == "pgx" {
					ins += `($1, $2, $3, $4, $5, $6)`
				} else {
					ins += `(?, ?, ?, ?, ?, ?)`
				}
				if _, err := database.TxExecContext(ctx, tx, ins, newID, req.PackageID, orgID, it.ImagePath, now, userID); err != nil {
					return err
				}
				keep = append(keep, newID)
				continue
			}
			upd := `UPDATE tour_package_images SET image_path = %s WHERE uuid = %s AND package_id = %s AND organization_id = %s`
			upd = fmt.Sprintf(upd, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))
			if _, err := database.TxExecContext(ctx, tx, upd, it.ImagePath, it.UUID, req.PackageID, orgID); err != nil {
				return err
			}
			keep = append(keep, it.UUID)
		}

		if len(keep) == 0 {
			del := fmt.Sprintf("DELETE FROM tour_package_images WHERE package_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := database.TxExecContext(ctx, tx, del, req.PackageID, orgID); err != nil {
				return err
			}
		} else {
			ph := make([]string, 0, len(keep))
			args := make([]interface{}, 0, 2+len(keep))
			args = append(args, req.PackageID, orgID)
			for i, id := range keep {
				ph = append(ph, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			del := fmt.Sprintf("DELETE FROM tour_package_images WHERE package_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(ph, ","))
			if _, err := database.TxExecContext(ctx, tx, del, args...); err != nil {
				return err
			}
		}
	}

	if req.Itineraries != nil {
		keep := make([]string, 0)
		for _, day := range req.Itineraries {
			for _, act := range day.Activities {
				activityTime := act.Time
				if activityTime == "" {
					activityTime = "00:00:00"
				}

				if act.UUID == "" {
					newID := uuid.New().String()
					ins := `INSERT INTO tour_package_itineraries (uuid, package_id, organization_id, day, activity, location, city_id, created_at, created_by) VALUES `
					if r.driver == "postgres" || r.driver == "pgx" {
						ins += `($1, $2, $3, $4, $5, $6, $7, $8, $9)`
					} else {
						ins += `(?, ?, ?, ?, ?, ?, ?, ?, ?)`
					}
					if _, err := database.TxExecContext(ctx, tx, ins, newID, req.PackageID, orgID, activityTime, act.Description, act.Location, act.City.ID, now, userID); err != nil {
						return err
					}
					keep = append(keep, newID)
					continue
				}

				upd := `UPDATE tour_package_itineraries SET day = %s, activity = %s, location = %s, city_id = %s, updated_at = %s, updated_by = %s WHERE uuid = %s AND package_id = %s AND organization_id = %s`
				upd = fmt.Sprintf(upd, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9))
				if _, err := database.TxExecContext(ctx, tx, upd, activityTime, act.Description, act.Location, act.City.ID, now, userID, act.UUID, req.PackageID, orgID); err != nil {
					return err
				}
				keep = append(keep, act.UUID)
			}
		}

		if len(keep) == 0 {
			del := fmt.Sprintf("DELETE FROM tour_package_itineraries WHERE package_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := database.TxExecContext(ctx, tx, del, req.PackageID, orgID); err != nil {
				return err
			}
		} else {
			ph := make([]string, 0, len(keep))
			args := make([]interface{}, 0, 2+len(keep))
			args = append(args, req.PackageID, orgID)
			for i, id := range keep {
				ph = append(ph, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			del := fmt.Sprintf("DELETE FROM tour_package_itineraries WHERE package_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(ph, ","))
			if _, err := database.TxExecContext(ctx, tx, del, args...); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *TourPackageRepository) GetTourPackageDetail(ctx context.Context, orgID, packageID string) (*model.TourPackageDetailResponse, error) {
	detail := &model.TourPackageDetailResponse{
		Schedules:    []model.TourPackageScheduleItem{},
		Pricing:      []model.TourPackagePricing{},
		PickupAreas:  []model.TourPackagePickupAreaItem{},
		Images:       []string{},
		Itineraries:  []model.TourPackageItineraryDetailItem{},
		Facilities:   []string{},
		Destinations: []model.TourPackageDestinationItem{},
		Addons:       []model.TourPackageAddon{},
	}

	metaQuery := `
		SELECT uuid, package_name, package_type, package_description, thumbnail, duration, min_pax, max_pax, active, status
		FROM tour_packages
		WHERE uuid = %s AND organization_id = %s
		LIMIT 1
	`
	metaQuery = fmt.Sprintf(metaQuery, r.getPlaceholder(1), r.getPlaceholder(2))

	var (
		metaPackageID string
		packageName   sql.NullString
		packageType   sql.NullInt64
		packageDesc   sql.NullString
		thumbnail     sql.NullString
		duration      sql.NullInt64
		minPax        sql.NullInt64
		maxPax        sql.NullInt64
		active        sql.NullBool
		status        sql.NullInt64
	)

	err := database.QueryRowContext(ctx, r.db, metaQuery, packageID, orgID).Scan(
		&metaPackageID,
		&packageName,
		&packageType,
		&packageDesc,
		&thumbnail,
		&duration,
		&minPax,
		&maxPax,
		&active,
		&status,
	)
	if err != nil {
		return nil, err
	}

	detail.Meta = model.TourPackageDetailMeta{
		PackageID:          metaPackageID,
		PackageName:        packageName.String,
		PackageType:        int(packageType.Int64),
		PackageDescription: packageDesc.String,
		Thumbnail:          thumbnail.String,
		Duration:           int(duration.Int64),
		MinPax:             int(minPax.Int64),
		MaxPax:             int(maxPax.Int64),
		Active:             active.Bool,
		Status:             int(status.Int64),
	}

	scheduleQuery := `
		SELECT date_start, date_end
		FROM tour_package_schedules
		WHERE package_id = %s AND organization_id = %s
		ORDER BY date_start ASC
	`
	scheduleQuery = fmt.Sprintf(scheduleQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	scheduleRows, err := database.QueryContext(ctx, r.db, scheduleQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer scheduleRows.Close()
	for scheduleRows.Next() {
		var ds, de time.Time
		if err := scheduleRows.Scan(&ds, &de); err != nil {
			return nil, err
		}
		detail.Schedules = append(detail.Schedules, model.TourPackageScheduleItem{
			DateStart: ds.Format("2006-01-02"),
			DateEnd:   de.Format("2006-01-02"),
		})
	}

	priceQuery := `
		SELECT uuid, min_pax, max_pax, price
		FROM tour_package_prices
		WHERE package_id = %s AND organization_id = %s
		ORDER BY min_pax ASC, max_pax ASC
	`
	priceQuery = fmt.Sprintf(priceQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	priceRows, err := database.QueryContext(ctx, r.db, priceQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer priceRows.Close()
	for priceRows.Next() {
		var priceID sql.NullString
		var minPax, maxPax sql.NullInt64
		var priceVal sql.NullFloat64
		if err := priceRows.Scan(&priceID, &minPax, &maxPax, &priceVal); err != nil {
			return nil, err
		}
		detail.Pricing = append(detail.Pricing, model.TourPackagePricing{
			PriceID: priceID.String,
			MinPax:  int(minPax.Int64),
			MaxPax:  int(maxPax.Int64),
			Price:   priceVal.Float64,
		})
	}

	pickupQuery := `
		SELECT city_id
		FROM tour_package_pickup
		WHERE package_id = %s AND organization_id = %s
		ORDER BY city_id ASC
	`
	pickupQuery = fmt.Sprintf(pickupQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	pickupRows, err := database.QueryContext(ctx, r.db, pickupQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer pickupRows.Close()
	for pickupRows.Next() {
		var cityID sql.NullInt64
		if err := pickupRows.Scan(&cityID); err != nil {
			return nil, err
		}
		detail.PickupAreas = append(detail.PickupAreas, model.TourPackagePickupAreaItem{CityID: int(cityID.Int64)})
	}

	imageQuery := `
		SELECT image_path
		FROM tour_package_images
		WHERE package_id = %s AND organization_id = %s
		ORDER BY created_at ASC
	`
	imageQuery = fmt.Sprintf(imageQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	imageRows, err := database.QueryContext(ctx, r.db, imageQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer imageRows.Close()
	for imageRows.Next() {
		var img sql.NullString
		if err := imageRows.Scan(&img); err != nil {
			return nil, err
		}
		if img.Valid {
			detail.Images = append(detail.Images, img.String)
		}
	}

	itinQuery := `
		SELECT day, activity, location, city_id
		FROM tour_package_itineraries
		WHERE package_id = %s AND organization_id = %s
		ORDER BY created_at ASC
	`
	itinQuery = fmt.Sprintf(itinQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	itinRows, err := database.QueryContext(ctx, r.db, itinQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer itinRows.Close()
	for itinRows.Next() {
		var (
			tm       time.Time
			act      sql.NullString
			location sql.NullString
			cityID   sql.NullInt64
		)
		if err := itinRows.Scan(&tm, &act, &location, &cityID); err != nil {
			return nil, err
		}
		detail.Itineraries = append(detail.Itineraries, model.TourPackageItineraryDetailItem{
			Time:        tm.Format("15:04:05"),
			Description: act.String,
			Location:    location.String,
			CityID:      int(cityID.Int64),
		})
	}

	facilityQuery := `
		SELECT facility
		FROM tour_package_facilities
		WHERE package_id = %s AND organization_id = %s
		ORDER BY created_at ASC
	`
	facilityQuery = fmt.Sprintf(facilityQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	facilityRows, err := database.QueryContext(ctx, r.db, facilityQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer facilityRows.Close()
	for facilityRows.Next() {
		var facility sql.NullString
		if err := facilityRows.Scan(&facility); err != nil {
			return nil, err
		}
		if facility.Valid && facility.String != "" {
			detail.Facilities = append(detail.Facilities, facility.String)
		}
	}

	destQuery := `
		SELECT city_id, destination
		FROM tour_package_destinations
		WHERE package_id = %s AND organization_id = %s
		ORDER BY city_id ASC
	`
	destQuery = fmt.Sprintf(destQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	destRows, err := database.QueryContext(ctx, r.db, destQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer destRows.Close()
	for destRows.Next() {
		var cityID sql.NullInt64
		var destination sql.NullString
		if err := destRows.Scan(&cityID, &destination); err != nil {
			return nil, err
		}
		detail.Destinations = append(detail.Destinations, model.TourPackageDestinationItem{
			CityID:      int(cityID.Int64),
			Destination: destination.String,
		})
	}

	addonQuery := `
		SELECT uuid, description, price
		FROM tour_package_addons
		WHERE package_id = %s AND organization_id = %s
		ORDER BY created_at ASC
	`
	addonQuery = fmt.Sprintf(addonQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	addonRows, err := database.QueryContext(ctx, r.db, addonQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer addonRows.Close()
	for addonRows.Next() {
		var addonID sql.NullString
		var description sql.NullString
		var priceVal sql.NullFloat64
		if err := addonRows.Scan(&addonID, &description, &priceVal); err != nil {
			return nil, err
		}
		detail.Addons = append(detail.Addons, model.TourPackageAddon{
			AddonID:     addonID.String,
			Description: description.String,
			Price:       priceVal.Float64,
		})
	}

	return detail, nil
}
