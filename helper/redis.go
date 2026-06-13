package helper

import (
	"context"
	"fmt"
	"service-travego/configs"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client
var ctx = context.Background()
var otpTTL time.Duration // OTP TTL duration

// InitRedis initializes Redis connection
func InitRedis(cfg *configs.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	redisClient = client

	// Set OTP TTL from config (in minutes), default to 5 minutes
	ttlMinutes := cfg.OTPTTL
	if ttlMinutes <= 0 {
		ttlMinutes = 5 // Default to 5 minutes
	}
	otpTTL = time.Duration(ttlMinutes) * time.Minute

	return client, nil
}

// GetRedisClient returns the initialized Redis client
func GetRedisClient() *redis.Client {
	return redisClient
}

// SetOTP stores OTP in Redis with expiration (configured via OTP_TTL env or default 5 minutes)
func SetOTP(key, otp string) error {
	if redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	// Use configured TTL, or default to 5 minutes if not set
	ttl := otpTTL
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	return redisClient.Set(ctx, fmt.Sprintf("otp:%s", key), otp, ttl).Err()
}

func SetOTPWithTTL(key, otp string, ttl time.Duration) error {
	if redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return redisClient.Set(ctx, fmt.Sprintf("otp:%s", key), otp, ttl).Err()
}

// GetOTP retrieves OTP from Redis
func GetOTP(key string) (string, error) {
	if redisClient == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	result, err := redisClient.Get(ctx, fmt.Sprintf("otp:%s", key)).Result()
	if err != nil {
		// Return error as string for easier checking
		return "", err
	}
	return result, nil
}

// DeleteOTP removes OTP from Redis
func DeleteOTP(key string) error {
	if redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return redisClient.Del(ctx, fmt.Sprintf("otp:%s", key)).Err()
}

// --- Refresh Token helpers ---

const refreshTokenPrefix = "refresh:"

// SetRefreshToken stores a refresh token in Redis with expiration.
// Key: "refresh:{userID}", Value: the refresh token string.
// Each call resets the TTL (sliding expiration for inactivity invalidation).
func SetRefreshToken(userID, token string, ttl time.Duration) error {
	if redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour // default 24 hours
	}
	return redisClient.Set(ctx, refreshTokenPrefix+userID, token, ttl).Err()
}

// GetRefreshToken retrieves a refresh token from Redis by userID.
func GetRefreshToken(userID string) (string, error) {
	if redisClient == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	result, err := redisClient.Get(ctx, refreshTokenPrefix+userID).Result()
	if err != nil {
		return "", err
	}
	return result, nil
}

// DeleteRefreshToken removes a refresh token from Redis by userID.
func DeleteRefreshToken(userID string) error {
	if redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return redisClient.Del(ctx, refreshTokenPrefix+userID).Err()
}

// ExtendRefreshTokenTTL resets the TTL of an existing refresh token (sliding expiration).
func ExtendRefreshTokenTTL(userID string, ttl time.Duration) error {
	if redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return redisClient.Expire(ctx, refreshTokenPrefix+userID, ttl).Err()
}

const refreshTokenReversePrefix = "refresh_rev:"

// SetRefreshTokenReverse stores a reverse mapping from refresh token to userID.
// This allows looking up the userID when only the refresh token is known.
func SetRefreshTokenReverse(token, userID string, ttl time.Duration) error {
	if redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return redisClient.Set(ctx, refreshTokenReversePrefix+token, userID, ttl).Err()
}

// GetRefreshTokenUserID retrieves the userID associated with a refresh token.
func GetRefreshTokenUserID(token string) (string, error) {
	if redisClient == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	result, err := redisClient.Get(ctx, refreshTokenReversePrefix+token).Result()
	if err != nil {
		return "", err
	}
	return result, nil
}

// DeleteRefreshTokenReverse removes the reverse mapping for a refresh token.
func DeleteRefreshTokenReverse(token string) error {
	if redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return redisClient.Del(ctx, refreshTokenReversePrefix+token).Err()
}
