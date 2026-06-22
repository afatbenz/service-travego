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

func (r *InventoryRepository) GenerateItemSKU(organizationID string) (string, error) {
	return utils.GenerateItemSKU(r.db, r.driver, organizationID)
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

func (r *InventoryRepository) GetAllItems(organizationID string, itemCategory int) ([]model.InventoryItemWithLabel, error) {
	var query string
	var args []interface{}
	args = append(args, organizationID)
	pos := 2

	if r.driver == "mysql" {
		query = fmt.Sprintf(`
			SELECT i.item_id, i.item_sku, i.item_name, i.item_uom, i.item_category, i.status, SUM(ig.stock) AS total_stock, GROUP_CONCAT(g.garage_name SEPARATOR ', ') AS garage_names, MAX(ig.created_at) AS created_at, MAX(ig.updated_at) AS updated_at
			FROM inventory_items i
			INNER JOIN inventory_item_garage ig ON ig.item_id = i.item_id
			INNER JOIN garage g ON g.garage_id = ig.garage_id
			WHERE i.organization_id = %s AND i.status = 1
		`, r.getPlaceholder(1))
	} else {
		query = fmt.Sprintf(`
			SELECT i.item_id, i.item_sku, i.item_name, i.item_uom, i.item_category, i.status, SUM(ig.stock) AS total_stock, STRING_AGG(g.garage_name, ', ') AS garage_names, MAX(ig.created_at) AS created_at, MAX(ig.updated_at) AS updated_at
			FROM inventory_items i
			INNER JOIN inventory_item_garage ig ON ig.item_id = i.item_id
			INNER JOIN garage g ON g.garage_id = ig.garage_id
			WHERE i.organization_id = %s AND i.status = 1
		`, r.getPlaceholder(1))
	}

	if itemCategory > 0 {
		query += fmt.Sprintf(" AND i.item_category = %s", r.getPlaceholder(pos))
		args = append(args, itemCategory)
		pos++
	}

	query += " GROUP BY i.item_id, i.item_sku, i.item_name, i.item_uom, i.item_category, i.status"

	rows, err := database.Query(r.db, query, args...)
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
	if item.ItemSKU == "" {
		item.ItemSKU = sku
	}

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

func (r *InventoryRepository) CreateItemWithoutStock(item *model.InventoryItem) error {
	item.ItemID = uuid.New().String()
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	item.Stock = 0

	sku, err := utils.GenerateItemSKU(r.db, r.driver, item.OrganizationID)
	if err != nil {
		return err
	}
	if item.ItemSKU == "" {
		item.ItemSKU = sku
	}

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
		0,
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
		SELECT ir.request_id, ir.item_category, i.item_name, i.item_uom, i.item_sku, 
		g.garage_name, ir.quantity, ir.status, e.fullname as employee_name
		FROM inventory_request ir INNER JOIN inventory_items i ON i.item_id = ir.item_id
		INNER JOIN garage g ON g.garage_id = ir.garage_id
		INNER JOIN employee e ON e.uuid = ir.employee_id
		WHERE ir.organization_id = %s 
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
			&item.ItemCategory,
			&item.ItemName,
			&item.ItemUOM,
			&item.ItemSKU,
			&item.GarageName,
			&item.Quantity,
			&item.Status,
			&item.EmployeeName,
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
		SELECT ir.request_id, ir.item_id, ir.garage_id, ir.organization_id, COALESCE(NULLIF(i.item_name, ''), ir.item_name) AS item_name, ir.item_category, i.item_sku, ir.quantity, ir.item_uom,
			   g.garage_name, g.garage_city, ir.status as request_status,
			   fu.unit_id, fu.vehicle_id, fu.plate_number,
			   e.fullname as requested_by, io.purchase_id, io.transaction_date, io.status as order_status,
			   ir.created_at, uc.fullname as created_by, ir.approve_at, ir.approve_by, ir.updated_at, uu.fullname as updated_by,
			   ir.received_at, er.fullname as received_by
		FROM inventory_request ir
		INNER JOIN employee e ON e.uuid = ir.employee_id
		INNER JOIN users uc ON uc.user_id = ir.created_by
		LEFT JOIN users uu ON uu.user_id = ir.updated_by
		LEFT JOIN employee er ON er.uuid = ir.received_by
		LEFT JOIN inventory_items i ON ir.item_id = i.item_id
		LEFT JOIN garage g ON ir.garage_id = g.garage_id
		LEFT JOIN inventory_orders io ON io.request_id = ir.request_id
		LEFT JOIN inventory_request_fleets if ON if.request_id = ir.request_id
		LEFT JOIN fleet_units fu ON fu.unit_id = if.unit_id
		WHERE ir.request_id = %s AND ir.organization_id = %s
	`
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))

	var item model.InventoryRequestWithLabel
	var garageCity string
	var itemSKU sql.NullString
	var unitID sql.NullString
	var vehicleID sql.NullString
	var plateNumber sql.NullString
	var createdByName sql.NullString
	var approveByName sql.NullString
	var updatedByName sql.NullString
	var purchaseID sql.NullString
	var transactionDate sql.NullString
	var orderStatus sql.NullInt64
	var receivedByName sql.NullString
	var approveAt sql.NullTime
	var receivedAt sql.NullTime

	err := database.QueryRow(r.db, query, requestID, organizationID).Scan(
		&item.RequestID,
		&item.ItemID,
		&item.GarageID,
		&item.OrganizationID,
		&item.ItemName,
		&item.ItemCategory,
		&itemSKU,
		&item.Quantity,
		&item.ItemUOM,
		&item.GarageName,
		&garageCity,
		&item.Status,
		&unitID,
		&vehicleID,
		&plateNumber,
		&item.EmployeeName,
		&purchaseID,
		&transactionDate,
		&orderStatus,
		&item.CreatedAt,
		&createdByName,
		&approveAt,
		&approveByName,
		&item.UpdatedAt,
		&updatedByName,
		&receivedAt,
		&receivedByName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	if itemSKU.Valid {
		item.ItemSKU = itemSKU.String
	}
	if unitID.Valid {
		item.UnitID = unitID.String
	}
	if vehicleID.Valid {
		item.VehicleID = vehicleID.String
	}
	if plateNumber.Valid {
		item.PlateNumber = plateNumber.String
	}
	if createdByName.Valid {
		item.CreatedBy = createdByName.String
	}
	if approveByName.Valid {
		item.ApproveBy = approveByName.String
	}
	if updatedByName.Valid {
		item.UpdatedBy = updatedByName.String
	}
	if purchaseID.Valid {
		item.PurchaseID = purchaseID.String
	}
	if transactionDate.Valid {
		item.TransactionDate = transactionDate.String
	}
	if orderStatus.Valid {
		item.OrderStatus = int(orderStatus.Int64)
	}
	if receivedByName.Valid {
		item.ReceivedBy = receivedByName.String
	}
	if approveAt.Valid {
		item.ApproveAt = approveAt.Time
	}
	if receivedAt.Valid {
		item.ReceivedAt = receivedAt.Time
	}

	r.ensureLocationsLoaded()
	item.GarageCity = garageCity
	if name, ok := r.cityMap[garageCity]; ok {
		item.GarageCityLabel = name
	}

	switch item.Status {
	case 1:
		item.RequestStatusLabel = "Selesai / Diterima"
	case 2:
		item.RequestStatusLabel = "Menunggu Persetujuan"
	case 3:
		item.RequestStatusLabel = "Pesanan Diproses"
	}

	switch item.OrderStatus {
	case 1:
		item.OrderStatusLabel = "Selesai / Diterima"
	case 2:
		item.OrderStatusLabel = "Sedang Diproses"
	case 0:
		item.OrderStatusLabel = "Dibatalkan"
	}

	switch item.ItemCategory {
	case 1:
		item.ItemCategoryLabel = "Asset Armada"
	case 2:
		item.ItemCategoryLabel = "Asset Umum"
	}

	return &item, nil
}

func (r *InventoryRepository) GetCurrentGarageItemStock(itemID, garageID, organizationID string) (int, error) {
	query := fmt.Sprintf(`
		SELECT stock FROM inventory_item_garage WHERE item_id = %s AND garage_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	var stock int
	if err := database.QueryRow(r.db, query, itemID, garageID, organizationID).Scan(&stock); err != nil {
		return 0, err
	}
	return stock, nil
}

func (r *InventoryRepository) CreateRequest(request *model.InventoryRequest) error {
	now := time.Now()
	request.CreatedAt = now
	request.UpdatedAt = now

	columns := "request_id, item_id, item_name, item_uom, item_category, garage_id, quantity, status, organization_id, employee_id, notes, created_at, created_by, updated_at, updated_by"
	values := fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, 2, %s, %s, %s, %s, %s, %s, %s)",
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12),
		r.getPlaceholder(13), r.getPlaceholder(14),
	)
	query := fmt.Sprintf("INSERT INTO inventory_request (%s) VALUES %s", columns, values)
	fmt.Println("CreateRequest query:", query)                      // Debugging line
	fmt.Println("CreateRequest request id:", request.RequestNumber) // Debugging line
	fmt.Println("CreateRequest ItemID, ", request.ItemID)
	fmt.Println("CreateRequest GarageID, ", request.GarageID)
	fmt.Println("CreateRequest GarageID, ", request.GarageID)

	_, err := database.Exec(r.db, query,
		request.RequestNumber,
		request.ItemID,
		request.ItemName,
		request.ItemUOM,
		request.ItemCategory,
		request.GarageID,
		request.Quantity,
		request.OrganizationID,
		request.EmployeeID,
		request.Notes,
		request.CreatedAt,
		request.CreatedBy,
		request.UpdatedAt,
		request.UpdatedBy,
	)

	return err
}

func (r *InventoryRepository) CreateRequestFleet(requestNumber, unitID, createdBy string) error {
	query := fmt.Sprintf(`
		INSERT INTO inventory_request_fleets (request_id, unit_id, created_at, created_by)
		VALUES (%s, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))

	_, err := database.Exec(r.db, query, requestNumber, unitID, time.Now(), createdBy)
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

func (r *InventoryRepository) GetEmployeePhoneByUUID(employeeUUID string) (string, error) {
	query := fmt.Sprintf(`
		SELECT phone FROM employee WHERE uuid = %s LIMIT 1
	`, r.getPlaceholder(1))

	var phone string
	if err := database.QueryRow(r.db, query, employeeUUID).Scan(&phone); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return phone, nil
}

func (r *InventoryRepository) GetRequestForApprove(requestID, organizationID string) (*model.InventoryRequest, error) {
	query := fmt.Sprintf(`
		SELECT ir.request_id, ir.item_id, ir.item_name, ir.garage_id, ir.quantity, ir.employee_id, ir.organization_id
		FROM inventory_request ir
		WHERE ir.request_id = %s AND ir.organization_id = %s
		LIMIT 1
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	var req model.InventoryRequest
	err := database.QueryRow(r.db, query, requestID, organizationID).Scan(
		&req.RequestID,
		&req.ItemID,
		&req.ItemName,
		&req.GarageID,
		&req.Quantity,
		&req.EmployeeID,
		&req.OrganizationID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &req, nil
}

func (r *InventoryRepository) ApproveInventoryRequest(requestID, organizationID, itemID, employeeID, updatedBy string) error {
	req, err := r.GetRequestForApprove(requestID, organizationID)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()

	now := time.Now()
	quantity := req.Quantity
	fmt.Println("check itemid here ... ", req.ItemID)
	fmt.Println("check garageid here ... ", req.GarageID)

	if req.ItemID != "" {
		currentStock, stockErr := r.GetItemGarageStock(req.ItemID, req.GarageID, organizationID)
		fmt.Println("check currentStock here ... ", currentStock)
		fmt.Println("check quantity here ... ", quantity)

		if stockErr != nil && stockErr != sql.ErrNoRows {
			fmt.Println("error get item garage stock ", stockErr)
			return stockErr
		}
		if stockErr == sql.ErrNoRows {
			currentStock = 0
		}
		newStock := currentStock - quantity
		if newStock < 0 {
			newStock = 0
		}
		fmt.Println("check newStock here ... ", newStock)
		stockQuery := fmt.Sprintf(`
			UPDATE inventory_item_garage SET stock = %s, updated_at = %s, updated_by = %s
			WHERE item_id = %s AND garage_id = %s AND organization_id = %s
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
		if _, err := database.TxExec(tx, stockQuery, newStock, now, updatedBy, req.ItemID, req.GarageID, organizationID); err != nil {
			fmt.Println("error insert item_garage ", err)
			return err
		}
		fmt.Println("check movement here ... ")
		movementQuery := fmt.Sprintf(`
			INSERT INTO inventory_movement (movement_id, item_id, garage_id, quantity, stock_before, stock_final, movement_type, notes, organization_id, created_at, created_by)
			VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11))
		if _, err := database.TxExec(tx, movementQuery, uuid.New().String(), req.ItemID, req.GarageID, quantity, currentStock, newStock, 3, "Request approved", organizationID, now, updatedBy); err != nil {
			fmt.Println("error insert movement ", err)
			return err
		}
		fmt.Println("check updateQuery here ... ")
		updateQuery := fmt.Sprintf(`
			UPDATE inventory_request SET item_id = %s, status = 1, received_at = %s, received_by = %s, updated_at = %s, updated_by = %s
			WHERE request_id = %s AND organization_id = %s
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7))
		_, err = database.TxExec(tx, updateQuery, req.ItemID, now, employeeID, now, updatedBy, requestID, organizationID)
		if err != nil {
			fmt.Println("error update inventory_request ", err)
			return err
		}
	} else {
		fmt.Println("check updateQuery here ... ")
		updateQuery := fmt.Sprintf(`
			UPDATE inventory_request SET status = 1, received_at = %s, received_by = %s, updated_at = %s, updated_by = %s
			WHERE request_id = %s AND organization_id = %s
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
		_, err = database.TxExec(tx, updateQuery, now, employeeID, now, updatedBy, requestID, organizationID)
	}

	if err != nil {
		return err
	}

	rollback = false
	return tx.Commit()
}

func (r *InventoryRepository) RejectInventoryRequest(requestID, organizationID, updatedBy string) error {
	query := fmt.Sprintf(`
		UPDATE inventory_request SET status = 0, updated_at = %s, updated_by = %s
		WHERE request_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))

	now := time.Now()
	_, err := database.Exec(r.db, query, now, updatedBy, requestID, organizationID)
	return err
}

func (r *InventoryRepository) GetOrdersList(organizationID string) ([]model.InventoryOrderWithDetail, error) {
	query := `
		SELECT io.purchase_id, io.request_id, io.suplier_id, io.quantity, io.item_price, io.total_amount, io.status,
		       io.organization_id, io.created_at,
		       s.suplier_name, s.suplier_city, io.item_id, i.item_uom, io.garage_id, i.item_name, i.item_sku, g.garage_name, g.garage_city, i.item_category, io.transaction_date
		FROM inventory_orders io
		LEFT JOIN supliers s ON io.suplier_id = s.suplier_id
		LEFT JOIN inventory_request ir ON io.request_id = ir.request_id
		LEFT JOIN inventory_items i ON io.item_id = i.item_id
		LEFT JOIN garage g ON io.garage_id = g.garage_id
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
			&item.SupplierName,
			&supplierCity,
			&item.ItemID,
			&item.ItemUOM,
			&item.GarageID,
			&item.ItemName,
			&item.ItemSKU,
			&item.GarageName,
			&garageCity,
			&item.ItemCategory,
			&item.TransactionDate,
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
		switch item.ItemCategory {
		case 1:
			item.ItemCategoryLabel = "Asset Armada"
		case 2:
			item.ItemCategoryLabel = "Asset Umum"
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *InventoryRepository) GetOrderByPurchaseID(purchaseID, organizationID string) (*model.InventoryOrderWithDetail, error) {
	query := `
		SELECT io.purchase_id, io.request_id, io.quantity, io.item_price, io.total_amount, io.status,
		       s.suplier_name, s.suplier_city,
			   io.item_id, i.item_name, i.item_sku, i.item_uom, i.item_category, io.transaction_date, 
			   g.garage_name, g.garage_city, io.created_at, u.fullname as create_by, u2.fullname as updated_by, io.updated_at, t.invoice_number
		FROM inventory_orders io
		LEFT JOIN supliers s ON io.suplier_id = s.suplier_id
		LEFT JOIN inventory_request ir ON io.request_id = ir.request_id
		LEFT JOIN inventory_items i ON io.item_id = i.item_id
		LEFT JOIN garage g ON io.garage_id = g.garage_id
		INNER JOIN users u ON u.user_id = io.created_by
		INNER JOIN users u2 ON u.user_id = io.updated_by
		INNER JOIN transactions t ON t.reference_id = io.purchase_id
		WHERE io.purchase_id = %s AND io.organization_id = %s
	`
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))

	var item model.InventoryOrderWithDetail
	var supplierCity int
	var garageCity string
	var itemCategory int
	var itemSKU sql.NullString
	var updateBy sql.NullString
	var updatedAt sql.NullTime
	err := database.QueryRow(r.db, query, purchaseID, organizationID).Scan(
		&item.PurchaseID,
		&item.RequestID,
		&item.Quantity,
		&item.Amount,
		&item.TotalAmount,
		&item.Status,
		&item.SupplierName,
		&supplierCity,
		&item.ItemID,
		&item.ItemName,
		&itemSKU,
		&item.ItemUOM,
		&itemCategory,
		&item.TransactionDate,
		&item.GarageName,
		&garageCity,
		&item.CreatedAt,
		&item.CreatedBy,
		&updateBy,
		&updatedAt,
		&item.InvoiceNumber,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	if itemSKU.Valid {
		item.ItemSKU = itemSKU.String
	}
	if updateBy.Valid {
		item.UpdatedBy = updateBy.String
	}
	if updatedAt.Valid {
		item.UpdatedAt = updatedAt.Time
	}
	item.ItemCategory = itemCategory
	switch item.ItemCategory {
	case 1:
		item.ItemCategoryLabel = "Asset Armada"
	case 2:
		item.ItemCategoryLabel = "Asset Umum"
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
		INSERT INTO inventory_orders (purchase_id, request_id, suplier_id, quantity, item_price, total_amount, status, organization_id, created_at, created_by, updated_at, updated_by, transaction_date)
		VALUES (%s, %s, %s, %s, %s, %s, 2, %s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12),
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
		order.TransactionDate,
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

func (r *InventoryRepository) GetItemGarageStockWithGarageName(itemID, garageID, organizationID string) (model.InventoryGarageStock, error) {
	query := fmt.Sprintf(`
		SELECT ig.stock, g.garage_name
		FROM inventory_item_garage ig
		INNER JOIN garage g ON ig.garage_id = g.garage_id
		WHERE ig.item_id = %s AND ig.organization_id = %s AND ig.garage_id = %s
		LIMIT 1
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	var stock model.InventoryGarageStock
	err := database.QueryRow(r.db, query, itemID, organizationID, garageID).Scan(&stock.Stock, &stock.GarageName)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.InventoryGarageStock{}, sql.ErrNoRows
		}
		return model.InventoryGarageStock{}, err
	}
	return stock, nil
}

func (r *InventoryRepository) UpdateItemGarageStock(itemID, garageID, organizationID, updatedBy string, stock int) error {
	now := time.Now()
	query := fmt.Sprintf(`
		UPDATE inventory_item_garage SET stock = %s, updated_at = %s, updated_by = %s
		WHERE garage_id = %s AND item_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))

	result, err := database.Exec(r.db, query, stock, now, updatedBy, garageID, itemID, organizationID)
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

func (r *InventoryRepository) TransferItemStock(organizationID, updatedBy string, req *model.TransferInventoryItemRequest, currentStockFrom, currentStockDest model.InventoryGarageStock) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()

	now := time.Now()
	newStockFrom := currentStockFrom.Stock - req.Stock
	newStockDest := currentStockDest.Stock + req.Stock

	query := fmt.Sprintf(`
		UPDATE inventory_item_garage SET stock = %s, updated_at = %s, updated_by = %s
		WHERE garage_id = %s AND item_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))

	if _, err := database.TxExec(tx, query, newStockFrom, now, updatedBy, req.GarageFrom, req.ItemID, organizationID); err != nil {
		return err
	}

	if err := r.createInventoryMovementInTx(tx, organizationID, &model.InventoryMovement{
		MovementID:   uuid.New().String(),
		ItemID:       req.ItemID,
		GarageID:     req.GarageFrom,
		Quantity:     req.Stock,
		StockBefore:  currentStockFrom.Stock,
		StockFinal:   newStockFrom,
		MovementType: 4,
		Notes:        fmt.Sprintf("Transfer Stock to %s", currentStockDest.GarageName),
		CreatedAt:    now,
		CreatedBy:    updatedBy,
	}); err != nil {
		return err
	}

	if _, err := database.TxExec(tx, query, newStockDest, now, updatedBy, req.GarageDestination, req.ItemID, organizationID); err != nil {
		return err
	}

	if err := r.createInventoryMovementInTx(tx, organizationID, &model.InventoryMovement{
		MovementID:   uuid.New().String(),
		ItemID:       req.ItemID,
		GarageID:     req.GarageDestination,
		Quantity:     req.Stock,
		StockBefore:  currentStockDest.Stock,
		StockFinal:   newStockDest,
		MovementType: 1,
		Notes:        fmt.Sprintf("Transfer Stock from %s", currentStockFrom.GarageName),
		CreatedAt:    now,
		CreatedBy:    updatedBy,
	}); err != nil {
		return err
	}

	rollback = false
	return tx.Commit()
}

func (r *InventoryRepository) createInventoryMovementInTx(tx *sql.Tx, organizationID string, movement *model.InventoryMovement) error {
	query := fmt.Sprintf(`
		INSERT INTO inventory_movement (movement_id, item_id, garage_id, quantity, stock_before, stock_final, movement_type, notes, organization_id, created_at, created_by)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11),
	)

	_, err := database.TxExec(tx, query,
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

func (r *InventoryRepository) UpdateOrderReceived(purchaseID, organizationID, updatedBy string) error {
	now := time.Now()
	query := fmt.Sprintf(`
		UPDATE inventory_orders SET status = %s, complete_date = %s, updated_at = %s, updated_by = %s WHERE purchase_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))

	_, err := database.Exec(r.db, query, 1, now, now, updatedBy, purchaseID, organizationID)
	return err
}

func (r *InventoryRepository) GetOrderForReceive(purchaseID, organizationID string) (*model.InventoryOrder, error) {
	query := `
		SELECT io.item_id, io.garage_id, io.quantity, io.created_by
		FROM inventory_orders io
		WHERE io.purchase_id = %s AND io.organization_id = %s
	`
	query = fmt.Sprintf(query, r.getPlaceholder(1), r.getPlaceholder(2))

	var item model.InventoryOrder
	err := database.QueryRow(r.db, query, purchaseID, organizationID).Scan(
		&item.ItemID,
		&item.GarageID,
		&item.Quantity,
		&item.CreatedBy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &item, nil
}

func (r *InventoryRepository) ReceivePurchaseOrder(purchaseID, organizationID, updatedBy string) error {
	order, err := r.GetOrderForReceive(purchaseID, organizationID)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()

	var currentStock int
	currentStock = 0
	stockQuery := fmt.Sprintf(`
		SELECT stock FROM inventory_item_garage 
		WHERE item_id = %s AND garage_id = %s AND organization_id = %s
		LIMIT 1
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	err = database.TxQueryRow(tx, stockQuery, order.ItemID, order.GarageID, organizationID).Scan(&currentStock)

	now := time.Now()
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if err == sql.ErrNoRows {
		insertQuery := fmt.Sprintf(`
			INSERT INTO inventory_item_garage (item_garage_id, item_id, garage_id, stock, organization_id, created_at, created_by, updated_at, updated_by)
			VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
		`,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9),
		)
		if _, err := database.TxExec(tx, insertQuery,
			uuid.New().String(),
			order.ItemID,
			order.GarageID,
			order.Quantity,
			organizationID,
			now,
			order.CreatedBy,
			now,
			order.CreatedBy,
		); err != nil {
			return err
		}
		currentStock = 0
	} else {
		updateQuery := fmt.Sprintf(`
			UPDATE inventory_item_garage SET stock = %s, updated_at = %s, updated_by = %s
			WHERE item_id = %s AND garage_id = %s AND organization_id = %s
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
		if _, err := database.TxExec(tx, updateQuery, currentStock+order.Quantity, now, updatedBy, order.ItemID, order.GarageID, organizationID); err != nil {
			return err
		}
		currentStock = currentStock + order.Quantity
	}

	movement := &model.InventoryMovement{
		MovementID:   uuid.New().String(),
		ItemID:       order.ItemID,
		GarageID:     order.GarageID,
		Quantity:     order.Quantity,
		StockBefore:  currentStock - order.Quantity,
		StockFinal:   currentStock,
		MovementType: 1,
		Notes:        fmt.Sprintf("Purchase Order %s", purchaseID),
		CreatedAt:    now,
		CreatedBy:    updatedBy,
	}

	if err := r.createInventoryMovementInTx(tx, organizationID, movement); err != nil {
		return err
	}

	updateOrderQuery := fmt.Sprintf(`
		UPDATE inventory_orders SET status = %s, complete_date = %s, updated_at = %s, updated_by = %s 
		WHERE purchase_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))

	if _, err := database.TxExec(tx, updateOrderQuery, 1, now, now, updatedBy, purchaseID, organizationID); err != nil {
		return err
	}

	updateTransactionQuery := fmt.Sprintf(`
		UPDATE transactions SET status = %s
		WHERE reference_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	if _, err := database.TxExec(tx, updateTransactionQuery, 1, purchaseID, organizationID); err != nil {
		return err
	}

	rollback = false
	return tx.Commit()
}

func (r *InventoryRepository) UpsertItemGarageInTx(tx *sql.Tx, itemID, garageID, organizationID, createdBy string, stock int) error {
	var existingStock int
	stockQuery := fmt.Sprintf(`
		SELECT stock FROM inventory_item_garage 
		WHERE item_id = %s AND garage_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	err := database.TxQueryRow(tx, stockQuery, itemID, garageID, organizationID).Scan(&existingStock)
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
		_, err = database.TxExec(tx, query,
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

	_, err = database.TxExec(tx, query, stock, now, createdBy, itemID, garageID, organizationID)
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

func (r *InventoryRepository) CreateSupplierFromExternal(supplier *model.Supplier) (string, error) {
	supplierID := uuid.New().String()
	now := time.Now()

	query := fmt.Sprintf(`
		INSERT INTO supliers (suplier_id, suplier_name, suplier_address, suplier_city, suplier_phone, supliter_email, created_at, created_by, updated_at, updated_by)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		r.getPlaceholder(9), r.getPlaceholder(10),
	)

	_, err := database.Exec(r.db, query,
		supplierID,
		supplier.SupplierName,
		supplier.SupplierAddress,
		supplier.SupplierCity,
		supplier.SupplierPhone,
		supplier.SupplierEmail,
		now,
		supplier.CreatedBy,
		now,
		supplier.UpdatedBy,
	)
	if err != nil {
		return "", err
	}
	return supplierID, nil
}

func (r *InventoryRepository) GetDB() *sql.DB {
	return r.db
}

func (r *InventoryRepository) GetDriver() string {
	return r.driver
}

func (r *InventoryRepository) GetAdminAccountNumber(organizationID string) (string, error) {
	query := fmt.Sprintf(`
		SELECT aa.account_number
		FROM assistant_accounts aa
		WHERE aa.user_type = 1
		  AND aa.status = 1
		  AND %s
		LIMIT 1
	`, r.getPlaceholder(1))

	var accountNumber string
	if err := database.QueryRow(r.db, query, organizationID).Scan(&accountNumber); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return accountNumber, nil
}

func (r *InventoryRepository) IsAdminByAccountNumber(accountNumber string) (bool, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(1) FROM assistant_accounts
		WHERE account_number = %s AND user_type = 1 AND status = 1
	`, r.getPlaceholder(1))

	var count int
	if err := database.QueryRow(r.db, query, accountNumber).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *InventoryRepository) CreatePurchaseOrder(order *model.InventoryOrder) error {
	if order.PurchaseID == "" {
		order.PurchaseID = uuid.New().String()
	}
	now := time.Now()
	order.CreatedAt = now
	order.UpdatedAt = now

	query := fmt.Sprintf(`
		INSERT INTO inventory_orders (purchase_id, request_id, item_id, item_category, garage_id, suplier_id, quantity, item_price, total_amount, status, organization_id, created_at, created_by, updated_at, updated_by, transaction_date)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14), r.getPlaceholder(15), r.getPlaceholder(16),
	)

	_, err := database.Exec(r.db, query,
		order.PurchaseID,
		order.RequestID,
		order.ItemID,
		order.ItemCategory,
		order.GarageID,
		order.SupplierID,
		order.Quantity,
		order.Amount,
		order.TotalAmount,
		2,
		order.OrganizationID,
		order.CreatedAt,
		order.CreatedBy,
		order.UpdatedAt,
		order.UpdatedBy,
		order.TransactionDate,
	)
	return err
}

func (r *InventoryRepository) CreateInventoryTransaction(organizationID string, txn *model.InventoryTransaction) error {
	now := time.Now()
	if txn.CreatedAt.IsZero() {
		txn.CreatedAt = now
	}
	query := fmt.Sprintf(`
		INSERT INTO transactions (transaction_id, transaction_type, order_type, transaction_category, transaction_item, invoice_number, description, transaction_date, payment_type, organization_id, amount, created_by, created_at, status, reference_id)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10),
		r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14), r.getPlaceholder(15),
	)

	_, err := database.Exec(r.db, query,
		txn.TransactionID,
		txn.TransactionType,
		1,
		txn.TransactionCategory,
		txn.TransactionItem,
		txn.InvoiceNumber,
		txn.Description,
		txn.TransactionDateStr,
		txn.PaymentType,
		organizationID,
		txn.Amount,
		txn.CreatedBy,
		txn.CreatedAt,
		txn.Status,
		txn.ReferenceID,
	)
	return err
}

func (r *InventoryRepository) GetItemLocations(itemID string) ([]model.InventoryItemLocation, error) {
	query := fmt.Sprintf(`
		SELECT ig.garage_id, g.garage_name, g.garage_address, g.garage_city, ig.stock, ig.updated_at
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
			&loc.GarageID,
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

func (r *InventoryRepository) GetItemMovements(organizationID, itemID, startDate, endDate, garageID string) ([]model.InventoryItemMovement, error) {
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
	if garageID != "" {
		query += fmt.Sprintf(" AND im.garage_id = %s", r.getPlaceholder(pos))
		args = append(args, garageID)
		pos++
	}

	query += " ORDER BY im.created_at DESC"
	fmt.Println("GetItemMovements query: ", query, organizationID, itemID, startDate, endDate, garageID)

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

func (r *InventoryRepository) GetItemOrderHistory(organizationID, itemID, startDate, endDate string) ([]model.InventoryHistory, error) {
	query := `
		SELECT io.request_id, io.purchase_id, io.quantity, io.item_price, io.total_amount, io.transaction_date, io.complete_date, e.fullname as received_by, ir.received_at 
		FROM inventory_orders io
		LEFT JOIN inventory_request ir ON ir.request_id = io.request_id
		LEFT JOIN employee e ON e.uuid = ir.received_by
		WHERE io.organization_id = $1 AND io.item_id = $2 AND io.status = 1
	`

	var args []interface{}
	args = append(args, organizationID, itemID)
	pos := 3

	if startDate != "" {
		query += fmt.Sprintf(" AND DATE(io.transaction_date) >= %s", r.getPlaceholder(pos))
		args = append(args, startDate)
		pos++
	}
	if endDate != "" {
		query += fmt.Sprintf(" AND DATE(io.transaction_date) <= %s", r.getPlaceholder(pos))
		args = append(args, endDate)
		pos++
	}
	query += " ORDER BY io.transaction_date DESC"
	fmt.Println("GetItemOrderHistory query: ", query, organizationID, itemID, startDate, endDate)

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []model.InventoryHistory
	for rows.Next() {
		var o model.InventoryHistory
		var receivedBy sql.NullString
		var receivedAt sql.NullTime
		var completeDate sql.NullString
		if err := rows.Scan(
			&o.RequestID,
			&o.PurchaseID,
			&o.Quantity,
			&o.ItemPrice,
			&o.TotalAmount,
			&o.TransactionDate,
			&completeDate,
			&receivedBy,
			&receivedAt,
		); err != nil {
			return nil, err
		}
		if completeDate.Valid {
			o.CompleteDate = completeDate.String
		} else {
			o.CompleteDate = ""
		}
		if receivedBy.Valid {
			o.ReceivedBy = receivedBy.String
		} else {
			o.ReceivedBy = ""
		}
		if receivedAt.Valid {
			o.ReceivedAt = receivedAt.Time.Format("2006-01-02 15:04:05")
		} else {
			o.ReceivedAt = ""
		}
		histories = append(histories, o)
	}
	return histories, nil
}

func (r *InventoryRepository) CancelPurchaseOrder(purchaseID, organizationID, updatedBy string) error {
	query := `
		UPDATE inventory_orders
		SET updated_at = $1, updated_by = $2, status = 0
		WHERE purchase_id = $3 AND organization_id = $4
	`
	_, err := r.db.Exec(query, time.Now().Format("2006-01-02 15:04:05"), updatedBy, purchaseID, organizationID)
	if err != nil {
		return err
	}
	return nil
}

func (r *InventoryRepository) CancelInventoryTransaction(organizationID, updatedBy, purchaseID string) error {
	query := `
		UPDATE transactions
		SET updated_at = $1, updated_by = $2, status = 0
		WHERE reference_id = $3 AND organization_id = $4
	`
	_, err := r.db.Exec(query, time.Now().Format("2006-01-02 15:04:05"), updatedBy, purchaseID, organizationID)
	if err != nil {
		return err
	}
	return nil
}
