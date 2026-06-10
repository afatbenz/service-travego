package helper

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthTokenClaims represents the JWT claims for authentication token
type AuthTokenClaims struct {
	Fullname         string `json:"fullname"`
	OrganizationName string `json:"organization_name"`
	OrganizationID   string `json:"organization_id"`
	IsAdmin          bool   `json:"is_admin"`
	Email            string `json:"email"`
	Username         string `json:"username"`
	Token            string `json:"token"`
	jwt.RegisteredClaims
}

// GenerateAuthToken generates a JWT token for authentication with configurable expiry
// expiryMinutes: token expiry in minutes (default: 90 if 0 or from AUTH_TOKEN_EXPIRY env)
func GenerateAuthToken(fullname string, organizationName string, organizationID string, isAdmin bool, email string, username string, encToken string, expiryMinutes int) (string, error) {
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
		OrganizationName: organizationName,
		OrganizationID:   organizationID,
		IsAdmin:          isAdmin,
		Email:            email,
		Username:         username,
		Token:            encToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	tokenString, err := jwtToken.SignedString([]byte(secret))
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
	return 60 // Default to 60 minutes
}

// GenerateRefreshToken generates a cryptographically secure random refresh token (hex string).
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// ParseAuthTokenClaims parses a JWT token string and extracts AuthTokenClaims.
// It parses without validating expiry so it can be used for logout even with expired tokens.
func ParseAuthTokenClaims(tokenString string) (*AuthTokenClaims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key-change-in-production"
	}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, err := parser.ParseWithClaims(tokenString, &AuthTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*AuthTokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
