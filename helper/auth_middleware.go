package helper

import (
	"fmt"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// AuthTokenMiddleware stores the auth token from response in locals
// This middleware should be used after the login handler to capture the token
func AuthTokenMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// If auth_token is already in locals (set by login handler), keep it
		// This middleware can be extended to extract token from Authorization header in the future
		if token := c.Locals("auth_token"); token != nil {
			// Token already set, continue
		}
		return c.Next()
	}
}

// JWTAuthorizationMiddleware extracts JWT token from Authorization header and validates it
// Sets user_id in locals for use in handlers
func JWTAuthorizationMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":         "error",
				"message":        "Authorization header is required",
				"data":           nil,
				"transaction_id": GetTransactionID(c),
			})
		}

		// Extract token from "Bearer <token>" or just "<token>"
		var tokenString string
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			// Format: "Bearer <token>"
			tokenString = parts[1]
		} else if len(parts) == 1 {
			// Format: "<token>" (without Bearer prefix)
			tokenString = parts[0]
		} else {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":         "error",
				"message":        "Invalid authorization header format. Expected 'Bearer <token>' or '<token>'",
				"data":           nil,
				"transaction_id": GetTransactionID(c),
			})
		}

		// Parse and validate token
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "your-secret-key-change-in-production"
		}

		token, err := jwt.ParseWithClaims(tokenString, &AuthTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":         "error",
				"message":        "Invalid or expired token",
				"data":           nil,
				"transaction_id": GetTransactionID(c),
			})
		}

		// Extract claims
		if claims, ok := token.Claims.(*AuthTokenClaims); ok && token.Valid {
			// Store user_id in locals
			c.Locals("user_id", claims.UserID)
			c.Locals("username", claims.Username)
			c.Locals("organization_id", claims.OrganizationID)
			c.Locals("organization_role", claims.OrganizationRole)
			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":         "error",
			"message":        "Invalid token",
			"data":           nil,
			"transaction_id": GetTransactionID(c),
		})
	}
}
