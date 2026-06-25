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
	switch orderType {
	case 1:
		prefix = "FO"
	case 2:
		prefix = "TO"
	default:
		prefix = "CO"
	}

	truncatedCode := orgCode
	if len(orgCode) >= 5 {
		truncatedCode = orgCode[:3] + orgCode[len(orgCode)-2:]
	}

	timePart := time.Now().Format("06020115")
	return fmt.Sprintf("%s-%s%d-%s", prefix, timePart, count+1, truncatedCode)
}

func GenerateTripID(orgCode string, seq int, now time.Time) string {
	timePart := now.Format("060102150405")
	finalTime := timePart[len(timePart)-4:]
	return fmt.Sprintf("SJL-%s%04d-%s", finalTime, seq, orgCode)
}

func placeholder(driver string, pos int) string {
	if driver == "postgres" || driver == "pgx" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}

func formatInvoiceNumber(orderType int, now time.Time, sequence int) string {
	datePart := now.Format("060201")
	ms := now.Nanosecond() / 1e6
	timePart := fmt.Sprintf("%03d", ms)
	return fmt.Sprintf("INV-%s%d-%s%04d", datePart, orderType, timePart, sequence)
}

func GenerateRequestNumber(db *sql.DB, driver, organizationID string) (string, error) {
	orgExpr := "organization_id = " + placeholder(driver, 1)
	if driver == "postgres" || driver == "pgx" {
		orgExpr = "organization_id::text = " + placeholder(driver, 1)
	}
	query := fmt.Sprintf("SELECT COUNT(1) FROM inventory_request WHERE %s ", orgExpr)

	var count int
	if err := database.QueryRow(db, query, organizationID).Scan(&count); err != nil {
		return "", err
	}
	seq := count + 1
	if seq < 1 {
		seq = 1
	}

	var seqStr string
	if seq > 9999 {
		seqStr = fmt.Sprintf("%d", seq)
	} else if seq > 999 {
		seqStr = fmt.Sprintf("0%d", seq)
	} else if seq > 99 {
		seqStr = fmt.Sprintf("00%d", seq)
	} else {
		seqStr = fmt.Sprintf("000%d", seq)
	}

	datePart := time.Now().Format("0601")
	return fmt.Sprintf("REQ-%s-00%s", datePart, seqStr), nil
}

func GenerateItemSKU(db *sql.DB, driver, organizationID string) (string, error) {
	orgExpr := "organization_id = " + placeholder(driver, 1)
	if driver == "postgres" || driver == "pgx" {
		orgExpr = "organization_id::text = " + placeholder(driver, 1)
	}
	query := fmt.Sprintf("SELECT COUNT(1) FROM inventory_items WHERE %s ", orgExpr)

	var count int
	if err := database.QueryRow(db, query, organizationID).Scan(&count); err != nil {
		return "", err
	}
	seq := count + 1
	if seq < 1 {
		seq = 1
	}

	now := time.Now()
	yearMonth := now.Format("0606")

	var skuSeq string
	if seq > 9999 {
		skuSeq = fmt.Sprintf("%d", seq)
	} else if seq > 999 {
		skuSeq = fmt.Sprintf("0%d", seq)
	} else if seq > 99 {
		skuSeq = fmt.Sprintf("00%d", seq)
	} else {
		skuSeq = fmt.Sprintf("000%d", seq)
	}

	return fmt.Sprintf("SKU-%s0-%s", skuSeq, yearMonth), nil
}

func GeneratePurchaseOrderID(db *sql.DB, driver, organizationID string) (string, error) {
	orgExpr := "organization_id = " + placeholder(driver, 1)
	if driver == "postgres" || driver == "pgx" {
		orgExpr = "organization_id::text = " + placeholder(driver, 1)
	}
	query := fmt.Sprintf("SELECT COUNT(1) FROM inventory_orders WHERE %s ", orgExpr)

	var count int
	if err := database.QueryRow(db, query, organizationID).Scan(&count); err != nil {
		return "", err
	}
	seq := count + 1
	if seq < 1 {
		seq = 1
	}

	var seqStr string
	if seq > 9999 {
		seqStr = fmt.Sprintf("%d", seq)
	} else if seq > 999 {
		seqStr = fmt.Sprintf("0%d", seq)
	} else if seq > 99 {
		seqStr = fmt.Sprintf("00%d", seq)
	} else {
		seqStr = fmt.Sprintf("000%d", seq)
	}

	now := time.Now()
	datePart := now.Format("06-01")
	return fmt.Sprintf("PO-00%s-%s", seqStr, datePart), nil
}

// GenerateSubsInvoiceID generates invoice ID for subscription with format TRV-{2digitrandom}{sequence}-{MMYY}
func GenerateSubsInvoiceID(db *sql.DB, driver string) (string, error) {
	query := "SELECT COUNT(1) FROM travego_transactions"
	var count int
	if err := database.QueryRow(db, query).Scan(&count); err != nil {
		return "", err
	}
	seq := count + 1

	// Generate 2 random digits
	rand1 := time.Now().UnixNano() % 100
	randPart := fmt.Sprintf("%02d", rand1)

	// Sequence as 000{count+1}
	var seqStr string
	if seq > 9999 {
		seqStr = fmt.Sprintf("%d", seq)
	} else if seq > 999 {
		seqStr = fmt.Sprintf("0%d", seq)
	} else if seq > 99 {
		seqStr = fmt.Sprintf("00%d", seq)
	} else {
		seqStr = fmt.Sprintf("000%d", seq)
	}

	// Date as MMYY
	datePart := time.Now().Format("0106")

	return fmt.Sprintf("TRV-%s%s-%s", randPart, seqStr, datePart), nil
}
