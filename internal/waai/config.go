package waai

import (
	"os"
)

// Config holds WhatsApp AI Assistant configuration
type Config struct {
	WagyDeviceID    string
	WagyToken       string
	WagyWebhookSecret string
	AnthropicAPIKey string
	RedisURL        string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		WagyDeviceID:      os.Getenv("WAGY_DEVICE_ID"),
		WagyToken:         os.Getenv("WAGY_TOKEN"),
		WagyWebhookSecret: os.Getenv("WAGY_WEBHOOK_SECRET"),
		AnthropicAPIKey:   os.Getenv("ANTHROPIC_API_KEY"),
		RedisURL:          os.Getenv("REDIS_URL"),
	}
}

// Validate checks if required configuration values are set
func (c *Config) Validate() error {
	requiredFields := map[string]string{
		"WAGY_DEVICE_ID":         c.WagyDeviceID,
		"WAGY_TOKEN":             c.WagyToken,
		"WAGY_WEBHOOK_SECRET":    c.WagyWebhookSecret,
		"ANTHROPIC_API_KEY":      c.AnthropicAPIKey,
		"REDIS_URL":              c.RedisURL,
	}

	for name, value := range requiredFields {
		if value == "" {
			return NewConfigError(name + " is not set")
		}
	}
	return nil
}

// ConfigError represents a configuration error
type ConfigError struct {
	Message string
}

func NewConfigError(msg string) *ConfigError {
	return &ConfigError{Message: msg}
}

func (e *ConfigError) Error() string {
	return "WAAI Config Error: " + e.Message
}
