package configs

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the application
type Config struct {
	App      AppConfig      `json:"app"`
	Database DatabaseConfig `json:"database"`
	JWT      JWTConfig      `json:"jwt"`
	Email    EmailConfig    `json:"email"`
	Redis    RedisConfig    `json:"redis"`
}

// AppConfig holds application configuration
type AppConfig struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Port         string `json:"port"`
	Environment  string `json:"environment"`
	AllowOrigins string `json:"allow_origins"`
	OTPLength    int    `json:"otp_length"` // Default: 8
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Driver   string `json:"driver"` // postgres, mysql
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"ssl_mode"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret          string `json:"secret"`
	Expiration      int    `json:"expiration"`        // in hours
	AuthTokenExpiry int    `json:"auth_token_expiry"` // in minutes, default: 90
}

// EmailConfig holds email configuration
type EmailConfig struct {
	From     string `json:"from"`
	Password string `json:"password"`
	SMTPHost string `json:"smtp_host"`
	SMTPPort string `json:"smtp_port"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	OTPTTL   int    `json:"otp_ttl"` // OTP TTL in minutes, default: 5
}

// LoadConfig loads configuration from JSON file
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// OverrideWithEnv overrides config values with environment variables if they exist
func OverrideWithEnv(cfg *Config) {
	// App config
	if envName := os.Getenv("APP_NAME"); envName != "" {
		cfg.App.Name = envName
	}
	// Support both PORT (cloud platforms) and APP_PORT
	if envPort := os.Getenv("PORT"); envPort != "" {
		cfg.App.Port = envPort
	} else if envPort := os.Getenv("APP_PORT"); envPort != "" {
		cfg.App.Port = envPort
	}
	if envEnv := os.Getenv("APP_ENV"); envEnv != "" {
		cfg.App.Environment = envEnv
	}
	if envOrigins := os.Getenv("APP_ALLOW_ORIGINS"); envOrigins != "" {
		cfg.App.AllowOrigins = envOrigins
	}
	// OTP Length config - default to 8 if not set
	if envOTPLength := os.Getenv("OTP_LENGTH"); envOTPLength != "" {
		if otpLength, err := strconv.Atoi(envOTPLength); err == nil && otpLength > 0 {
			cfg.App.OTPLength = otpLength
		}
	} else if cfg.App.OTPLength == 0 {
		cfg.App.OTPLength = 8 // Default to 8 digits
	}

	// Database config - prioritize .env.dev over app.json
	if envDriver := os.Getenv("DB_DRIVER"); envDriver != "" {
		cfg.Database.Driver = envDriver
	}
	if envHost := os.Getenv("DB_HOST"); envHost != "" {
		cfg.Database.Host = envHost
	}
	if envPort := os.Getenv("DB_PORT"); envPort != "" {
		cfg.Database.Port = envPort
	}
	// Support both DB_USERNAME and DB_USER
	if envUser := os.Getenv("DB_USERNAME"); envUser != "" {
		cfg.Database.Username = envUser
	} else if envUser := os.Getenv("DB_USER"); envUser != "" {
		cfg.Database.Username = envUser
	}
	if envPass := os.Getenv("DB_PASSWORD"); envPass != "" {
		cfg.Database.Password = envPass
	}
	// Support both DB_DATABASE and DB_NAME
	if envDB := os.Getenv("DB_DATABASE"); envDB != "" {
		cfg.Database.Database = envDB
	} else if envDB := os.Getenv("DB_NAME"); envDB != "" {
		cfg.Database.Database = envDB
	}
	if envSSL := os.Getenv("DB_SSL_MODE"); envSSL != "" {
		cfg.Database.SSLMode = envSSL
	}

	// Redis config - support both REDIS_HOST (with port) and separate REDIS_HOST/REDIS_PORT
	if envHost := os.Getenv("REDIS_HOST"); envHost != "" {
		// If REDIS_HOST contains :, split it (format: host:port)
		if strings.Contains(envHost, ":") {
			parts := strings.Split(envHost, ":")
			cfg.Redis.Host = parts[0]
			if len(parts) > 1 {
				cfg.Redis.Port = parts[1]
			}
		} else {
			cfg.Redis.Host = envHost
		}
	}
	if envPort := os.Getenv("REDIS_PORT"); envPort != "" {
		cfg.Redis.Port = envPort
	}
	if envPass := os.Getenv("REDIS_PASSWORD"); envPass != "" {
		cfg.Redis.Password = envPass
	}
	if envDB := os.Getenv("REDIS_DB"); envDB != "" {
		if db, err := strconv.Atoi(envDB); err == nil {
			cfg.Redis.DB = db
		}
	}
	// OTP TTL config - default to 5 minutes if not set
	if envOTPTTL := os.Getenv("OTP_TTL"); envOTPTTL != "" {
		if otpTTL, err := strconv.Atoi(envOTPTTL); err == nil && otpTTL > 0 {
			cfg.Redis.OTPTTL = otpTTL
		}
	} else if cfg.Redis.OTPTTL == 0 {
		cfg.Redis.OTPTTL = 5 // Default to 5 minutes
	}

	// JWT config
	if envSecret := os.Getenv("JWT_SECRET"); envSecret != "" {
		cfg.JWT.Secret = envSecret
	}
	if envExp := os.Getenv("JWT_EXPIRATION"); envExp != "" {
		if exp, err := strconv.Atoi(envExp); err == nil {
			cfg.JWT.Expiration = exp
		}
	}
	// Auth Token Expiry config - default to 90 minutes if not set
	if envAuthExpiry := os.Getenv("AUTH_TOKEN_EXPIRY"); envAuthExpiry != "" {
		if authExpiry, err := strconv.Atoi(envAuthExpiry); err == nil && authExpiry > 0 {
			cfg.JWT.AuthTokenExpiry = authExpiry
		}
	} else if cfg.JWT.AuthTokenExpiry == 0 {
		cfg.JWT.AuthTokenExpiry = 90 // Default to 90 minutes
	}

	// Email config - must be set from environment variables
	if envFrom := os.Getenv("EMAIL_FROM"); envFrom != "" {
		cfg.Email.From = envFrom
	}
	if envPass := os.Getenv("EMAIL_PASSWORD"); envPass != "" {
		cfg.Email.Password = envPass
	}
	if envHost := os.Getenv("EMAIL_SMTP_HOST"); envHost != "" {
		cfg.Email.SMTPHost = envHost
	}
	if envPort := os.Getenv("EMAIL_SMTP_PORT"); envPort != "" {
		cfg.Email.SMTPPort = envPort
	}
}

// ValidateEmailConfig validates that email configuration is properly set
func ValidateEmailConfig(cfg *EmailConfig) error {
	if cfg.From == "" {
		return fmt.Errorf("EMAIL_FROM environment variable is required")
	}
	if cfg.Password == "" {
		return fmt.Errorf("EMAIL_PASSWORD environment variable is required")
	}
	if cfg.SMTPHost == "" {
		return fmt.Errorf("EMAIL_SMTP_HOST environment variable is required")
	}
	if cfg.SMTPPort == "" {
		return fmt.Errorf("EMAIL_SMTP_PORT environment variable is required")
	}
	return nil
}
