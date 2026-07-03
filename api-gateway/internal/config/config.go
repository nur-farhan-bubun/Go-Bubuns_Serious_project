package config

import "os"

// Config holds all configuration for the API Gateway.
type Config struct {
	Port               string
	UserServiceURL     string
	MatchServiceURL    string
	ChatServiceURL     string
	LocationServiceURL string
	ClerkJWTIssuer     string
	ClerkJWTKey        string
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		UserServiceURL:     getEnv("USER_SERVICE_URL", "http://user-service:8081"),
		MatchServiceURL:    getEnv("MATCH_SERVICE_URL", "http://match-service:8082"),
		ChatServiceURL:     getEnv("CHAT_SERVICE_URL", "http://chat-service:8083"),
		LocationServiceURL: getEnv("LOCATION_SERVICE_URL", "http://location-service:8084"),
		ClerkJWTIssuer:     getEnv("CLERK_JWT_ISSUER", ""),
		ClerkJWTKey:        getEnv("CLERK_JWT_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
