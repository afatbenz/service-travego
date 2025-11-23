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
