package helper

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from .env file based on environment
func LoadEnv() error {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development" // Default to development
	}

	var envFile string
	switch env {
	case "production":
		envFile = ".env.production"
	case "preproduction", "preprod":
		envFile = ".env.preprod"
	case "development", "dev", "local":
		envFile = ".env.dev"
	default:
		envFile = ".env.dev" // Default to .env.dev
	}

	// Check if .env file exists
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		// If file doesn't exist, try .env as fallback
		if _, err := os.Stat(".env"); err == nil {
			envFile = ".env"
		} else {
			// If no .env file found, return error
			return fmt.Errorf("environment file %s not found", envFile)
		}
	}

	// Load .env file
	err := godotenv.Load(envFile)
	if err != nil {
		return fmt.Errorf("error loading %s file: %w", envFile, err)
	}

	return nil
}

// GetEnv gets environment variable with default value
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
