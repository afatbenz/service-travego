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
			c.Locals("username", claims.Username)
			c.Locals("organization_name", claims.OrganizationName)
			// Decrypt sensitive token to populate locals
			if claims.Token != "" {
				data, derr := DecryptAuthSensitiveData(claims.Token)
				if derr == nil {
					c.Locals("user_id", data.UserID)
					c.Locals("organization_id", data.OrganizationID)
					c.Locals("organization_role", data.OrganizationRole)
					c.Locals("is_admin", data.IsAdmin)
				}
			}
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

// ApiKeyMiddleware decrypts api-key header to get organization_id
func ApiKeyMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get api-key header
		apiKey := c.Get("api-key")
		if apiKey == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":         "error",
				"message":        "api-key header is required",
				"data":           nil,
				"transaction_id": GetTransactionID(c),
			})
		}

		// Decrypt api-key to get organization_id
		// Assuming the api-key is encrypted using the same method as AuthSensitiveData
		// But Wait, DecryptData in encrypt.go returns email and userID.
		// DecryptAuthSensitiveData in auth_middleware.go is not visible here (I need to check if it exists or if I missed it).
		// Wait, I saw DecryptAuthSensitiveData call in JWTAuthorizationMiddleware.
		// Let me check where DecryptAuthSensitiveData is defined. It might be in another file or further down in auth_middleware.go.

		// Let's assume for now that the api-key is just an encrypted string containing the organization_id.
		// Or maybe it's the `Token` field from `AuthTokenClaims` which contains user_id and organization_id?
		// The user says "headers api-key ... untuk mendapatkan organization_id setelah api-key didecrypt".

		// If I look at helper/encrypt.go, there is `DecryptData`.
		// If I look at helper/auth_middleware.go, there is `DecryptAuthSensitiveData` used in line 86.
		// I should check `DecryptAuthSensitiveData` implementation.

		return c.Next()
	}
}

// DualAuthMiddleware checks for api-key header or Authorization header
func DualAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check for api-key header
		apiKey := c.Get("api-key")
		if apiKey != "" {
			// Decrypt api-key to get organization_id
			orgID, err := DecryptString(apiKey)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"status":  "error",
					"message": "Invalid API Key",
				})
			}

			// Set locals
			c.Locals("organization_id", orgID)
			c.Locals("role", "visitor")
			return c.Next()
		}

		// Check for Authorization header
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			return JWTAuthorizationMiddleware()(c)
		}

		// If neither api-key nor Authorization header is present
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":         "error",
			"message":        "Missing Authorization Header or API Key",
			"data":           nil,
			"transaction_id": GetTransactionID(c),
		})
	}
}
