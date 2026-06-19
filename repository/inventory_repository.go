package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"service-travego/database"
	"service-travego/model"
	"service-travego/utils"
	"strings"
	"time"

	"github.com/google/uuid"
)

type InventoryRepository struct {
	db      *sql.DB
	driver  string
	cityMap map[string]string
}

func NewInventoryRepository(db *sql.DB, driver string) *InventoryRepository {
	return &InventoryRepository{
		db:     db,
		driver: driver,
	}
}

func (r *InventoryRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

func (r *InventoryRepository) GetOrganizationCodeByOrgID(orgID string) (string, error) {
	query := fmt.Sprintf("SELECT organization_code FROM organizations WHERE organization_id = %s", r.getPlaceholder(1))
	var code string
	err := database.QueryRow(r.db, query, orgID).Scan(&code)
	if err == sql.ErrNoRows {
		return "", sql.ErrNoRows
	}
	return code, err
}

func (r *InventoryRepository) ensureLocationsLoaded() {
	if r.cityMap != nil {
		return
	}
	f, err := os.Open("config/location.json")
	if err != nil {
		r.cityMap = map[string]string{}
		return
	}
	defer f.Close()
	var loc model.Location
	if err := json.NewDecoder(f).Decode(&loc); err != nil {
		r.cityMap = map[string]string{}
		return
	}
	r.cityMap = make(map[string]string, len(loc.Cities))
	for _, c := range loc.Cities {
		r.cityMap[c.ID] = c.Name
	}
}

func (r *InventoryRepository) GetAllItems(organizationID string) ([]model.InventoryItemWithLabel, error) {
	var query string
	if r.driver == "mysql" {
		query = fmt.Sprintf(`
			SELECT i.item_id, i.item_sku, i.item_name, i.item_uom, i.item_category, i.status, SUM(ig.stock) AS total_stock, GROUP_CONCAT(g.garage_name SEPARATOR ', ') AS garage_names, MAX(ig.created_at) AS created_at, MAX(ig.updated_at) AS updated_at
			FROM inventory_items i
			INNER JOIN inventory_item_garage ig ON ig.item_id = i.item_id
			INNER JOIN garage g ON g.garage_id = ig.garage_id
			WHERE i.organization_id = %s AND i.status > 0
			GROUP BY i.item_id, i.item_sku, i.item_name, i.item_uom, i.item_category, i.status
		`, r.getPlaceholder(1))
	} else {
		query = fmt.Sprintf(`
			SELECT i.item_id, i.item_sku, i.item_name, i.item_uom, i.item_category, i.status, SUM(ig.stock) AS total_stock, STRING_AGG(g.garage_name, ', ') AS garage_names, MAX(ig.created_at) AS created_at, MAX(ig.updated_at) AS updated_at
			FROM inventory_items i
			INNER JOIN inventory_item_garage ig ON ig.item_id = i.item_id
			INNER JOIN garage g ON g.garage_id = ig.garage_id
			WHERE i.organization_id = %s AND i.status > 0
			GROUP BY i.item_id, i.item_sku, i.item_name, i.item_uom, i.item_category, i.status
		`, r.getPlaceholder(1))
	}

	rows, err := database.Query(r.db, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.InventoryItemWithLabel
	for rows.Next() {
		var item model.InventoryItemWithLabel
		var totalStock sql.NullInt64
		var updatedAt sql.NullTime
		var itemSKU sql.NullString
		if err := rows.Scan(
			&item.ItemID,
			&itemSKU,
			&item.ItemName,
			&item.ItemUOM,
			&item.ItemCategory,
			&item.Status,
			&totalStock,
			&item.GarageNames,
			&item.CreatedAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		if itemSKU.Valid {
			item.ItemSKU = itemSKU.String
		}
		if totalStock.Valid {
			item.TotalStock = int(totalStock.Int64)
		}
		if updatedAt.Valid {
			item.UpdatedAt = updatedAt.Time
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *InventoryRepository) GetItemByID(itemID, organizationID string) (*model.InventoryItem, error) {
	query := fmt.Sprintf(`
		SELECT item_id, organization_id, item_name, item_uom, item_category, stock, status,
		       created_at, created_by, updated_at, updated_by
		FROM inventory_items
		WHERE item_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	var item model.InventoryItem
	err := database.QueryRow(r.db, query, itemID, organizationID).Scan(
		&item.ItemID,
		&item.OrganizationID,
		&item.ItemName,
		&item.ItemUOM,
		&item.ItemCategory,
		&item.Stock,
		&item.Status,
		&item.CreatedAt,
		&item.CreatedBy,
		&item.UpdatedAt,
		&item.UpdatedBy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &item, nil
}

func (r *InventoryRepository) CreateItem(item *model.InventoryItem) error {
	item.ItemID = uuid.New().String()
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now

	sku, err := utils.GenerateItemSKU(r.db, r.driver, item.OrganizationID)
	if err != nil {
		return err
	}
	item.ItemSKU = sku

	query := fmt.Sprintf(`
		INSERT INTO inventory_items (item_id, item_sku, organization_id, item_name, item_uom, item_category, stock, status, created_at, created_by, updated_at, updated_by)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12),
	)

	_, err = database.Exec(r.db, query,
		item.ItemID,
		item.ItemSKU,
		item.OrganizationID,
		item.ItemName,
		item.ItemUOM,
		item.ItemCategory,
		item.Stock,
		1,
		item.CreatedAt,
		item.CreatedBy,
		item.UpdatedAt,
		item.UpdatedBy,
	)
	return err
}

func (r *InventoryRepository) UpdateItem(itemID, organizationID, updatedBy string, updates map[string]interface{}) error {
	now := time.Now()
	updates["updated_at"] = now
	updates["updated_by"] = updatedBy

	var setParts []string
	var args []interface{}
	pos := 1

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = %s", key, r.getPlaceholder(pos)))
		args = append(args, value)
		pos++
	}

	query := fmt.Sprintf("UPDATE inventory_items SET %s WHERE item_id = %s AND organization_id = %s",
		strings.Join(setParts, ", "),
		r.getPlaceholder(pos),
		r.getPlaceholder(pos+1),
	)

	args = append(args, itemID, organizationID)

	result, err := database.Exec(r.db, query, args...)
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

func (r *InventoryRepository) DeleteItem(itemID, organizationID string) error {
	query := fmt.Sprintf("UPDATE inventory_items SET status = 0 WHERE item_id = %s AND organization_id = %s",
		r.getPlaceholder(1), r.getPlaceholder(2))

	result, err := database.Exec(r.db, query, itemID, organizationID)
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

func (r *InventoryRepository) GetRequestList(organizationID string) ([]model.InventoryRequestWithLabel, error) {
	query := `
		SELECT ir.request_id, ir.request_number, ir.item_id, ir.garage_id, ir.quantity, ir.status, ir.organization_id,
		       ir.created_at, ir.created_by, ir.approve_at, ir.approve_by, ir.updated_at, ir.updated_by,
		       i.item_name, g.garage_name, g.garage_city
		FROM inventory_request ir
		LEFT JOIN inventory_items i ON ir.item_id = i.item_id
		LEFT JOIN garage g ON ir.garage_id = g.garage_id
		WHERE ir.organization_id = %s AND ir.status > 0
		ORDER BY ir.created_at DESC
	`
	query = fmt.Sprintf(query, r.getPlaceholder(1))

	rows, err := database.Query(r.db, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	r.ensureLocationsLoaded()
	var items []model.InventoryRequestWithLabel
	for rows.Next() {
		var item model.InventoryRequestWithLabel
		var garageCity string
		if err := rows.Scan(
			&item.RequestID,
			&item.RequestNumber,
			&item.ItemID,
			&item.GarageID,
			&item.Quantity,
			&item.Status,
			&item.OrganizationID,
			&item.CreatedAt,
			&item.CreatedBy,
			&item.ApproveAt,
			&item.ApproveBy,
			&item.UpdatedAt,
			&item.UpdatedBy,
			&item.ItemName,
			&item.GarageName,
			&garageCity,
		); err != nil {
			return nil, err
		}
		item.GarageCity = garageCity
		if name, ok := r.cityMap[garageCity]; ok {
			item.GarageCityLabel = name
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *InventoryRepository) GetRequestByID(requestID, organizationID string) (*model.InventoryRequestWithLabel, error) {
	query := `
		SELECT ir.request_id, ir.request_number, ir.item_id, ir.garage_id, ir.quantity, ir.status, ir.organization_id,
		       ir.created_at, ir.created_by, ir.approve_at, ir.approve_by, ir.updated_at, ir.updated_by,
		       i.item_name, g.garage_name, g.garage_city
		FROM inventory_request ir
		LEFT JOIN inventory_items i ON ir.item_id = i.item_id
		LEFT JOIN garage g ON ir.garage_id = g.garage_id
		WHERE ir.request_id = %s AND ir.organization_id = %s
	`
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))

	var item model.InventoryRequestWithLabel
	var garageCity string
	err := database.QueryRow(r.db, query, requestID, organizationID).Scan(
		&item.RequestID,
		&item.RequestNumber,
		&item.ItemID,
		&item.GarageID,
		&item.Quantity,
		&item.Status,
		&item.OrganizationID,
		&item.CreatedAt,
		&item.CreatedBy,
		&item.ApproveAt,
		&item.ApproveBy,
		&item.UpdatedAt,
		&item.UpdatedBy,
		&item.ItemName,
		&item.GarageName,
		&garageCity,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	item.GarageCity = garageCity
	r.ensureLocationsLoaded()
	if name, ok := r.cityMap[garageCity]; ok {
		item.GarageCityLabel = name
	}

	return &item, nil
}

func (r *InventoryRepository) CreateRequest(request *model.InventoryRequest) error {
	request.RequestID = uuid.New().String()
	now := time.Now()
	request.CreatedAt = now
	request.UpdatedAt = now

	query := fmt.Sprintf(`
		INSERT INTO inventory_request (request_id, request_number, item_id, garage_id, quantity, status, organization_id, created_at, created_by, updated_at, updated_by)
		VALUES (%s, %s, %s, %s, %s, 2, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10),
	)

	_, err := database.Exec(r.db, query,
		request.RequestID,
		request.RequestNumber,
		request.ItemID,
		request.GarageID,
		request.Quantity,
		request.OrganizationID,
		request.CreatedAt,
		request.CreatedBy,
		request.UpdatedAt,
		request.UpdatedBy,
	)
	return err
}

func (r *InventoryRepository) UpdateRequestStatus(requestID, organizationID, updatedBy string, status int) error {
	query := fmt.Sprintf(`
		UPDATE inventory_request SET status = %s, updated_at = %s, updated_by = %s WHERE request_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5))

	now := time.Now()
	_, err := database.Exec(r.db, query, status, now, updatedBy, requestID, organizationID)
	return err
}

func (r *InventoryRepository) UpdateRequestApprove(requestID, organizationID, updatedBy string, approveAt time.Time) error {
	query := fmt.Sprintf(`
		UPDATE inventory_request SET status = 2, approve_at = %s, approve_by = %s, updated_at = %s, updated_by = %s WHERE request_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))

	now := time.Now()
	_, err := database.Exec(r.db, query, approveAt, updatedBy, now, updatedBy, requestID, organizationID)
	return err
}

func (r *InventoryRepository) GetOrdersList(organizationID string) ([]model.InventoryOrderWithDetail, error) {
	query := `
		SELECT io.purchase_id, io.request_id, io.suplier_id, io.quantity, io.amount, io.total_amount, io.status,
		       io.organization_id, io.created_at, io.created_by, io.updated_at, io.updated_by,
		       s.suplier_name, s.suplier_city, ir.item_id, ir.garage_id, i.item_name, g.garage_name, g.garage_city
		FROM inventory_orders io
		LEFT JOIN supliers s ON io.suplier_id = s.suplier_id
		LEFT JOIN inventory_request ir ON io.request_id = ir.request_id
		LEFT JOIN inventory_items i ON ir.item_id = i.item_id
		LEFT JOIN garage g ON ir.garage_id = g.garage_id
		WHERE io.organization_id = %s AND io.status > 0
		ORDER BY io.created_at DESC
	`
	query = fmt.Sprintf(query, r.getPlaceholder(1))

	rows, err := database.Query(r.db, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	r.ensureLocationsLoaded()
	var items []model.InventoryOrderWithDetail
	for rows.Next() {
		var item model.InventoryOrderWithDetail
		var supplierCity int
		var garageCity string
		if err := rows.Scan(
			&item.PurchaseID,
			&item.RequestID,
			&item.SupplierID,
			&item.Quantity,
			&item.Amount,
			&item.TotalAmount,
			&item.Status,
			&item.OrganizationID,
			&item.CreatedAt,
			&item.CreatedBy,
			&item.UpdatedAt,
			&item.UpdatedBy,
			&item.SupplierName,
			&supplierCity,
			&item.ItemID,
			&item.GarageID,
			&item.ItemName,
			&item.GarageName,
			&garageCity,
		); err != nil {
			return nil, err
		}
		item.SupplierCity = supplierCity
		if name, ok := r.cityMap[fmt.Sprintf("%d", supplierCity)]; ok {
			item.SupplierCityLabel = name
		}
		item.GarageCity = garageCity
		if name, ok := r.cityMap[garageCity]; ok {
			item.GarageCityLabel = name
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *InventoryRepository) GetOrderByPurchaseID(purchaseID, organizationID string) (*model.InventoryOrderWithDetail, error) {
	query := `
		SELECT io.purchase_id, io.request_id, io.suplier_id, io.quantity, io.amount, io.total_amount, io.status,
		       io.organization_id, io.created_at, io.created_by, io.updated_at, io.updated_by,
		       s.suplier_name, s.suplier_city, ir.item_id, ir.garage_id, i.item_name, g.garage_name, g.garage_city
		FROM inventory_orders io
		LEFT JOIN supliers s ON io.suplier_id = s.suplier_id
		LEFT JOIN inventory_request ir ON io.request_id = ir.request_id
		LEFT JOIN inventory_items i ON ir.item_id = i.item_id
		LEFT JOIN garage g ON ir.garage_id = g.garage_id
		WHERE io.purchase_id = %s AND io.organization_id = %s
	`
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))

	var item model.InventoryOrderWithDetail
	var supplierCity int
	var garageCity string
	err := database.QueryRow(r.db, query, purchaseID, organizationID).Scan(
		&item.PurchaseID,
		&item.RequestID,
		&item.SupplierID,
		&item.Quantity,
		&item.Amount,
		&item.TotalAmount,
		&item.Status,
		&item.OrganizationID,
		&item.CreatedAt,
		&item.CreatedBy,
		&item.UpdatedAt,
		&item.UpdatedBy,
		&item.SupplierName,
		&supplierCity,
		&item.ItemID,
		&item.GarageID,
		&item.ItemName,
		&item.GarageName,
		&garageCity,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	item.SupplierCity = supplierCity
	r.ensureLocationsLoaded()
	if name, ok := r.cityMap[fmt.Sprintf("%d", supplierCity)]; ok {
		item.SupplierCityLabel = name
	}
	item.GarageCity = garageCity
	if name, ok := r.cityMap[garageCity]; ok {
		item.GarageCityLabel = name
	}

	return &item, nil
}

func (r *InventoryRepository) CreateOrder(order *model.InventoryOrder) error {
	order.PurchaseID = uuid.New().String()
	now := time.Now()
	order.CreatedAt = now
	order.UpdatedAt = now

	query := fmt.Sprintf(`
		INSERT INTO inventory_orders (purchase_id, request_id, suplier_id, quantity, amount, total_amount, status, organization_id, created_at, created_by, updated_at, updated_by)
		VALUES (%s, %s, %s, %s, %s, %s, 2, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11),
	)

	_, err := database.Exec(r.db, query,
		order.PurchaseID,
		order.RequestID,
		order.SupplierID,
		order.Quantity,
		order.Amount,
		order.TotalAmount,
		order.OrganizationID,
		order.CreatedAt,
		order.CreatedBy,
		order.UpdatedAt,
		order.UpdatedBy,
	)
	return err
}

func (r *InventoryRepository) UpdateOrderSupplier(purchaseID, organizationID, supplierID, updatedBy string) error {
	query := fmt.Sprintf(`
		UPDATE inventory_orders SET suplier_id = %s, updated_at = %s, updated_by = %s WHERE purchase_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5))

	now := time.Now()
	_, err := database.Exec(r.db, query, supplierID, now, updatedBy, purchaseID, organizationID)
	return err
}

func (r *InventoryRepository) GetSuppliers(organizationID string) ([]model.Supplier, error) {
	query := `
		SELECT suplier_id, suplier_name, suplier_address, suplier_city, suplier_phone, supliter_email,
		       created_at, created_by, updated_at, updated_by
		FROM supliers
		ORDER BY created_at DESC
	`

	rows, err := database.Query(r.db, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	r.ensureLocationsLoaded()
	var items []model.Supplier
	for rows.Next() {
		var item model.Supplier
		var city int
		if err := rows.Scan(
			&item.SupplierID,
			&item.SupplierName,
			&item.SupplierAddress,
			&city,
			&item.SupplierPhone,
			&item.SupplierEmail,
			&item.CreatedAt,
			&item.CreatedBy,
			&item.UpdatedAt,
			&item.UpdatedBy,
		); err != nil {
			return nil, err
		}
		if name, ok := r.cityMap[fmt.Sprintf("%d", city)]; ok {
			item.SupplierCityLabel = name
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *InventoryRepository) GetSupplierByID(supplierID string) (*model.Supplier, error) {
	query := fmt.Sprintf(`
		SELECT suplier_id, suplier_name, suplier_address, suplier_city, suplier_phone, supliter_email,
		       created_at, created_by, updated_at, updated_by
		FROM supliers
		WHERE suplier_id = %s
	`, r.getPlaceholder(1))

	var item model.Supplier
	var city int
	err := database.QueryRow(r.db, query, supplierID).Scan(
		&item.SupplierID,
		&item.SupplierName,
		&item.SupplierAddress,
		&city,
		&item.SupplierPhone,
		&item.SupplierEmail,
		&item.CreatedAt,
		&item.CreatedBy,
		&item.UpdatedAt,
		&item.UpdatedBy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	r.ensureLocationsLoaded()
	if name, ok := r.cityMap[fmt.Sprintf("%d", city)]; ok {
		item.SupplierCityLabel = name
	}

	return &item, nil
}

func (r *InventoryRepository) CreateSupplier(supplier *model.Supplier) error {
	supplier.SupplierID = uuid.New().String()
	now := time.Now()
	supplier.CreatedAt = now
	supplier.UpdatedAt = now

	query := fmt.Sprintf(`
		INSERT INTO supliers (suplier_id, suplier_name, suplier_address, suplier_city, suplier_phone, supliter_email, created_at, created_by, updated_at, updated_by)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		r.getPlaceholder(9), r.getPlaceholder(10),
	)

	_, err := database.Exec(r.db, query,
		supplier.SupplierID,
		supplier.SupplierName,
		supplier.SupplierAddress,
		supplier.SupplierCity,
		supplier.SupplierPhone,
		supplier.SupplierEmail,
		supplier.CreatedAt,
		supplier.CreatedBy,
		supplier.UpdatedAt,
		supplier.UpdatedBy,
	)
	return err
}

func (r *InventoryRepository) DeleteSupplier(supplierID string) error {
	query := fmt.Sprintf("DELETE FROM supliers WHERE suplier_id = %s",
		r.getPlaceholder(1))

	_, err := database.Exec(r.db, query, supplierID)
	return err
}

func (r *InventoryRepository) GetItemGarageStock(itemID, garageID, organizationID string) (int, error) {
	query := fmt.Sprintf(`
		SELECT stock FROM inventory_item_garage 
		WHERE item_id = %s AND garage_id = %s AND organization_id = %s
		LIMIT 1
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	var stock int
	err := database.QueryRow(r.db, query, itemID, garageID, organizationID).Scan(&stock)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, sql.ErrNoRows
		}
		return 0, err
	}
	return stock, nil
}

func (r *InventoryRepository) UpsertItemGarage(itemID, garageID, organizationID, createdBy string, stock int) error {
	_, err := r.GetItemGarageStock(itemID, garageID, organizationID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	now := time.Now()

	if err == sql.ErrNoRows {
		query := fmt.Sprintf(`
			INSERT INTO inventory_item_garage (item_garage_id, item_id, garage_id, stock, organization_id, created_at, created_by, updated_at, updated_by)
			VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
		`,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9),
		)
		_, err = database.Exec(r.db, query,
			uuid.New().String(),
			itemID,
			garageID,
			stock,
			organizationID,
			now,
			createdBy,
			now,
			createdBy,
		)
		return err
	}

	query := fmt.Sprintf(`
		UPDATE inventory_item_garage SET stock = %s, updated_at = %s, updated_by = %s 
		WHERE item_id = %s AND garage_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))

	_, err = database.Exec(r.db, query, stock, now, createdBy, itemID, garageID, organizationID)
	return err
}

func (r *InventoryRepository) CreateInventoryMovement(organizationID string, movement *model.InventoryMovement) error {
	query := fmt.Sprintf(`
		INSERT INTO inventory_movement (movement_id, item_id, garage_id, quantity, stock_before, stock_final, movement_type, notes, organization_id, created_at, created_by)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11),
	)

	_, err := database.Exec(r.db, query,
		movement.MovementID,
		movement.ItemID,
		movement.GarageID,
		movement.Quantity,
		movement.StockBefore,
		movement.StockFinal,
		movement.MovementType,
		movement.Notes,
		organizationID,
		movement.CreatedAt,
		movement.CreatedBy,
	)
	return err
}

func (r *InventoryRepository) GetItemIDByName(itemName, organizationID string) (string, error) {
	query := fmt.Sprintf(`
		SELECT item_id FROM inventory_items 
		WHERE item_name = %s AND organization_id = %s AND status > 0
		LIMIT 1
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	var itemID string
	err := database.QueryRow(r.db, query, itemName, organizationID).Scan(&itemID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", sql.ErrNoRows
		}
		return "", err
	}
	return itemID, nil
}

func (r *InventoryRepository) GetSupplierIDByName(supplierName string) (string, error) {
	query := fmt.Sprintf(`
		SELECT suplier_id FROM supliers 
		WHERE suplier_name = %s
		LIMIT 1
	`, r.getPlaceholder(1))

	var supplierID string
	err := database.QueryRow(r.db, query, supplierName).Scan(&supplierID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", sql.ErrNoRows
		}
		return "", err
	}
	return supplierID, nil
}

func (r *InventoryRepository) GetItemLocations(itemID string) ([]model.InventoryItemLocation, error) {
	query := fmt.Sprintf(`
		SELECT g.garage_name, g.garage_address, g.garage_city, ig.stock, ig.updated_at
		FROM inventory_item_garage ig
		INNER JOIN garage g ON g.garage_id = ig.garage_id
		WHERE ig.item_id = %s
	`, r.getPlaceholder(1))

	rows, err := database.Query(r.db, query, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	r.ensureLocationsLoaded()
	var locations []model.InventoryItemLocation
	for rows.Next() {
		var loc model.InventoryItemLocation
		var garageCity string
		var updatedAt sql.NullTime
		if err := rows.Scan(
			&loc.GarageName,
			&loc.GarageAddress,
			&garageCity,
			&loc.Stock,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		if name, ok := r.cityMap[garageCity]; ok {
			loc.GarageCityLabel = name
		}
		loc.GarageCity = garageCity
		if updatedAt.Valid {
			loc.UpdatedAt = updatedAt.Time
		}
		locations = append(locations, loc)
	}

	return locations, nil
}

func (r *InventoryRepository) GetItemDetail(itemID string) (*model.InventoryItemDetail, error) {
	query := fmt.Sprintf(`
		SELECT item_id, item_sku, item_name, item_uom, item_category, status
		FROM inventory_items
		WHERE item_id = %s AND status > 0
		LIMIT 1
	`, r.getPlaceholder(1))

	var item model.InventoryItemDetail
	err := database.QueryRow(r.db, query, itemID).Scan(
		&item.ItemID,
		&item.ItemSKU,
		&item.ItemName,
		&item.ItemUOM,
		&item.ItemCategory,
		&item.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	locations, err := r.GetItemLocations(itemID)
	if err != nil {
		return nil, err
	}
	item.Locations = locations

	return &item, nil
}

func (r *InventoryRepository) GetItemMovements(organizationID, itemID, startDate, endDate string) ([]model.InventoryItemMovement, error) {
	query := `
		SELECT im.movement_id, g.garage_name, mt.label as movement_type, im.quantity, im.stock_before, im.stock_final, i.item_uom, im.notes, im.created_at as movement_date
		FROM inventory_movement im
		INNER JOIN garage g ON im.garage_id = g.garage_id
		INNER JOIN inventory_movement_types mt ON mt.id = im.movement_type
		INNER JOIN inventory_items i ON i.item_id = im.item_id
		WHERE im.organization_id = $1 AND im.item_id = $2
	`

	var args []interface{}
	args = append(args, organizationID, itemID)
	pos := 3

	if startDate != "" {
		query += fmt.Sprintf(" AND DATE(im.created_at) >= %s", r.getPlaceholder(pos))
		args = append(args, startDate)
		pos++
	}
	if endDate != "" {
		query += fmt.Sprintf(" AND DATE(im.created_at) <= %s", r.getPlaceholder(pos))
		args = append(args, endDate)
		pos++
	}
	query += " ORDER BY im.created_at DESC"
	fmt.Println("GetItemMovements query: ", query, organizationID, itemID, startDate, endDate)

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movements []model.InventoryItemMovement
	for rows.Next() {
		var m model.InventoryItemMovement
		var notes sql.NullString
		if err := rows.Scan(
			&m.MovementID,
			&m.GarageName,
			&m.MovementType,
			&m.Quantity,
			&m.StockBefore,
			&m.StockFinal,
			&m.ItemUom,
			&notes,
			&m.MovementDate,
		); err != nil {
			return nil, err
		}
		if notes.Valid {
			m.Notes = notes.String
		} else {
			m.Notes = ""
		}
		if m.StockBefore < m.StockFinal {
			m.Label = "+"
		} else if m.StockBefore > m.StockFinal {
			m.Label = "-"
		} else {
			m.Label = ""
		}
		movements = append(movements, m)
	}

	return movements, nil
}
