package helper

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

const TransactionIDKey = "transaction_id"

func GenerateTransactionID() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "DEV"
	}

	envPrefix := strings.ToUpper(env)
	if len(envPrefix) > 3 {
		envPrefix = envPrefix[:3]
	} else {
		envPrefix = strings.ToUpper(envPrefix) + strings.Repeat("X", 3-len(envPrefix))
	}

	now := time.Now()
	timestamp := now.Format("060102150405")

	rand.Seed(time.Now().UnixNano())
	randomDigits := fmt.Sprintf("%03d", rand.Intn(1000))

	return envPrefix + timestamp + randomDigits
}

func GetTransactionID(c *fiber.Ctx) string {
	if txID, ok := c.Locals(TransactionIDKey).(string); ok {
		return txID
	}
	return GenerateTransactionID()
}

func TransactionIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		txID := GenerateTransactionID()
		c.Locals(TransactionIDKey, txID)
		return c.Next()
	}
}
