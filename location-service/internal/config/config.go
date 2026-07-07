package config

import "os"

// Config holds configuration for the Location Service.
type Config struct {
	Port        string
	RedisURL    string
	DatabaseURL string
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8084"),
		RedisURL:    getEnv("REDIS_URL", "redis:6379"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@location-db:5432/locations?sslmode=disable"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
