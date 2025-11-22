package helper

import (
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
