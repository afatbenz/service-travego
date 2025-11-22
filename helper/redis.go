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
	return client, nil
}

// SetOTP stores OTP in Redis with expiration (5 minutes)
func SetOTP(key, otp string) error {
	if redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return redisClient.Set(ctx, fmt.Sprintf("otp:%s", key), otp, 5*time.Minute).Err()
}

// GetOTP retrieves OTP from Redis
func GetOTP(key string) (string, error) {
	if redisClient == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	return redisClient.Get(ctx, fmt.Sprintf("otp:%s", key)).Result()
}

// DeleteOTP removes OTP from Redis
func DeleteOTP(key string) error {
	if redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return redisClient.Del(ctx, fmt.Sprintf("otp:%s", key)).Err()
}
