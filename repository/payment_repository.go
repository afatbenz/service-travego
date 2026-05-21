package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"service-travego/database"
	"service-travego/utils"
)

// PaymentRepository adalah interface untuk akses data payment
type PaymentRepository interface {
	GetOrderTotalAmount(orderID string, orderType int64) (int64, error)
	UpdatePaymentStatus(orderID string, orderType int64, status int) error
	GetOrderDetails(orderID string) (organizationID string, totalAmount int64, orderType int64, err error)
	UpdatePaymentOrder(orderID string, organizationID string, grossAmount float64, settledAt string, settledBy string, status int, remainingAmount float64) error
	InsertPaymentOrder(paymentID string, orderType int64, orderID string, organizationID string, paymentType int, paymentMethod int, invoiceNumber string, createdAt string, createdBy string) error
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

// UpdatePaymentOrder updates the row in payment_orders table
func (r *paymentRepository) UpdatePaymentOrder(orderID string, organizationID string, grossAmount float64, settledAt string, settledBy string, status int, remainingAmount float64) error {
	query := fmt.Sprintf(`
		UPDATE payment_orders 
		SET total_amount = %s, settled_at = %s, notes = %s, status = %s, remaining_amount = %s 
		WHERE order_id = %s AND organization_id = %s`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7))
	fmt.Printf("[DEBUG] UpdatePaymentOrder - query: %s\n", query)

	_, err := database.Exec(r.db, query, grossAmount, settledAt, settledBy, status, remainingAmount, orderID, organizationID)
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

func (r *paymentRepository) getPlaceholder(n int) string {
	d := strings.ToLower(r.driver)
	if d == "postgres" || d == "pgx" || d == "pq" {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}
