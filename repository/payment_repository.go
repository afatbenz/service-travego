package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"service-travego/database"
	"service-travego/model"
	"service-travego/utils"
)

// PaymentRepository adalah interface untuk akses data payment
type PaymentRepository interface {
	GetOrderTotalAmount(orderID string, orderType int64) (int64, error)
	UpdatePaymentStatus(orderID string, orderType int64, status int) error
	UpdateOrderStatus(orderID string, orderType int64, status int, paymentStatus int) error
	GetOrderDetails(orderID string) (organizationID string, totalAmount int64, orderType int64, err error)
	UpdatePaymentOrderNotification(orderID string, organizationID string, totalAmount int64, paymentAmount float64, transactionID string, paymentType string) error
	InsertPaymentMidtrans(req *model.MidtransWebhookRequest, createdAt string) error
	InsertPaymentOrder(paymentID string, orderType int64, orderID string, organizationID string, paymentType int, paymentMethod int, invoiceNumber string, createdAt string, createdBy string) error
	GetPaymentOrderMeta(orderID string, organizationID string) (invoiceNumber string, orderType int64, createdBy string, err error)
	GetLatestPaymentOrderRemainingAmount(orderID string, organizationID string, orderType int64) (remainingAmount sql.NullFloat64, err error)
	GetFleetOrderEmailData(orderID string, organizationID string) (customerName string, customerEmail string, fleetName string, pickupLocation string, startDate time.Time, endDate time.Time, destination string, err error)
	TransactionExistsByInvoice(organizationID string, invoiceNumber string) (bool, error)
	InsertTransactionMidtrans(transactionID string, orderType int64, invoiceNumber string, description string, transactionDate time.Time, status int, amount float64, organizationID string, transactionType int, TransactionItem int, createdAt time.Time, createdBy string) error
	GetNextInvoiceNumber(organizationID string, orderType int) (string, error)
}

type paymentRepository struct {
	db     *sql.DB
	driver string
}

// NewPaymentRepository membuat instance baru dari PaymentRepository
func NewPaymentRepository(db *sql.DB, driver string) PaymentRepository {
	return &paymentRepository{
		db:     db,
		driver: driver,
	}
}

// GetOrderDetails retrieves organization_id and total_amount from fleet_orders or tour_package_orders
func (r *paymentRepository) GetOrderDetails(orderID string) (string, int64, int64, error) {
	// Try fleet_orders first
	query := fmt.Sprintf("SELECT organization_id, total_amount FROM fleet_orders WHERE order_id = %s LIMIT 1", r.getPlaceholder(1))
	var orgID string
	var totalAmount int64
	err := database.QueryRow(r.db, query, orderID).Scan(&orgID, &totalAmount)
	if err == nil {
		return orgID, totalAmount, 1, nil
	}

	// Try tour_package_orders
	query = fmt.Sprintf("SELECT organization_id, total_amount FROM tour_package_orders WHERE order_id = %s LIMIT 1", r.getPlaceholder(1))
	err = database.QueryRow(r.db, query, orderID).Scan(&orgID, &totalAmount)
	if err == nil {
		return orgID, totalAmount, 2, nil
	}

	return "", 0, 0, fmt.Errorf("order not found: %s", orderID)
}

// UpdatePaymentOrderNotification updates payment_orders on Midtrans notification
func (r *paymentRepository) UpdatePaymentOrderNotification(orderID string, organizationID string, totalAmount int64, paymentAmount float64, transactionID string, paymentType string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	orgExpr := "organization_id = " + r.getPlaceholder(2)
	d := strings.ToLower(r.driver)
	if d == "postgres" || d == "pgx" || d == "pq" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(2)
	}

	orgExprUpdate := "organization_id = " + r.getPlaceholder(7)
	if d == "postgres" || d == "pgx" || d == "pq" {
		orgExprUpdate = "organization_id::text = " + r.getPlaceholder(7)
	}

	sumPaidQuery := fmt.Sprintf(`
		SELECT COALESCE(SUM(COALESCE(payment_amount, 0)), 0)
		FROM payment_orders
		WHERE order_id = %s AND %s AND COALESCE(status, 0) > 0
	`, r.getPlaceholder(1), orgExpr)

	var sumPaid float64
	if qerr := database.TxQueryRow(tx, sumPaidQuery, orderID, organizationID).Scan(&sumPaid); qerr != nil && qerr != sql.ErrNoRows {
		err = qerr
		return err
	}

	txExistsQuery := fmt.Sprintf(`
		SELECT 1
		FROM payment_orders
		WHERE order_id = %s AND %s AND transaction_id = %s AND COALESCE(status, 0) > 0
		LIMIT 1
	`, r.getPlaceholder(1), orgExpr, r.getPlaceholder(3))

	var one int
	txExistsErr := database.TxQueryRow(tx, txExistsQuery, orderID, organizationID, transactionID).Scan(&one)
	transactionAlreadyCounted := txExistsErr == nil
	if txExistsErr != nil && txExistsErr != sql.ErrNoRows {
		err = txExistsErr
		return err
	}

	totalPaid := sumPaid
	if !transactionAlreadyCounted {
		totalPaid += paymentAmount
	}

	remainingAmount := float64(totalAmount) - totalPaid
	if remainingAmount < 0 {
		remainingAmount = 0
	}

	query := fmt.Sprintf(`
		UPDATE payment_orders
		SET total_amount = %s, payment_amount = %s, transaction_id = %s, status = 1, remaining_amount = %s, notes = 'Midtrans - ' || %s
		WHERE order_id = %s AND %s`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), orgExprUpdate)

	_, err = database.TxExec(tx, query, totalAmount, paymentAmount, transactionID, remainingAmount, paymentType, orderID, organizationID)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

// InsertPaymentMidtrans inserts Midtrans notification payload into payment_midtrans
func (r *paymentRepository) InsertPaymentMidtrans(req *model.MidtransWebhookRequest, createdAt string) error {
	query := fmt.Sprintf(`
		INSERT INTO payment_midtrans
			(transaction_id, transaction_status, order_id, payment_type, merchant_id, gross_amount, currency, transaction_time, payment_status, created_at)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
		r.getPlaceholder(9), r.getPlaceholder(10))

	_, err := database.Exec(r.db, query,
		req.TransactionID,
		req.TransactionStatus,
		req.OrderID,
		req.PaymentType,
		req.MerchantID,
		req.GrossAmount,
		req.Currency,
		req.TransactionTime,
		req.PaymentStatus,
		createdAt,
	)
	return err
}

// InsertPaymentOrder inserts a new row into payment_orders
func (r *paymentRepository) InsertPaymentOrder(paymentID string, orderType int64, orderID string, organizationID string, paymentType int, paymentMethod int, invoiceNumber string, createdAt string, createdBy string) error {
	query := fmt.Sprintf(`
		INSERT INTO payment_orders (payment_id, order_type, order_id, organization_id, payment_type, payment_method, invoice_number, created_at, created_by)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9))

	// Ensure empty strings are passed as NULL for UUID columns using sql.NullString
	pid := sql.NullString{String: paymentID, Valid: paymentID != ""}
	oid := sql.NullString{String: orderID, Valid: orderID != ""}
	orgid := sql.NullString{String: organizationID, Valid: organizationID != ""}
	cb := sql.NullString{String: createdBy, Valid: createdBy != ""}

	fmt.Printf("[DEBUG] InsertPaymentOrder - pid: %v, oid: %v, orgid: %v, cb: %v\n", pid, oid, orgid, cb)

	_, err := database.Exec(r.db, query, pid, orderType, oid, orgid, paymentType, paymentMethod, invoiceNumber, createdAt, cb)
	return err
}

func (r *paymentRepository) GetPaymentOrderMeta(orderID string, organizationID string) (string, int64, string, error) {
	createdByExpr := "COALESCE(created_by, '')"
	d := strings.ToLower(r.driver)
	if d == "postgres" || d == "pgx" || d == "pq" {
		createdByExpr = "COALESCE(created_by::text, '')"
	}
	query := fmt.Sprintf(`
		SELECT
			COALESCE(invoice_number, ''),
			COALESCE(order_type, 0),
			%s
		FROM payment_orders
		WHERE order_id = %s AND organization_id = %s
		ORDER BY created_at DESC
		LIMIT 1
	`, createdByExpr, r.getPlaceholder(1), r.getPlaceholder(2))

	var invoiceNumber string
	var orderType int64
	var createdBy string
	if err := database.QueryRow(r.db, query, orderID, organizationID).Scan(&invoiceNumber, &orderType, &createdBy); err != nil {
		return "", 0, "", err
	}
	return invoiceNumber, orderType, createdBy, nil
}

func (r *paymentRepository) GetLatestPaymentOrderRemainingAmount(orderID string, organizationID string, orderType int64) (sql.NullFloat64, error) {
	orgExpr := "organization_id = " + r.getPlaceholder(3)
	d := strings.ToLower(r.driver)
	if d == "postgres" || d == "pgx" || d == "pq" {
		orgExpr = "organization_id::text = " + r.getPlaceholder(3)
	}

	query := fmt.Sprintf(`
		SELECT remaining_amount
		FROM payment_orders
		WHERE order_id = %s AND order_type = %s AND %s
		ORDER BY created_at DESC
		LIMIT 1
	`, r.getPlaceholder(1), r.getPlaceholder(2), orgExpr)

	var remaining sql.NullFloat64
	if err := database.QueryRow(r.db, query, orderID, orderType, organizationID).Scan(&remaining); err != nil {
		return sql.NullFloat64{}, err
	}
	return remaining, nil
}

func (r *paymentRepository) GetFleetOrderEmailData(orderID string, organizationID string) (string, string, string, string, time.Time, time.Time, string, error) {
	orgExpr := "fo.organization_id = " + r.getPlaceholder(2)
	destExpr := "''"
	d := strings.ToLower(r.driver)
	if d == "postgres" || d == "pgx" || d == "pq" {
		orgExpr = "fo.organization_id::text = " + r.getPlaceholder(2)
		destExpr = "COALESCE(string_agg(d.location, ', ' ORDER BY d.location), '')"
	} else {
		destExpr = "COALESCE(GROUP_CONCAT(d.location SEPARATOR ', '), '')"
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(c.customer_name, ''),
			COALESCE(c.customer_email, ''),
			COALESCE(f.fleet_name, ''),
			COALESCE(fo.pickup_location, ''),
			fo.start_date,
			fo.end_date,
			%s
		FROM fleet_orders fo
		INNER JOIN fleets f ON fo.fleet_id = f.uuid
		INNER JOIN customer_orders co ON co.order_id = fo.order_id
		INNER JOIN customers c ON c.customer_id = co.customer_id
		LEFT JOIN fleet_order_destinations d ON d.order_id = fo.order_id
		WHERE fo.order_id = %s AND %s
		GROUP BY c.customer_name, c.customer_email, f.fleet_name, fo.pickup_location, fo.start_date, fo.end_date
		LIMIT 1
	`, destExpr, r.getPlaceholder(1), orgExpr)

	var customerName, customerEmail, fleetName, pickupLocation, destination string
	var startDate, endDate time.Time
	if err := database.QueryRow(r.db, query, orderID, organizationID).Scan(&customerName, &customerEmail, &fleetName, &pickupLocation, &startDate, &endDate, &destination); err != nil {
		return "", "", "", "", time.Time{}, time.Time{}, "", err
	}
	return customerName, customerEmail, fleetName, pickupLocation, startDate, endDate, destination, nil
}

func (r *paymentRepository) TransactionExistsByInvoice(organizationID string, invoiceNumber string) (bool, error) {
	query := fmt.Sprintf(`
		SELECT 1
		FROM transactions
		WHERE organization_id = %s AND invoice_number = %s
		LIMIT 1
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	var one int
	err := database.QueryRow(r.db, query, organizationID, invoiceNumber).Scan(&one)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}

func (r *paymentRepository) InsertTransactionMidtrans(transactionID string, orderType int64, invoiceNumber string, description string, transactionDate time.Time, status int, amount float64, organizationID string, transactionType int, TransactionItem int, createdAt time.Time, createdBy string) error {
	query := fmt.Sprintf(`
		INSERT INTO transactions
			(transaction_id, order_type, invoice_number, description, transaction_date, status, amount, organization_id, transaction_type, transaction_item, created_at, created_by)
		VALUES
			(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6),
		r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12))

	orgID := sql.NullString{String: organizationID, Valid: strings.TrimSpace(organizationID) != ""}
	cb := sql.NullString{String: createdBy, Valid: strings.TrimSpace(createdBy) != ""}

	_, err := database.Exec(
		r.db,
		query,
		transactionID,
		orderType,
		invoiceNumber,
		description,
		transactionDate,
		status,
		amount,
		orgID,
		transactionType,
		TransactionItem,
		createdAt,
		cb,
	)
	return err
}

// GetNextInvoiceNumber generates the next invoice number using utils
func (r *paymentRepository) GetNextInvoiceNumber(organizationID string, orderType int) (string, error) {
	return utils.GenerateInvoiceNumber(r.db, r.driver, organizationID, orderType, time.Now())
}

// GetOrderTotalAmount mengambil total_amount dari tabel order yang sesuai
func (r *paymentRepository) GetOrderTotalAmount(orderID string, orderType int64) (int64, error) {
	var table string
	if orderType == 1 {
		table = "fleet_orders"
	} else if orderType == 2 {
		table = "tour_package_orders"
	} else {
		return 0, fmt.Errorf("invalid order type: %d", orderType)
	}

	query := fmt.Sprintf("SELECT total_amount FROM %s WHERE order_id = %s LIMIT 1", table, r.getPlaceholder(1))
	var totalAmount int64
	err := database.QueryRow(r.db, query, orderID).Scan(&totalAmount)
	if err != nil {
		return 0, err
	}

	return totalAmount, nil
}

// UpdatePaymentStatus mengupdate kolom payment_status di tabel order yang sesuai
func (r *paymentRepository) UpdatePaymentStatus(orderID string, orderType int64, status int) error {
	var table string
	if orderType == 1 {
		table = "fleet_orders"
	} else if orderType == 2 {
		table = "tour_package_orders"
	} else {
		return fmt.Errorf("invalid order type: %d", orderType)
	}

	query := fmt.Sprintf("UPDATE %s SET payment_status = %s, status = %s WHERE order_id = %s", table, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	_, err := database.Exec(r.db, query, status, status, orderID)
	return err
}

// UpdateOrderStatus updates kolom status di tabel order yang sesuai
func (r *paymentRepository) UpdateOrderStatus(orderID string, orderType int64, status int, paymentStatus int) error {
	var table string
	if orderType == 1 {
		table = "fleet_orders"
	} else if orderType == 2 {
		table = "tour_package_orders"
	} else {
		return fmt.Errorf("invalid order type: %d", orderType)
	}

	query := fmt.Sprintf("UPDATE %s SET status = %s, payment_status = %s, updated_at = now() WHERE order_id = %s", table, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	fmt.Println("UpdateOrderStatus query:")
	fmt.Println(query)
	fmt.Println(status, orderID)
	_, err := database.Exec(r.db, query, status, paymentStatus, orderID)
	return err
}

func (r *paymentRepository) getPlaceholder(n int) string {
	d := strings.ToLower(r.driver)
	if d == "postgres" || d == "pgx" || d == "pq" {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}
