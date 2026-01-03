package helper

import (
	"fmt"
	"strings"
	"unicode"
)

// IsEmpty checks if a string is empty or contains only whitespace
func IsEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

// ToTitle converts first letter of each word to uppercase
func ToTitle(s string) string {
	if len(s) == 0 {
		return s
	}

	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	for i := 1; i < len(runes); i++ {
		if unicode.IsSpace(runes[i-1]) {
			runes[i] = unicode.ToUpper(runes[i])
		} else {
			runes[i] = unicode.ToLower(runes[i])
		}
	}
	return string(runes)
}

// Truncate truncates a string to a specified length
func Truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// NormalizePhoneNumber normalizes phone number format
// If phone starts with "0", replace with "62"
// If phone already starts with "62", keep as is
func NormalizePhoneNumber(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return phone
	}

	// If already starts with 62, return as is
	if strings.HasPrefix(phone, "62") {
		return phone
	}

	// If starts with 0, replace with 62
	if strings.HasPrefix(phone, "0") {
		return "62" + phone[1:]
	}

	// Otherwise return as is
	return phone
}

// FormatRupiah formats a float64 amount to Rupiah format (e.g. Rp2.750.000,00)
func FormatRupiah(amount float64) string {
	// Format with 2 decimal places
	s := fmt.Sprintf("%.2f", amount)
	parts := strings.Split(s, ".")
	integerPart := parts[0]
	decimalPart := parts[1]

	// Add thousand separators
	var result []byte
	n := len(integerPart)
	for i, r := range integerPart {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(r))
	}

	return "Rp" + string(result) + "," + decimalPart
}
