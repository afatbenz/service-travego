package helper

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthTokenClaims represents the JWT claims for authentication token
type AuthTokenClaims struct {
    Fullname         string `json:"fullname"`
    OrganizationID   string `json:"organization_id"`
    Username         string `json:"username"`
    UserID           string `json:"user_id"`
    OrganizationRole int    `json:"organization_role"`
    Gender           string `json:"gender"`
    IsAdmin          bool   `json:"is_admin"`
    jwt.RegisteredClaims
}

// GenerateAuthToken generates a JWT token for authentication with configurable expiry
// expiryMinutes: token expiry in minutes (default: 90 if 0 or from AUTH_TOKEN_EXPIRY env)
func GenerateAuthToken(fullname, organizationID, username, userID string, organizationRole int, gender string, isAdmin bool, expiryMinutes int) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key-change-in-production" // Default for development
	}

	// Get expiry from parameter, env, or default to 90 minutes
	if expiryMinutes <= 0 {
		expiryMinutes = GetAuthTokenExpiry()
	}

	// Create claims
    claims := AuthTokenClaims{
        Fullname:         fullname,
        OrganizationID:   organizationID,
        Username:         username,
        UserID:           userID,
        OrganizationRole: organizationRole,
        Gender:           gender,
        IsAdmin:          isAdmin,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryMinutes) * time.Minute)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
        },
    }

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// GetAuthTokenExpiry retrieves auth token expiry from environment variable or returns default
func GetAuthTokenExpiry() int {
	if envExpiry := os.Getenv("AUTH_TOKEN_EXPIRY"); envExpiry != "" {
		if expiry, err := strconv.Atoi(envExpiry); err == nil && expiry > 0 {
			return expiry
		}
	}
	return 90 // Default to 90 minutes
}
