package utils

import (
	"database/sql"
	"fmt"
	"service-travego/database"
	"time"
)

func GenerateInvoiceNumber(db *sql.DB, driver, organizationID string, orderType int, now time.Time) (string, error) {
	orgExpr := "organization_id = " + placeholder(driver, 2)
	if driver == "postgres" || driver == "pgx" {
		orgExpr = "organization_id::text = " + placeholder(driver, 2)
	}
	query := fmt.Sprintf(`SELECT COUNT(1) FROM transactions WHERE order_type = %s AND %s`, placeholder(driver, 1), orgExpr)

	var count int
	if err := database.QueryRow(db, query, orderType, organizationID).Scan(&count); err != nil {
		return "", err
	}
	seq := count + 1
	if seq < 1 {
		seq = 1
	}
	return formatInvoiceNumber(orderType, now, seq), nil
}

func GenerateInvoiceNumberTx(tx *sql.Tx, driver, organizationID string, orderType int, now time.Time) (string, error) {
	orgExpr := "organization_id = " + placeholder(driver, 2)
	if driver == "postgres" || driver == "pgx" {
		orgExpr = "organization_id::text = " + placeholder(driver, 2)
	}
	query := fmt.Sprintf(`SELECT COUNT(1) FROM transactions WHERE order_type = %s AND %s`, placeholder(driver, 1), orgExpr)

	var count int
	if err := database.TxQueryRow(tx, query, orderType, organizationID).Scan(&count); err != nil {
		return "", err
	}
	seq := count + 1
	if seq < 1 {
		seq = 1
	}
	return formatInvoiceNumber(orderType, now, seq), nil
}

func placeholder(driver string, pos int) string {
	if driver == "postgres" || driver == "pgx" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}

func formatInvoiceNumber(orderType int, now time.Time, sequence int) string {
	return fmt.Sprintf("INV-%s0%d-%05d", now.Format("02012006"), orderType, sequence)
}
