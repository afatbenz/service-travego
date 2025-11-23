package helper

import (
	"os"
	"strings"
)

// GetAssetURL returns the full URL for an asset path
// If path starts with /assets, prepends APP_HOST environment variable
// If APP_HOST is not set, returns the path as is
func GetAssetURL(path string) string {
	if path == "" {
		return path
	}

	// Only process paths that start with /assets
	if !strings.HasPrefix(path, "/assets") {
		return path
	}

	// Get APP_HOST from environment
	appHost := os.Getenv("APP_HOST")
	if appHost == "" {
		// If APP_HOST is not set, return path as is
		return path
	}

	// Remove trailing slash from APP_HOST if present
	appHost = strings.TrimSuffix(appHost, "/")

	// Return full URL
	return appHost + path
}
