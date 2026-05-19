package repository

import (
	"database/sql"
	"fmt"
	"service-travego/database"
)

// PaymentRepository adalah interface untuk akses data payment
type PaymentRepository interface {
	GetOrderTotalAmount(orderID string, orderType int64) (int64, error)
	UpdatePaymentStatus(orderID string, orderType int64, status int) error
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

	query := fmt.Sprintf("UPDATE %s SET payment_status = %s WHERE order_id = %s", table, r.getPlaceholder(1), r.getPlaceholder(2))
	_, err := database.Exec(r.db, query, status, orderID)
	return err
}

func (r *paymentRepository) getPlaceholder(n int) string {
	if r.driver == "postgres" {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}
