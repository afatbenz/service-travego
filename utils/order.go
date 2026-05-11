package utils

import (
	"fmt"
	"time"
)

func GenerateInvoiceNumber(orderType int, now time.Time, sequence int) string {
	if sequence < 1 {
		sequence = 1
	}
	return fmt.Sprintf("INV-%d%s-000%d", orderType, now.Format("01200602"), sequence)
}
