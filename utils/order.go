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

// GenerateOrderID generates order ID based on order type
func GenerateOrderID(orderType int, orgCode string, count int) string {
	var prefix string
	if orderType == 1 {
		prefix = "FO"
	} else if orderType == 2 {
		prefix = "TP"
	} else {
		prefix = "ORD"
	}

	truncatedCode := orgCode
	if len(orgCode) >= 5 {
		truncatedCode = orgCode[:3] + orgCode[len(orgCode)-2:]
	}

	timePart := time.Now().Format("06020115")
	return fmt.Sprintf("%s-%s%d-%s", prefix, timePart, count+1, truncatedCode)
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
