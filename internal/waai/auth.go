package waai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// TenantAuthData represents tenant authentication data stored in Redis
type TenantAuthData struct {
	OrganizationID   string `json:"organization_id"`
	UserID           string `json:"user_id"`
	RoleName         string `json:"role_name"`
	FullName         string `json:"full_name,omitempty"`
	OrganizationName string `json:"organization_name,omitempty"`
	Phone            string `json:"phone,omitempty"`
	StoredAt         string `json:"stored_at,omitempty"`
}

// AuthManager handles tenant authentication data storage in Redis
type AuthManager struct {
	client *redis.Client
}

// NewAuthManager creates a new auth manager
func NewAuthManager(rdb *redis.Client) *AuthManager {
	return &AuthManager{
		client: rdb,
	}
}

// GetAuthKey returns the Redis key for tenant auth data
func GetAuthKey(phone string) string {
	phone = normalizeAuthPhone(phone)
	return fmt.Sprintf("%s-whatsapp-authorized", phone)
}

// SaveTenantAuth saves tenant authentication data to Redis
// Data includes fullname, organization_id, organization_name, role
// TTL is set to 24 hours
func (am *AuthManager) SaveTenantAuth(ctx context.Context, phone string, tenant *TenantInfo) error {
	if am == nil || am.client == nil {
		return nil
	}
	if tenant == nil {
		return fmt.Errorf("tenant info is nil")
	}

	fullName := tenant.FullName
	if fullName == "" {
		fullName = tenant.Name
	}

	authData := TenantAuthData{
		OrganizationID:   tenant.OrganizationID,
		UserID:           tenant.UserID,
		RoleName:         tenant.RoleName,
		FullName:         fullName,
		OrganizationName: tenant.OrganizationName,
		Phone:            tenant.Phone,
		StoredAt:         time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(authData)
	if err != nil {
		return fmt.Errorf("failed to marshal auth data: %w", err)
	}

	key := GetAuthKey(phone)
	err = am.client.Set(ctx, key, data, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("redis error: %w", err)
	}

	return nil
}

// GetTenantAuth retrieves tenant authentication data from Redis
func (am *AuthManager) GetTenantAuth(ctx context.Context, phone string) (*TenantAuthData, error) {
	if am == nil || am.client == nil {
		return nil, nil
	}
	key := GetAuthKey(phone)

	val, err := am.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}

	var authData TenantAuthData
	err = json.Unmarshal([]byte(val), &authData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal auth data: %w", err)
	}

	return &authData, nil
}

// ClearTenantAuth removes tenant authentication data from Redis
func (am *AuthManager) ClearTenantAuth(ctx context.Context, phone string) error {
	if am == nil || am.client == nil {
		return nil
	}
	key := GetAuthKey(phone)
	err := am.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis error: %w", err)
	}
	return nil
}

// RefreshTenantAuthTTL extends the TTL of tenant auth data (sliding expiration)
func (am *AuthManager) RefreshTenantAuthTTL(ctx context.Context, phone string) error {
	if am == nil || am.client == nil {
		return nil
	}
	key := GetAuthKey(phone)
	err := am.client.Expire(ctx, key, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("redis error: %w", err)
	}
	return nil
}

func normalizeAuthPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.TrimSuffix(phone, "@s.whatsapp.net")
	phone = strings.TrimPrefix(phone, "+")
	return phone
}
